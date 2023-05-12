#!/usr/bin/env bash
GC=go
WD=$(pwd)
COMPILE_CHECK_DIR="internal/test/compilecheck"

# dev_compile_check will attempt to build in the development environment.
function dev_compile_check {
	GO111MODULE=on ${GC} build ./...
	GO111MODULE=on ${GC} build ${BUILD_TAGS} ./...
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

	# Change the directory back to the working directory.
	cd ${WD}
}

dev_compile_check
compile_check on 1.19
compile_check on 1.13
