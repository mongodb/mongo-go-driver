#!/usr/bin/env bash
# Run containerized tests
set -eux

echo "Running internal/test"

# Defaults so -u doesn't explode if these aren't provided
: "${TEST_TIMEOUT:=1800}" # seconds
: "${BUILD_TAGS:=}"       # optional build tags
: "${PKG_CONFIG_PATH:=}"
: "${LD_LIBRARY_PATH:=}"
: "${MACOS_LIBRARY_PATH:=}"

# Only add -tags if BUILD_TAGS is non-empty
tags=()
if [ -n "$BUILD_TAGS" ]; then
  tags=("$BUILD_TAGS")
fi

pushd internal/test
# Run tests with env on the go process so both build and run see it
env PKG_CONFIG_PATH="$PKG_CONFIG_PATH" \
  LD_LIBRARY_PATH="$LD_LIBRARY_PATH" \
  DYLD_LIBRARY_PATH="$MACOS_LIBRARY_PATH" \
  go test ${tags[@]+"${tags[@]}"} -v -timeout "${TEST_TIMEOUT}s" -p 1 . >>../../test.suite
popd
