#!/bin/sh
set -eu

go build -buildmode=pie -trimpath -ldflags="-s -w" -o ykc ./cmd
