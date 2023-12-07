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

ENV="-e MONGODB_VERSION=$MONGODB_VERSION -e TOPOLOGY=$TOPOLOGY"
ENV="$ENV -e MAKEFILE_TARGET=$MAKEFILE_TARGET -e AUTH=$AUTH"
ENV="$ENV -e ORCHESTRATION_FILE=$ORCHESTRATION_FILE -e SSL=$SSL"
ENV="$ENV -e GO_BUILD_TAGS=$GO_BUILD_TAGS"
ENV="$ENV -e MONGODB_URI=mongodb://host.docker.internal"

# Ensure host.docker.internal is available on Linux.
EXTRA_ARGS=""
if [ "$(uname -s)" = "Linux" ]; then
    EXTRA_ARGS="--add-host"
fi

# If there is a tty, add the -t arg.
test -t 1 && EXTRA_ARGS="-t $EXTRA_ARGS"

VOL="-v `pwd`:/src"
VOL="$VOL -v $DRIVERS_TOOLS:/root/drivers-evergreen-tools"

docker run $PLATFORM --rm $VOL $ENV $EXTRA_ARGS -i go-test
if [ -f "test.suite" ]; then
    tail test.suite
fi
