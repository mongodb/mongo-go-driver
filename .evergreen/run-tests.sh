#!/bin/bash

set -o errexit
set -o xtrace

export GOPATH=$(dirname $(dirname $(dirname `pwd`)))
export GOCACHE="$(pwd)/.cache"

if [ "Windows_NT" = "$OS" ]; then
    export GOPATH=$(cygpath -m $GOPATH)
    export GOCACHE=$(cygpath -m $GOCACHE)
fi

export GOROOT="${GOROOT}"
export PATH="${GOROOT}/bin:${GCC_PATH}:$GOPATH/bin:$PATH"
export GOFLAGS=-mod=vendor

SSL=${SSL:-nossl}
if [ "$SSL" != "nossl" ]; then
    export MONGO_GO_DRIVER_CA_FILE="$DRIVERS_TOOLS/.evergreen/x509gen/ca.pem"
    export MONGO_GO_DRIVER_KEY_FILE="$DRIVERS_TOOLS/.evergreen/x509gen/client.pem"

    if [ "Windows_NT" = "$OS" ]; then
        export MONGO_GO_DRIVER_CA_FILE=$(cygpath -m $MONGO_GO_DRIVER_CA_FILE)
        export MONGO_GO_DRIVER_KEY_FILE=$(cygpath -m $MONGO_GO_DRIVER_KEY_FILE)
    fi
fi

AUTH=${AUTH} \
SSL=${SSL} \
MONGO_GO_DRIVER_CA_FILE=${MONGO_GO_DRIVER_CA_FILE} \
MONGO_GO_DRIVER_KEY_FILE=${MONGO_GO_DRIVER_KEY_FILE} \
MONGODB_URI="${MONGODB_URI}" \
TOPOLOGY=${TOPOLOGY} \
make evg-test
