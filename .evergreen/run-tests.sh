#!/bin/bash

set -o errexit
set -o xtrace

export GOPATH=$(dirname $(dirname $(dirname `pwd`)))
export GOCACHE="$(pwd)/.cache"

if [ "Windows_NT" = "$OS" ]; then
    export GOPATH=$(cygpath -m $GOPATH)
    export GOCACHE=$(cygpath -m $GOCACHE)

    mkdir -p c:/libmongocrypt/include
    mkdir -p c:/libmongocrypt/bin
    curl https://s3.amazonaws.com/mciuploads/libmongocrypt/windows/latest_release/libmongocrypt.tar.gz --output libmongocrypt.tar.gz
    tar -xvzf libmongocrypt.tar.gz
    cp ./bin/mongocrypt.dll c:/libmongocrypt/bin
    cp ./include/mongocrypt/*.h c:/libmongocrypt/include
    export PATH=$PATH:/cygdrive/c/libmongocrypt/bin
else
    git clone https://github.com/mongodb/libmongocrypt
    ./libmongocrypt/.evergreen/compile.sh
fi

export GOROOT="${GOROOT}"
export PATH="${GOROOT}/bin:${GCC_PATH}:$GOPATH/bin:$PATH"
export PKG_CONFIG_PATH=$(pwd)/install/libmongocrypt/lib/pkgconfig:$(pwd)/install/mongo-c-driver/lib/pkgconfig
export LD_LIBRARY_PATH=$(pwd)/install/libmongocrypt/lib
export GOFLAGS=-mod=vendor

SSL=${SSL:-nossl}
if [ "$SSL" != "nossl" ]; then
    export MONGO_GO_DRIVER_CA_FILE="${DRIVERS_TOOLS}/.evergreen/x509gen/ca.pem"
    export MONGO_GO_DRIVER_KEY_FILE="${DRIVERS_TOOLS}/.evergreen/x509gen/client.pem"
    export MONGO_GO_DRIVER_PKCS8_ENCRYPTED_KEY_FILE="${DRIVERS_TOOLS}/.evergreen/x509gen/client-pkcs8-encrypted.pem"
    export MONGO_GO_DRIVER_PKCS8_UNENCRYPTED_KEY_FILE="${DRIVERS_TOOLS}/.evergreen/x509gen/client-pkcs8-unencrypted.pem"

    if [ "Windows_NT" = "$OS" ]; then
        export MONGO_GO_DRIVER_CA_FILE=$(cygpath -m $MONGO_GO_DRIVER_CA_FILE)
        export MONGO_GO_DRIVER_KEY_FILE=$(cygpath -m $MONGO_GO_DRIVER_KEY_FILE)
        export MONGO_GO_DRIVER_PKCS8_ENCRYPTED_KEY_FILE=$(cygpath -m $MONGO_GO_DRIVER_PKCS8_ENCRYPTED_KEY_FILE)
        export MONGO_GO_DRIVER_PKCS8_UNENCRYPTED_KEY_FILE=$(cygpath -m $MONGO_GO_DRIVER_PKCS8_UNENCRYPTED_KEY_FILE)
    fi
fi

AUTH=${AUTH} \
SSL=${SSL} \
MONGO_GO_DRIVER_CA_FILE=${MONGO_GO_DRIVER_CA_FILE} \
MONGO_GO_DRIVER_KEY_FILE=${MONGO_GO_DRIVER_KEY_FILE} \
MONGO_GO_DRIVER_PKCS8_ENCRYPTED_KEY_FILE=${MONGO_GO_DRIVER_PKCS8_ENCRYPTED_KEY_FILE} \
MONGO_GO_DRIVER_PKCS8_UNENCRYPTED_KEY_FILE=${MONGO_GO_DRIVER_PKCS8_UNENCRYPTED_KEY_FILE} \
MONGODB_URI="${MONGODB_URI}" \
TOPOLOGY=${TOPOLOGY} \
BUILD_TAGS="-tags cse" \
AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID}" \
AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY}" \
make evg-test \
PKG_CONFIG_PATH=$PKG_CONFIG_PATH \
LD_LIBRARY_PATH=$LD_LIBRARY_PATH
