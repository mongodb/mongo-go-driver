#!/usr/bin/env bash
# run-compile-check-test
# Run compile check tests for all supported Go versions.
set -eu
set +x

echo "Running internal/test/compilecheck"
pushd internal/test/compilecheck
go test -timeout 30m -v ./... >>../../../test.suite
popd
