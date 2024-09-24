#!/usr/bin/env bash
# Keep this in sync with go version used in static-analysis Evergreen build variant.
set -ex
GOOS_ORIG=${GOOS:-}
export GOOS=
GOARCH_ORIG=${GOARCH:-}
export GOARCH=

go install golang.org/dl/go1.22.7@latest
go1.22.7 download
export PATH="$(go1.22.7 env GOROOT)/bin:$PATH"
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.1

export GOOS=$GOOS_ORIG
export GOARCH=$GOARCH_ORIG
golangci-lint run --config .golangci.yml ./...
