#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "==> go build ./..."
go build ./...

echo "==> go vet ./..."
go vet ./...

echo "==> go test -race ./..."
go test -race ./...

echo "==> golangci-lint run"
golangci-lint run

echo "CI checks passed."
