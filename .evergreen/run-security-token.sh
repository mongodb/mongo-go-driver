#!/bin/bash

set -o errexit

export GOPATH=$(dirname $(dirname $(dirname `pwd`)))
export GOCACHE="$(pwd)/.cache"
export DRIVERS_TOOLS="$(pwd)/../drivers-tools"

if [ "Windows_NT" = "$OS" ]; then
    export GOPATH=$(cygpath -m $GOPATH)
    export GOCACHE=$(cygpath -m $GOCACHE)
    export DRIVERS_TOOLS=$(cygpath -m $DRIVERS_TOOLS)
fi

export GOROOT="${GOROOT}"
export PATH="${GOROOT}/bin:${GCC_PATH}:$GOPATH/bin:$PATH"
export PROJECT="${project}"
export GOFLAGS=-mod=vendor

MONGODB_URI="${MONGODB_URI}" \
TOPOLOGY=${TOPOLOGY} \
MONGO_GO_DRIVER_COMPRESSOR=${MONGO_GO_DRIVER_COMPRESSOR} \
# This JWT security token is for testing purposes only and does not authorize anything.
MONGODB_SECURITY_TOKEN="eyJhbGciOiJub25lIiwidHlwIjoiSldUIiwia2lkIjoiIn0.eyJpc3MiOiJodHRwczovL2Zha2Vpc3MiLCJzdWIiOiJhbGljZSIsImF1ZCI6Imh0dHBzOi8vZmFrZWF1ZCIsIm1vbmdvZGIvdGVuYW50SWQiOiJhYmNkMTIzNGFiY2QxMjM0YWJjZDEyMzQiLCJtb25nb2RiL2V4cGVjdFByZWZpeCI6ZmFsc2UsIm1vbmdvZGIvcm9sZXMiOlsicm9sZTEiLCJyb2xlMiJdLCJleHAiOjE3OTM5ODM3MjR9." \
make evg-test-security-token
