#!/usr/bin/env bash
# Compare wall-clock time of the Go port vs the C++ -O2 binary across a
# range of row counts. Both runs use identical seed/now/TZ for fairness.
#
# Usage: ./scripts/bench-compare.sh [GO_BIN] [CPP_BIN]
set -euo pipefail

GO_BIN="${1:-$(pwd)/sample_account}"
CPP_BIN="${2:-/Volumes/dev/src/cpp/work/sample_account/sample_account}"
CPP_CWD="$(dirname "$CPP_BIN")"
ARGS="-ilfmatpwcgbdorynq"

if [[ ! -x "$GO_BIN" ]]; then
  echo "FAIL: $GO_BIN not built. Run 'make build' first." >&2
  exit 1
fi
if [[ ! -x "$CPP_BIN" ]]; then
  echo "FAIL: $CPP_BIN not built. Build the C++ reference with 'make CXXFLAGS_OPT=-O2'." >&2
  exit 1
fi

export SAMPLE_ACCOUNT_SEED=42
export SAMPLE_ACCOUNT_NOW=1700000000
export TZ=Asia/Tokyo

# Time a command N times (best of 3) and print wall seconds.
best_of_3() {
  local cmd="$1"
  local best=999999
  for _ in 1 2 3; do
    local start end elapsed
    start=$(python3 -c 'import time; print(time.time())')
    eval "$cmd" > /dev/null
    end=$(python3 -c 'import time; print(time.time())')
    elapsed=$(python3 -c "print($end - $start)")
    if (( $(python3 -c "print(int($elapsed < $best))") )); then
      best=$elapsed
    fi
  done
  echo "$best"
}

printf '%-12s | %12s | %12s | %8s\n' 'count' 'C++ (-O2) s' 'Go s' 'speedup'
printf '%s\n' '--------------------------------------------------------'
for count in 100 1000 10000 100000 1000000; do
  cpp_t=$(best_of_3 "(cd '$CPP_CWD' && '$CPP_BIN' $ARGS $count)")
  go_t=$(best_of_3 "'$GO_BIN' $ARGS $count")
  speedup=$(python3 -c "print(f'{$cpp_t / $go_t:.2f}x')")
  printf '%-12s | %12.4f | %12.4f | %8s\n' "$count" "$cpp_t" "$go_t" "$speedup"
done
