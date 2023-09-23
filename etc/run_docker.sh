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

VOL="-v `pwd`:/src"
VOL="$VOL -v $DRIVERS_TOOLS:/root/drivers-evergreen-tools"
USE_TTY=""
test -t 1 && USE_TTY="-t"

docker run $PLATFORM --rm $VOL $ENV -i $USE_TTY go-test
if [ -f "test.suite" ]; then
    tail test.suite
fi
