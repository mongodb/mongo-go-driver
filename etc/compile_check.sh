#!/usr/bin/env bash
set -e # exit when any command fails
set -x # show all commands being run
#
## Default to Go 1.18 if GO_VERSION is not set.
##
## Use the "=" operator (instead of the more common ":-" operator) so that it
## allows setting GO_VERSION="" to use the Go installation in the PATH, and it
## sets the GO_VERSION variable if the default is used.
#GC=go${GO_VERSION="1.18"}
#COMPILE_CHECK_DIR="internal/cmd/compilecheck"
#
## compile_check will attempt to build the internal/test/compilecheck project
## using the provided Go version. This is to simulate an end-to-end use case.
#function compile_check {
#  # Change the directory to the compilecheck test directory.
#  pushd ${COMPILE_CHECK_DIR}
#
#  # If a custom Go version is set using the GO_VERSION env var (e.g. "1.18"),
#  # add the GOPATH bin directory to PATH and then install that Go version.
#  if [ ! -z "$GO_VERSION" ]; then
#    PATH=$(go env GOPATH)/bin:$PATH
#    export PATH
#
#    go install golang.org/dl/go$GO_VERSION@latest
#    ${GC} download
#  fi
#
#  ${GC} version
#  ${GC} mod tidy
#
#  # Check simple build.
#  ${GC} build ./...
#
#  # Check build with dynamic linking.
#  ${GC} build -buildmode=plugin
#
#  # Check build with tags.
#  ${GC} build $BUILD_TAGS ./...
#
#  # Check build with various architectures.
#  GOOS=linux GOARCH=386 ${GC} build ./...
#  GOOS=linux GOARCH=arm ${GC} build ./...
#  GOOS=linux GOARCH=arm64 ${GC} build ./...
#  GOOS=linux GOARCH=amd64 ${GC} build ./...
#  GOOS=linux GOARCH=ppc64le ${GC} build ./...
#  GOOS=linux GOARCH=s390x ${GC} build ./...
#
#  # Remove the binaries.
#  rm compilecheck
#  rm compilecheck.so
#
#  # Change the directory back to the working directory.
#  popd
#}
#
#compile_check
##!/usr/bin/env bash
#set -e # exit when any command fails
#set -x # show all commands being run
#
#GC=go
#COMPILE_CHECK_DIR="internal/cmd/compilecheck"
#ARCHITECTURES=("386" "arm" "arm64" "ppc64le" "s390x")
#BUILD_CMD="${GC} build -buildvcs=false"
#
## Compilation modules are not part of the workspace as testcontainers requires
## a version of klauspost/compress not supported by the Go Driver / other modules
## in the workspace.
#export GOWORK=off
#
## compile_check will attempt to build the internal/test/compilecheck project
## using the provided Go version. This is to simulate an end-to-end use case.
#function compile_check {
#  # Change the directory to the compilecheck test directory.
#  pushd "${COMPILE_CHECK_DIR}" >/dev/null
#
#  ${GC} version
#  ${GC} mod tidy
#
#  # Standard build
#  $BUILD_CMD ./...
#
#  # Dynamic linking
#  $BUILD_CMD -buildmode=plugin
#
#  # Check build with tags.
#  [[ -n "$BUILD_TAGS" ]] && $BUILD_CMD $BUILD_TAGS ./...
#
#  # Check build with various architectures.
#  for ARCH in "${ARCHITECTURES[@]}"; do
#    GOOS=linux GOARCH=$ARCH $BUILD_CMD ./...
#  done
#
#  # Change the directory back to the working directory.
#  popd >/dev/null
#}
#
#compile_check
