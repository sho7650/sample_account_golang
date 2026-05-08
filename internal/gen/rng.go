package gen

import (
	"math/rand/v2"
	"os"
	"strconv"
	"time"
)

// Rng wraps math/rand/v2 PCG with a fixed Next() that returns a non-negative
// 31-bit int (mirrors C rand()'s [0, RAND_MAX] convention used by the C++
// reference, but with PCG's far better statistical quality).
//
// nowUnix is captured once and reused by RollDate so the hot path doesn't
// re-read environment variables per row.
type Rng struct {
	pcg     *rand.PCG
	nowUnix int64
}

const goldenGamma uint64 = 0x9E3779B97F4A7C15

// splitmix64 is the standard fast 64-bit avalanche hash used to derive
// independent sub-seeds from a single master seed. Stateless, deterministic.
func splitmix64(x uint64) uint64 {
	x += goldenGamma
	x = (x ^ (x >> 30)) * 0xBF58476D1CE4E5B9
	x = (x ^ (x >> 27)) * 0x94D049BB133111EB
	return x ^ (x >> 31)
}

// NewMasterRng creates a top-level Rng seeded from the supplied 64-bit value.
// Two PCG seeds are derived deterministically.
func NewMasterRng(seed uint64) *Rng {
	s1 := splitmix64(seed)
	s2 := splitmix64(s1 ^ goldenGamma)
	return &Rng{pcg: rand.NewPCG(s1, s2), nowUnix: CurrentTime().Unix()}
}

// NewRowRng builds a per-row RNG whose state is fully determined by the
// (master, row) pair. This makes each row independent, which is the
// foundation of the parallel runner.
func NewRowRng(master, row uint64) *Rng {
	s1 := splitmix64(master ^ row)
	s2 := splitmix64(s1 + goldenGamma)
	return &Rng{pcg: rand.NewPCG(s1, s2), nowUnix: CurrentTime().Unix()}
}

// NewRowRngWithNow is the parallel-friendly constructor — the caller passes
// a pre-cached "now" value so the hot loop avoids per-row env lookups.
func NewRowRngWithNow(master, row uint64, nowUnix int64) *Rng {
	s1 := splitmix64(master ^ row)
	s2 := splitmix64(s1 + goldenGamma)
	return &Rng{pcg: rand.NewPCG(s1, s2), nowUnix: nowUnix}
}

// Next returns a non-negative 31-bit int. We slice off the high bit so
// downstream `n % k` computations behave like C's rand().
func (r *Rng) Next() int {
	return int(r.pcg.Uint64() >> 33)
}

// RollDate samples a random calendar date in [epoch, nowUnix).
// The components are derived from a single random instant for self-consistency.
// Hot path: avoids any os.Getenv or time.Now call.
func (r *Rng) RollDate() (year, month, day int) {
	if r.nowUnix <= 0 {
		return 1970, 1, 1
	}
	pick := time.Unix(int64(r.pcg.Uint64()%uint64(r.nowUnix)), 0).In(time.Local)
	return pick.Year(), int(pick.Month()), pick.Day()
}

// CurrentTime returns the moment the process treats as "now". Honors the
// SAMPLE_ACCOUNT_NOW env var (Unix epoch seconds) when set; otherwise
// returns the wall clock. Used by RollDate and AgeGen.BirthYear.
func CurrentTime() time.Time {
	if v := os.Getenv("SAMPLE_ACCOUNT_NOW"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return time.Unix(n, 0).In(time.Local)
		}
	}
	return time.Now()
}

// MasterSeed returns the seed passed via SAMPLE_ACCOUNT_SEED, falling back
// to a wall-clock-derived value when unset.
func MasterSeed() uint64 {
	if v := os.Getenv("SAMPLE_ACCOUNT_SEED"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			return n
		}
	}
	return uint64(time.Now().UnixNano())
}
