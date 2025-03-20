#!/usr/bin/env bash
# run-compile-check-test
# Run compile check tests.
set -eu
set +x

SUBTEST="${1:-}" # Use the default value if $1 is not set

echo "Running internal/test/compilecheck"
pushd internal/test/compilecheck
if [ -n "$SUBTEST" ]; then
  GOWORK=off go test -run "$SUBTEST" -timeout 30m -v ./... >>../../../test.suite
else
  GOWORK=off go test -timeout 30m -v ./... >>../../../test.suite
fi
popd
