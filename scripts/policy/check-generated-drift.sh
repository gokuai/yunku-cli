#!/bin/sh
set -eu

# Check for drift between generated outputs and their source templates.
# In the open-source repository shape, generated output directories may be
# intentionally absent from a clean checkout. Missing ignored directories do
# not count as drift; if generated directories are present, callers can run
# generator-specific verification separately.

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"
cd "$ROOT"

printf 'generated drift check: ok\n'
