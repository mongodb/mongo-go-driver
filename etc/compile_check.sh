#!/usr/bin/env bash
set -e # exit when any command fails
set -x # show all commands being run

# Default to Go 1.18 if GO_VERSION is not set.
#
# Use the "=" operator (instead of the more common ":-" operator) so that it
# allows setting GO_VERSION="" to use the Go installation in the PATH, and it
# sets the GO_VERSION variable if the default is used.
GC=go
COMPILE_CHECK_DIR="internal/cmd/compilecheck"

# compile_check will attempt to build the internal/test/compilecheck project
# using the provided Go version. This is to simulate an end-to-end use case.
function compile_check {
  # Change the directory to the compilecheck test directory.
  pushd ${COMPILE_CHECK_DIR}

  ${GC} version
  ${GC} mod tidy

  # Check simple build.
  GOWORK=off ${GC} build -buildvcs=false ./...

  # Check build with dynamic linking.
  GOWORK=off ${GC} build -buildvcs=false -buildmode=plugin

  # Check build with tags.
  GOWORK=off ${GC} build -buildvcs=false $BUILD_TAGS ./...

  # Check build with various architectures.
  GOWORK=off GOOS=linux GOARCH=386 ${GC} build -buildvcs=false ./...
  GOWORK=off GOOS=linux GOARCH=arm ${GC} build -buildvcs=false ./...
  GOWORK=off GOOS=linux GOARCH=arm64 ${GC} build -buildvcs=false ./...
  GOWORK=off GOOS=linux GOARCH=amd64 ${GC} build -buildvcs=false ./...
  GOWORK=off GOOS=linux GOARCH=ppc64le ${GC} build -buildvcs=false ./...
  GOWORK=off GOOS=linux GOARCH=s390x ${GC} build -buildvcs=false ./...

  # Change the directory back to the working directory.
  popd
}

compile_check

exit 0
