#!/usr/bin/env bash
#
# Script to run a test suite in docker locally
set -eux

if [ -z "$DRIVERS_TOOLS" ]; then
    echo "Please set DRIVERS_TOOLS env variable."
    exit 1
fi
PLATFORM=${DOCKER_PLATFORM:-}

pushd $DRIVERS_TOOLS/.evergreen/docker/ubuntu20.04
docker build $PLATFORM -t drivers-evergreen-tools .
popd
docker build $PLATFORM -t go-test .

# Handle environment variables and optional positional arg for the makefile target.

MAKEFILE_TARGET=${1:-evg-test-versioned-api}
MONGODB_VERSION=${MONGODB_VERSION:-latest}
TOPOLOGY=${TOPOLOGY:-replica_set}
ORCHESTRATION_FILE=${ORCHESTRATION_FILE:-basic.json}
AUTH=${AUTH:-""}
SSL=${SSL:=""}
GO_BUILD_TAGS=${GO_BUILD_TAGS:-""}

ARGS="$PLATFORM --rm -i"
ARGS="$ARGS -e MONGODB_VERSION=$MONGODB_VERSION -e TOPOLOGY=$TOPOLOGY"
ARGS="$ARGS -e MAKEFILE_TARGET=$MAKEFILE_TARGET -e AUTH=$AUTH"
ARGS="$ARGS -e ORCHESTRATION_FILE=$ORCHESTRATION_FILE -e SSL=$SSL"
ARGS="$ARGS -e GO_BUILD_TAGS=$GO_BUILD_TAGS"
ARGS="$ARGS -e DRIVERS_TOOLS=/root/drivers-evergeen-tools"
ARGS="$ARGS -e MONGODB_URI=mongodb://host.docker.internal"

# Ensure host.docker.internal is available on Linux.
if [ "$(uname -s)" = "Linux" ]; then
    ARGS="$ARGS --add-host host.docker.internal:127.0.0.1"
fi

# If there is a tty, add the -t arg.
test -t 1 && ARGS="-t $ARGS"

ARGS="$ARGS -v `pwd`:/src"
ARGS="$ARGS -v $DRIVERS_TOOLS:/root/drivers-evergreen-tools"

docker run $ARGS go-test
if [ -f "test.suite" ]; then
    tail test.suite
fi
