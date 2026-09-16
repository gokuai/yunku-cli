#!/bin/sh
set -eu

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"

cd "$ROOT"
go test ./test/integration/recovery/...
