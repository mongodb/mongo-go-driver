#!/usr/bin/env bash
set -ex

# Unset the cross-compiler overrides while downloading binaries.
GOOS_ORIG=${GOOS:-}
export GOOS=
GOARCH_ORIG=${GOARCH:-}
export GOARCH=

# Keep this in sync with go version used in static-analysis Evergreen build variant.
go install golang.org/dl/go1.22.7@latest
go1.22.7 download
GOROOT="$(go1.22.7 env GOROOT)"
PATH="$GOROOT/bin:$PATH"
export GOROOT
export PATH
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.1

export GOOS=$GOOS_ORIG
export GOARCH=$GOARCH_ORIG
golangci-lint run --config .golangci.yml ./...
