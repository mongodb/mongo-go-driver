#!/usr/bin/env bash
# run-compile-check-test
# Run compile check tests.
set -eu
set +x

# Use specified GO_VERSIONS or default to all supported versions.
# To run the compile check tests with specific Go versions, set the GO_VERSIONS environment variable before running this script.
# For example, to use a single version: GO_VERSIONS=1.25 ./etc/run-compile-check-test.sh
# Or to use multiple versions: GO_VERSIONS="1.23,1.24,1.25" ./etc/run-compile-check-test.sh
if [ -z "${GO_VERSIONS:-}" ]; then
    GO_VERSIONS="1.19,1.20,1.21,1.22,1.23,1.24,1.25"
fi
export GO_VERSIONS

echo "Running internal/test/compilecheck with Go versions: $GO_VERSIONS"
pushd internal/test/compilecheck
GOWORK=off go test -timeout 30m -v ./... >>../../../test.suite
popd
