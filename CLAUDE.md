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

## Release Flow

Releases are fully automated via [release-please](https://github.com/googleapis/release-please) +
[GoReleaser](https://goreleaser.com/). Three workflows live under `.github/workflows/`:

| Workflow | Trigger | Responsibility |
|---|---|---|
| `ci.yml` | every push to `main`, every PR to `main` | go vet, golangci-lint, `go test -race`, snapshot tests |
| `release-please.yml` (job 1) | push to `main` | Maintain a Release PR that bumps version + CHANGELOG from Conventional Commits |
| `release-please.yml` (job 2) | when the Release PR is merged and a tag is created | GoReleaser builds linux/amd64, linux/arm64, darwin/arm64, windows/amd64 binaries and attaches them to the Release |

### Adding a release-worthy change

1. Use a Conventional Commits prefix on every commit/PR title (`feat:`, `fix:`, `perf:`, etc.)
2. Land it on `main` via PR
3. release-please opens or updates a "chore: release X.Y.Z" PR with the rolled-up CHANGELOG
4. Merge that PR — the workflow tags `vX.Y.Z`, creates a GitHub Release, and triggers GoReleaser
5. Binaries appear under https://github.com/sho7650/sample_account_golang/releases/tag/vX.Y.Z

### Version source of truth

`internal/version/version.go` carries `const Version = "X.Y.Z" // x-release-please-version`.
release-please updates that line on every release; **do not edit it by hand** —
the trailing comment is the marker the automation matches on.

### Repo settings prerequisites

- Settings → Actions → General → Workflow permissions: "Read and write permissions"
- Settings → Actions → General → "Allow GitHub Actions to create and approve pull requests": **enabled**

Without these, release-please cannot open the Release PR with the default `GITHUB_TOKEN`.

### Local validation

```sh
nix shell nixpkgs#goreleaser -c goreleaser check          # validate .goreleaser.yaml
nix shell nixpkgs#goreleaser -c goreleaser release --snapshot --clean --skip=publish  # dry-run cross-build
```
