#!/usr/bin/env bash
#
# Script to run a test suite in docker locally
set -eux

if [ -z "$DRIVERS_TOOLS" ]; then
    echo "Please set DRIVERS_TOOLS env variable."
    exit 1
fi
PLATFORM=${DOCKER_PLATFORM:-}
docker build $PLATFORM -t go-test .

# Handle environment variables and optional positional arg for the taskfile target.
TASKFILE_TARGET=${TASKFILE_TARGET:-$1}
TASKFILE_TARGET=${TASKFILE_TARGET:-evg-test-versioned-api}
GO_BUILD_TAGS=${GO_BUILD_TAGS:-""}

ARGS=" -e TASKFILE_TARGET=$TASKFILE_TARGET"
ARGS="$ARGS -e GO_BUILD_TAGS=$GO_BUILD_TAGS"
ARGS="$ARGS go-test"

$DRIVERS_TOOLS/.evergreen/docker/run-client.sh $ARGS
if [ -f "test.suite" ]; then
    tail test.suite
fi
