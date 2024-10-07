#!/usr/bin/env bash
set -ex

GO_VERSION=1.22.8
GOLANGCI_LINT_VERSION=1.60.1

# Unset the cross-compiler overrides while downloading binaries.
GOOS_ORIG=${GOOS:-}
export GOOS=
GOARCH_ORIG=${GOARCH:-}
export GOARCH=

# Keep this in sync with go version used in static-analysis Evergreen build variant.
go install golang.org/dl/go$GO_VERSION@latest
go$GO_VERSION download
PATH="$(go$GO_VERSION env GOROOT)/bin:$PATH"
export PATH
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v$GOLANGCI_LINT_VERSION

export GOOS=$GOOS_ORIG
export GOARCH=$GOARCH_ORIG
golangci-lint run --config .golangci.yml ./...
