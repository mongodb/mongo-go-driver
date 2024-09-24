#!/usr/bin/env bash
# Keep this in sync with go version used in static-analysis Evergreen build variant.
set -ex
go install golang.org/dl/go1.22.7@latest
# Prepend the cross-platform path if necessary.
if [ -n "$GOOS" ]; then
    export PATH="$GOPATH/bin/$GOOS_$GOGOARCH:$PATH"
fi
go1.22.7 download
export PATH="$(go1.22.7 env GOROOT)/bin:$PATH"
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.1
golangci-lint run --config .golangci.yml ./...
