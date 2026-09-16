#!/bin/sh
set -eu

# build-skills-zip.sh — Package the skills/ tree into a distributable zip.
#
# Usage: ./scripts/release/build-skills-zip.sh <output.zip>
#
# The zip is flattened (SKILL.md at the root) and bundles the CLI reference
# doc so links resolve after installation — the repo-relative
# ../../docs/reference.md target is not shipped with the skill package.

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"
OUTPUT="${1:?usage: build-skills-zip.sh <output.zip>}"

export LANG=C
export LC_ALL=C
export LC_CTYPE=C

# Resolve to an absolute path: the staging step cd's elsewhere before zipping.
case "$OUTPUT" in
  /*) ;;
  *)  OUTPUT="$ROOT/${OUTPUT#./}" ;;
esac

mkdir -p "$(dirname -- "$OUTPUT")"
rm -f "$OUTPUT"

staging="$(mktemp -d)"
trap 'rm -rf "$staging"' EXIT

cp -R "$ROOT/skills/." "$staging/"
cp "$ROOT/docs/reference.md" "$staging/references/cli-reference.md"
sed -i.bak 's|(\.\./\.\./docs/reference\.md)|(./cli-reference.md)|g' "$staging/references/error-codes.md"
rm -f "$staging/references/error-codes.md.bak"

(
  cd "$staging"
  env -u LC_ALL -u LC_CTYPE LANG=C LC_ALL=C LC_CTYPE=C zip -qr "$OUTPUT" .
)
