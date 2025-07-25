#!/usr/bin/env bash
# perf-pr-comment
# Generates a report of Go Driver perf changes for the current branch.

set -eux

pushd ./internal/cmd/perfcomp >/dev/null || exist
GOWORK=off go run main.go --project="mongo-go-driver" ${VERSION_ID}
popd >/dev/null
