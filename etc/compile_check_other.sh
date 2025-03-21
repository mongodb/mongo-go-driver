#!/usr/bin/env bash
set -e # exit when any command fails
set -x # show all commands being run

GC=go
COMPILE_CHECK_DIR="internal/cmd/compilecheck"
ARCHITECTURES=("386" "arm" "arm64" "ppc64le" "s390x")
BUILD_CMD="${GC} build -buildvcs=false"

# Compilation modules are not part of the workspace as testcontainers requires
# a version of klauspost/compress not supported by the Go Driver / other modules
# in the workspace.
export GOWORK=off

# compile_check will attempt to build the internal/test/compilecheck project
# using the provided Go version. This is to simulate an end-to-end use case.
function compile_check {
  # Change the directory to the compilecheck test directory.
  pushd "${COMPILE_CHECK_DIR}" >/dev/null

  ${GC} version
  ${GC} mod tidy

  # Standard build
  $BUILD_CMD ./...

  # Dynamic linking
  $BUILD_CMD -buildmode=plugin

  # Check build with tags.
  [[ -n "$BUILD_TAGS" ]] && $BUILD_CMD $BUILD_TAGS ./...

  # Check build with various architectures.
  for ARCH in "${ARCHITECTURES[@]}"; do
    GOOS=linux GOARCH=$ARCH $BUILD_CMD ./...
  done

  # Change the directory back to the working directory.
  popd >/dev/null
}

compile_check
