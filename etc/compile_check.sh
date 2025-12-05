#!/usr/bin/env bash
set -e # exit when any command fails
set -x # show all commands being run

ARCHITECTURES=("386" "arm" "arm64" "ppc64le" "s390x")

go version

# Standard build
go build ./...

# Check build with tags if specified.
if [ -n "${BUILD_TAGS:-}" ]; then
    go build ${BUILD_TAGS} ./...
fi

# Check build with various architectures.
for ARCH in "${ARCHITECTURES[@]}"; do
    GOOS=linux GOARCH=$ARCH go build ./...
done
