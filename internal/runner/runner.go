package runner

import (
	"bufio"
	"io"
	"runtime"
	"sync"

	"sample_account/internal/field"
	"sample_account/internal/gen"
	"sample_account/internal/repo"
)

const (
	// outBufSize is the bufio.Writer buffer in bytes (1 MiB) — large enough
	// to amortize syscall overhead on large counts without ballooning memory.
	outBufSize = 1 << 20

	// serialThreshold below which we skip goroutine setup entirely. Below
	// this row count, goroutine launch + WaitGroup overhead dominates.
	serialThreshold = 1000

	// subChunkRows is how many rows a parallel worker generates before
	// shipping the buffer to the writer. Larger sub-chunks amortize channel
	// overhead and let workers stay busy while the writer drains earlier
	// workers in row order.
	subChunkRows = 65536

	// subChunkCap pre-sizes each sub-chunk buffer. 256 bytes/row is a
	// generous upper bound for our 17-column max output.
	subChunkCap = subChunkRows * 256

	// outboxDepth is how many completed sub-chunks each worker may queue
	// ahead of the writer. Bigger depth = better pipelining (later workers
	// can run ahead while writer drains earlier ones) at the cost of memory.
	// Memory bound: workers * (outboxDepth + 1) * subChunkCap.
	// At workers=16, depth=4: 16 * 5 * 16 MiB = 1.25 GiB peak.
	outboxDepth = 4
)

// Deps holds the heavyweight read-only data needed to generate rows.
// Generators are derived from these on demand. PersonGen is lightweight
// enough to construct per-worker so they don't share mutable state.
type Deps struct {
	Persons     []repo.PersonRecord
	Prefectures *repo.PrefectureRepo
	Ages        *repo.AgeRepo

	// AddressGen is read-only after construction (it just wraps Prefectures)
	// so workers can share a single instance.
	AddressGen *gen.AddressGen
}

// Run produces `count` rows of the selected fields and writes them to w
// using the auto-tuned worker count (NumCPU above serialThreshold,
// single-threaded below).
//
// masterSeed seeds an independent per-row PCG via splitmix64(master ^ row),
// which is what makes the parallel path produce the same output as serial.
func Run(w io.Writer, count int, fields []field.Field, deps Deps, masterSeed uint64) error {
	return RunWithJobs(w, count, fields, deps, masterSeed, 0)
}

// RunWithJobs is Run with an explicit worker count.
//
//   jobs == 0 → auto: NumCPU workers when count >= serialThreshold, else serial.
//   jobs == 1 → forced single-threaded path.
//   jobs >= 2 → forced parallel path with exactly N workers.
//
// Output is byte-identical regardless of worker count because each row's
// RNG is derived from (masterSeed, rowIndex), not from a shared stream.
func RunWithJobs(w io.Writer, count int, fields []field.Field, deps Deps, masterSeed uint64, jobs int) error {
	if count <= 0 {
		return nil
	}
	switch {
	case jobs == 1:
		return runSerial(w, count, fields, deps, masterSeed)
	case jobs >= 2:
		return runParallel(w, count, fields, deps, masterSeed, jobs)
	default:
		// auto
		if count < serialThreshold {
			return runSerial(w, count, fields, deps, masterSeed)
		}
		return runParallel(w, count, fields, deps, masterSeed, runtime.NumCPU())
	}
}

func runSerial(w io.Writer, count int, fields []field.Field, deps Deps, masterSeed uint64) error {
	bw := bufio.NewWriterSize(w, outBufSize)
	defer bw.Flush()

	personGen := gen.NewPersonGen(deps.Persons)
	ageGen := gen.NewAgeGen(deps.Ages)
	nowUnix := gen.CurrentTime().Unix()

	// Reuse a single byte slice across rows; each row truncates back to 0.
	buf := make([]byte, 0, 256)
	for row := 0; row < count; row++ {
		buf = appendRow(buf[:0], row, fields, personGen, deps.AddressGen, ageGen, gen.NewRowRngWithNow(masterSeed, uint64(row), nowUnix))
		if _, err := bw.Write(buf); err != nil {
			return err
		}
	}
	return nil
}

