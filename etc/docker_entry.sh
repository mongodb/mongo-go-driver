#!/usr/bin/env bash
#
# Entry point for Dockerfile for launching a server and running a go test.
#
set -eux

# TODO: make some things configurable

export MONGODB_VERSION=latest
export TOPOLOGY=replica_set
export ORCHESTRATION_FILE=basic.json
export DRIVERS_TOOLS=$HOME/drivers-evergreen-tools
export PROJECT_ORCHESTRATION_HOME=$DRIVERS_TOOLS/.evergreen/orchestration
export MONGO_ORCHESTRATION_HOME=$HOME

if [ ! -d $DRIVERS_TOOLS ]; then
    git clone https://github.com/mongodb-labs/drivers-evergreen-tools.git $DRIVERS_TOOLS
fi

# Disable ipv6
sed -i "s/\"ipv6\": true,/\"ipv6\": false,/g" $PROJECT_ORCHESTRATION_HOME/configs/${TOPOLOGY}s/$ORCHESTRATION_FILE


bash $DRIVERS_TOOLS/.evergreen/run-orchestration.sh

cd /src
export GOPATH=$(go env GOPATH)
export GOCACHE="$(pwd)/.cache"
export LD_LIBRARY_PATH=$HOME/install/libmongocrypt/lib
export PKG_CONFIG_PATH=$HOME/install/libmongocrypt/lib/pkgconfig:$HOME/install/mongo-c-driver/lib/pkgconfig
export BUILD_TAGS="-tags=cse"
make evg-test
