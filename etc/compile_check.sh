#!/usr/bin/env bash
set -e # exit when any command fails
set -x # show all commands being run

GC=go
COMPILE_CHECK_DIR="internal/test/compilecheck"
DEV_MIN_VERSION=1.19

# version will flatten a version string of upto 4 components for inequality
# comparison.
function version {
	echo "$@" | awk -F. '{ printf("%d%03d%03d%03d\n", $1,$2,$3,$4); }';
}

# compile_check will attempt to build the the internal/test/compilecheck project
# using the provided Go version. This is to simulate an end-to-end use case.
# This check will only run on environments where the Go version is greater than
# or equal to the given version.
function compile_check {
	# Change the directory to the compilecheck test directory.
	cd ${COMPILE_CHECK_DIR}

	# Test vendoring
	go mod vendor
	${GC} build -mod=vendor

	rm -rf vendor

	MACHINE_VERSION=`${GC} version | { read _ _ v _; echo ${v#go}; }`

	# If the version is not 1.13, then run "go mod tidy"
	if [ $(version $MACHINE_VERSION) -ge $(version 1.15) ]; then
		go mod tidy
	fi

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
	cd -
}

compile_check