// runParallel generates `count` rows using `workers` goroutines and
// streams them to w in row order via a bounded sub-chunk pipeline.
//
// Per-worker memory is capped at subChunkCap * (outboxDepth + 1) bytes —
// independent of `count` — so billion-row generations stay within tens of
// MB of resident memory.
//
// Pipeline shape (per worker):
//
//	worker(wi): row range [start, end)  --→ outbox[wi] (chan, depth=2) --→ writer
//
// The writer drains outbox[0] to completion, then outbox[1], and so on,
// preserving strictly ascending row order. A sync.Pool recycles byte
// slices so we avoid 100k+ allocations on huge counts.
func runParallel(w io.Writer, count int, fields []field.Field, deps Deps, masterSeed uint64, workers int) error {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if workers > count {
		workers = count
	}

	chunk := count / workers
	nowUnix := gen.CurrentTime().Unix()
	personGen := gen.NewPersonGen(deps.Persons)
	ageGen := gen.NewAgeGen(deps.Ages)

	// Pool of recyclable []byte buffers. Workers grab one per sub-chunk;
	// the writer puts it back after writing. Capped capacity keeps memory
	// bounded even if a buffer briefly grows past subChunkCap.
	pool := &sync.Pool{
		New: func() any {
			b := make([]byte, 0, subChunkCap)
			return &b
		},
	}

	outboxes := make([]chan *[]byte, workers)
	for i := range outboxes {
		outboxes[i] = make(chan *[]byte, outboxDepth)
	}

	var wg sync.WaitGroup
	for wi := 0; wi < workers; wi++ {
		wi := wi
		start := wi * chunk
		end := start + chunk
		if wi == workers-1 {
			end = count
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(outboxes[wi])
			scratch := make([]byte, 0, 256)
			row := start
			for row < end {
				bufPtr := pool.Get().(*[]byte)
				buf := (*bufPtr)[:0]
				limit := row + subChunkRows
				if limit > end {
					limit = end
				}
				for ; row < limit; row++ {
					scratch = appendRow(scratch[:0], row, fields, personGen, deps.AddressGen, ageGen,
						gen.NewRowRngWithNow(masterSeed, uint64(row), nowUnix))
					buf = append(buf, scratch...)
				}
				*bufPtr = buf
				outboxes[wi] <- bufPtr
			}
		}()
	}

	bw := bufio.NewWriterSize(w, outBufSize)
	var writeErr error
	for wi := 0; wi < workers; wi++ {
		for bufPtr := range outboxes[wi] {
			if writeErr == nil {
				if _, err := bw.Write(*bufPtr); err != nil {
					writeErr = err
				}
			}
			// Recycle even on error so producers can drain and exit.
			*bufPtr = (*bufPtr)[:0]
			pool.Put(bufPtr)
		}
	}
	wg.Wait()
	if writeErr != nil {
		return writeErr
	}
	return bw.Flush()
}

// appendRow writes one CSV row (no trailing comma, with terminating newline)
// to buf and returns the resulting slice. Per-row state (RowContext) is
// derived from rng so the (masterSeed, row) pair fully determines output.
func appendRow(
	buf []byte,
	row int,
	fields []field.Field,
	person *gen.PersonGen,
	address *gen.AddressGen,
	age *gen.AgeGen,
	rng *gen.Rng,
) []byte {
	deps := field.Deps{Person: person, Address: address, Age: age, Rng: rng}
	ctx := field.RowContext{
		Row:   row,
		First: rng.Next(),
		Last:  rng.Next(),
		Pref:  address.WeightedPrefectureIndex(rng.Next()),
		Ward:  rng.Next(),
		City:  rng.Next(),
		Age:   rng.Next(),
	}
	for j, f := range fields {
		if j > 0 {
			buf = append(buf, ',')
		}
		buf = f.Emit(buf, ctx, deps)
	}
	return append(buf, '\n')
}
