#!/bin/bash
set -e

VERSION="v1.0.6"

echo "==> Running GoReleaser snapshot..."
goreleaser release --snapshot --clean

echo "==> Running post-goreleaser.sh..."
YKC_PACKAGE_VERSION=$VERSION ./scripts/release/post-goreleaser.sh

echo "==> Verifying artifacts..."
echo "Binary version:"
./dist/ykc-darwin-arm64/ykc version 2>/dev/null || ./dist/ykc-linux-amd64/ykc version

echo "npm package.json:"
cat dist/npm/yunku-cli/package.json | grep '"version"'

echo "ykc-skills.zip:"
ls -lh dist/ykc-skills.zip

echo "==> npm publish dry-run..."
cd dist/npm/yunku-cli
npm publish --access public --dry-run

echo "==> All checks passed!"