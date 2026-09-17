#!/bin/sh
set -eu

# 输出路径可通过 YKC_BUILD_OUTPUT 覆盖，默认 ./ykc
OUTPUT="${YKC_BUILD_OUTPUT:-ykc}"

go build -buildmode=pie -trimpath -ldflags="-s -w" -o "$OUTPUT" ./cmd
