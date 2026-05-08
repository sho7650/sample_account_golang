# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`sample_account` is a Go port of the C++17 [sample_account](../cpp/work/sample_account/)
CLI: a synthetic Japanese personal-account CSV generator. Output columns
are selected by short/long flags in the order they appear; data is
embedded into the binary via `go:embed`.

## Build & Run

The Go toolchain lives entirely inside Nix.

```sh
nix develop                    # enter dev shell (go 1.26, golangci-lint, gofumpt, delve)
make build                     # → ./sample_account (statically linked, ~7 MB)
make test                      # go test ./... -race
make test-cover                # coverage report (currently >85%)
make snapshot                  # ./tests/expected/ golden-file diff
make bench                     # benchmarks
make bench-compare             # head-to-head vs C++ -O2 binary
make lint                      # golangci-lint
```

The binary can be run from any directory because all CSV data is embedded
(no `file not found: data/...` like the C++ version).

```sh
./sample_account --help
./sample_account [OPTIONS] [COUNT]
```

## Architecture

Multi-package layout, dependencies flow downward:

```
cmd/sample_account/main.go              entry point
            │
            ▼
internal/cli                            argv parser (preserves flag order)
internal/runner                         parallel runner (NumCPU chunks)
            │
            ▼
internal/field                          17 Field implementations + Registry
            │
            ▼
internal/gen                            PersonGen / AddressGen / AgeGen / Rng (PCG)
            │
            ▼
internal/repo                           //go:embed CSV → structs
```

### Adding a new column

1. Add a struct (zero-sized) to `internal/field/fields.go` implementing the
   `Field` interface (`ShortFlag`, `LongName`, `Description`, `Emit`).
2. Register it in `DefaultRegistry()` in `internal/field/registry.go`.
3. That's it. The CLI parser, `--help` output, and short-option string all
   derive from the registry — no other edits needed.

### Per-row state

The runner builds `field.RowContext` per row from a row-local RNG seeded
with `splitmix64(masterSeed XOR rowIndex)`. Multiple fields can share
context values to stay consistent (e.g. `First` drives both first-name
and email's local-part).

### Repository contract

Read-only after construction. CSVs are embedded under `internal/repo/data/`.
Cumulative distribution (for population-weighted prefecture lookup) and
per-prefecture address offsets are computed at load time. Substitute
data sources by replacing the embed FS or building a repo manually
from any `io.Reader`.

### Parallel generation

The runner partitions [0, count) into `runtime.NumCPU()` row ranges and
hands each a `bytes.Buffer`. Workers fill their buffer independently
(no shared mutable state — the row RNG is built per row from the master
seed) and the main goroutine flushes buffers in worker order, preserving
strictly ascending row numbers.

For `count < 1000` the runner falls back to a single-threaded path to
avoid goroutine launch overhead.

## Determinism / Test Hooks

- `SAMPLE_ACCOUNT_SEED` — pin the master RNG seed (replaces wall-clock
  seeding). Each row derives an independent sub-seed from this master
  via splitmix64.
- `SAMPLE_ACCOUNT_NOW` — pin "current time" (Unix epoch seconds) used by
  the date column and `birthyear` column.
- `TZ=Asia/Tokyo` — required for stable `birthyear` / `date` across CI
  environments running in UTC.

Snapshots in `tests/expected/` are generated with all three pinned:

```sh
SAMPLE_ACCOUNT_SEED=42 SAMPLE_ACCOUNT_NOW=1700000000 TZ=Asia/Tokyo \
  ./sample_account ... > tests/expected/<name>.csv
```

## Notable Constraints / Gotchas

- The Go output **does not match** the C++ output byte-for-byte. The
  C `rand()` LCG and Go's PCG produce different sequences. Snapshots
  in `tests/expected/` are independent from the C++ project's.
- `internal/cli` has a custom argv scanner because Go's standard `flag`
  package does not preserve flag order, which is required for column-order
  parity.
- `--telehpne` typo alias is intentionally preserved as `--telephone`
  for compatibility with downstream scripts.
- All RNG calls share `splitmix64` for sub-seed derivation. Changing the
  derivation breaks reproducibility.
- The runner's `serialThreshold = 1000` is empirically tuned to where
  goroutine overhead exceeds parallelism gains. Profile before changing.

## Performance

Apple M4 Max, 16 cores, all 17 columns:

| count | C++ -O2 | Go (parallel) | speedup |
|-------|---------|---------------|---------|
| 100 | 0.040s | 0.030s | 1.31x |
| 10,000 | 0.068s | 0.044s | 1.53x |
| 1,000,000 | 1.900s | 0.087s | **22x** |

Pure generation (excluding process startup) at 1M rows: 42 ms in Go vs
1.9 s in C++ → **45x**. Speedup grows with row count because the parallel
section dominates over fixed startup.

Major optimizations vs. naive port:
1. PersonGen pre-joins `kanji,kana` strings at construction (no per-row
   string concatenation).
2. `RollDate` and `BirthYear` cache `nowUnix` / `nowYear` at construction
   (eliminates `os.Getenv` and `time.Now` from the hot loop).
3. `bufio.Writer` with 1 MiB buffer + `strconv.Append*` directly into
   `[]byte` (no `fmt.Fprintf` reflection).
4. Cumulative population distribution + `sort.Search` for prefecture
   weighting (linear → binary).
5. Pre-computed per-prefecture address offsets (linear → O(1)).

## Testing

Test files live alongside the code they test (`*_test.go`). Snapshot
tests live under `tests/snapshot_test.go` behind `//go:build snapshot`
to keep the default test run fast.

```sh
go test ./... -race                      # unit + integration, all packages
TZ=Asia/Tokyo go test -tags=snapshot ./tests/...   # E2E golden-file diff
go test -bench=. -benchmem ./...         # benchmarks
```
