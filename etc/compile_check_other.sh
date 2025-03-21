#!/usr/bin/env bash
set -e # exit when any command fails
set -x # show all commands being run

: ${GO_VERSION:="1.18"} # Default to 1.18 if GO_VERSION is unset or empty
: ${GC:=go$GO_VERSION}  # Use existing GC or default to "go$GO_VERSION"

COMPILE_CHECK_DIR="internal/cmd/compilecheck"
ARCHITECTURES=("386" "arm" "arm64" "ppc64le" "s390x")
BUILD_CMD="${GC} build -buildvcs=false"

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
