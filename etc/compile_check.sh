#!/usr/bin/env bash
set -e # exit when any command fails
set -x # show all commands being run

GC=go
COMPILE_CHECK_DIR="internal/cmd/compilecheck"
# shellcheck disable=SC2034

# compile_check will attempt to build the internal/test/compilecheck project
# using the provided Go version. This is to simulate an end-to-end use case.
function compile_check {
	# Change the directory to the compilecheck test directory.
	pushd ${COMPILE_CHECK_DIR}

	${GC} version
	${GC} mod tidy

	# Check simple build.
	${GC} build ./...

	# Check build with dynamic linking.
	${GC} build -buildmode=plugin

	# Check build with tags.
	${GC} build $BUILD_TAGS ./...

	# Check build with various architectures.
	GOOS=linux GOARCH=386 ${GC} build ./...
	GOOS=linux GOARCH=arm ${GC} build ./...
	GOOS=linux GOARCH=arm64 ${GC} build ./...
	GOOS=linux GOARCH=amd64 ${GC} build ./...
	GOOS=linux GOARCH=ppc64le ${GC} build ./...
	GOOS=linux GOARCH=s390x ${GC} build ./...

	# Remove the binaries.
	rm compilecheck
	rm compilecheck.so

	# Change the directory back to the working directory.
	popd
}

compile_check
