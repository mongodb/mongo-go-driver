#!/usr/bin/env bash
# run-compile-check-test
# Run compile check tests.
set -eu
set +x

echo "Running internal/test/compilecheck"
pushd internal/test/compilecheck
GOWORK=off go test -v ./... >>../../../test.suite
popd
