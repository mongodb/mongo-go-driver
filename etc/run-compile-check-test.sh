#!/usr/bin/env bash
# run-compile-check-test
# Run compile check tests.
set -eu
set +x

SUBTEST=$1

echo "Running internal/test/compilecheck"
pushd internal/test/compilecheck
if [ -n "${SUBTEST-}" ]; then
  # If $1 is set and non-empty, use it with -run
  GOWORK=off go test -run "$SUBTEST" -timeout 30m -v ./... >>../../../test.suite
else
  # If $1 is not set, run tests without -run
  GOWORK=off go test -timeout 30m -v ./... >>../../../test.suite
fi
popd
