#!/usr/bin/env bash
#
# Script to run a test suite in docker locally
set -eux

if [ -z "$DRIVERS_TOOLS" ]; then
    echo "Please set DRIVERS_TOOLS env variable."
    exit 1
fi
pushd $DRIVERS_TOOLS/.evergreen/docker
docker build -t drivers-evergreen-tools .
popd
docker build -t go-test .

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

# Remove old mongodb binaries to save time.
rm -rf $DRIVERS_TOOLS/mongodb

docker run --rm $VOL $ENV -t go-test
if [ -f "test.suite" ]; then
    tail test.suite
fi
