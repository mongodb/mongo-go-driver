#!/usr/bin/env bash
set -ex

# Keep this in sync with go version used in static-analysis Evergreen build variant.
GO_VERSION=1.25.0
GOLANGCI_LINT_VERSION=2.8.0

# Unset the cross-compiler overrides while downloading binaries.
GOOS_ORIG=${GOOS:-}
export GOOS=
GOARCH_ORIG=${GOARCH:-}
export GOARCH=

go install golang.org/dl/go$GO_VERSION@latest
go${GO_VERSION} download
GOROOT="$(go${GO_VERSION} env GOROOT)"
PATH="$GOROOT/bin:$PATH"
export PATH
export GOROOT
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b "$(go env GOPATH)"/bin v${GOLANGCI_LINT_VERSION}

export GOOS=$GOOS_ORIG
export GOARCH=$GOARCH_ORIG
golangci-lint run --config .golangci.yml ./...
