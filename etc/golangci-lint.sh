#!/usr/bin/env bash
# Keep this in sync with go version used in static-analysis Evergreen build variant.
go install golang.org/dl/go1.22.7@latest
go1.22.7 download 2> /dev/null
PATH="$(go1.22.7 env GOROOT)/bin:$PATH" golangci-lint run --config .golangci.yml ./...