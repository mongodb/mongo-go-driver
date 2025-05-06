#!/usr/bin/env bash
# run-goleak-test
# Run goroutine leak tests.
set -eu
set +x

echo "Running internal/test/goleak"
pushd internal/test/goleak
go test -v ./... >> ../../../test.suite
popd
