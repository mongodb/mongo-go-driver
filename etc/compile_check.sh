#!/usr/bin/env bash
set -e # exit when any command fails

GC=go
WD=$(pwd)
COMPILE_CHECK_DIR="internal/test/compilecheck"
DEV_MIN_VERSION=1.19


# version will flatten a version string of upto 4 components for inequality
# comparison.
function version {
	echo "$@" | awk -F. '{ printf("%d%03d%03d%03d\n", $1,$2,$3,$4); }';
}

# dev_compile_check will attempt to build in the development environment. This
# check will only run on environments where the Go version is greater than or
# equal to the DEV_MIN_VERSION.
function dev_compile_check {
	VERSION=`${GC} version | { read _ _ v _; echo ${v#go}; }`

	if [ $(version $VERSION) -ge $(version $DEV_MIN_VERSION) ]; then
		GO111MODULE=on ${GC} build ./...
		GO111MODULE=on ${GC} build ${BUILD_TAGS} ./...
	fi
}

# compile_check will attemps to build the the internal/test/compilecheck project
# using the provided go version. This is to simulate a end-to-end use case.
function compile_check {
	GO111MODULE=$1
	VERSION=$2

	# Change the directory to the compilecheck test directory.
	cd ${COMPILE_CHECK_DIR}

	# If the version is not 1.13, then run "go mod tidy"
	if [ "$VERSION" != 1.13 ]; then
		go mod tidy -go=$VERSION
	fi

	${GC} build ./...
	${GC} build -buildmode=plugin
	${GC} build ${BUILD_TAGS} ./...

	GOOS=linux GOARCH=386 ${GC} build ${BUILD_TAGS}
	GOOS=linux GOARCH=arm ${GC} build ${BUILD_TAGS}
	GOOS=linux GOARCH=arm64 ${GC} build ${BUILD_TAGS}
	GOOS=linux GOARCH=ppc64le ${GC} build ${BUILD_TAGS}
	GOOS=linux GOARCH=s390x ${GC} build ${BUILD_TAGS}

	# Reset any changes to the "go.mod" and "go.sum" files.
	git checkout HEAD -- go.mod

	# Remove the binaries.
	rm compilecheck
	rm compilecheck.so

	# Change the directory back to the working directory.
	cd ${WD}
}

dev_compile_check
compile_check on 1.19
compile_check on 1.13
