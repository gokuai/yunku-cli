#!/bin/sh
set -eu

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"

run() {
  printf '+ %s\n' "$*"
  "$@"
}

cd "$ROOT"

run ./scripts/dev/lint.sh
run go test ./...
COVERAGE_PROFILE="$(mktemp "${TMPDIR:-/tmp}/ykc-coverage.XXXXXX")"
run ./scripts/dev/coverage.sh "$COVERAGE_PROFILE"
rm -f "$COVERAGE_PROFILE"
run git diff --check
