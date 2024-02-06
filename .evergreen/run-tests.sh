#!/bin/bash

set -o errexit

export GOPATH=$(dirname $(dirname $(dirname `pwd`)))
export GOCACHE="$(pwd)/.cache"
export DRIVERS_TOOLS=${DRIVERS_TOOLS:-""}

if [ -z $DRIVERS_TOOLS ]; then
  export DRIVERS_TOOLS="$(dirname $(dirname $(dirname `pwd`)))/drivers-tools"
fi

if [ "Windows_NT" = "${OS:-}" ]; then
    export GOPATH=$(cygpath -m $GOPATH)
    export GOCACHE=$(cygpath -m $GOCACHE)
    export DRIVERS_TOOLS=$(cygpath -m $DRIVERS_TOOLS)
fi

export GOROOT="${GOROOT}"
export PATH="${GOROOT}/bin:${GCC_PATH}:$GOPATH/bin:$PATH"
export PROJECT="${project}"

if [ "$(uname -s)" = "Darwin" ]; then
  export PKG_CONFIG_PATH=$(pwd)/install/libmongocrypt/lib/pkgconfig
  export DYLD_FALLBACK_LIBRARY_PATH=$(pwd)/install/libmongocrypt/lib
else
  export PKG_CONFIG_PATH=$(pwd)/install/libmongocrypt/lib64/pkgconfig
  export LD_LIBRARY_PATH=$(pwd)/install/libmongocrypt/lib64
fi

export GOFLAGS=-mod=vendor

SSL=${SSL:-nossl}
if [ "$SSL" != "nossl" -a -z "${SERVERLESS+x}" ]; then
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

if [ -f "secrets-export.sh" ]; then
    source $(pwd)/secrets-export.sh
fi

# If GO_BUILD_TAGS is not set, set the default Go build tags to "cse" to enable
# client-side encryption, which requires linking the libmongocrypt C library.
if [ -z ${GO_BUILD_TAGS+x} ]; then
  GO_BUILD_TAGS="cse"
fi

if [[ $GO_BUILD_TAGS == *"cse"* ]]; then
  if [ "Windows_NT" = "$OS" ]; then
    if [ ! -d /cygdrive/c/libmongocrypt/bin ]; then
      bash $(pwd)/etc/install-libmongocrypt.sh
    fi
    export PATH=$PATH:/cygdrive/c/libmongocrypt/bin
  elif [ ! -d "$PKG_CONFIG_PATH" ]; then
    bash $(pwd)/etc/install-libmongocrypt.sh
  fi
fi

if [ "${SKIP_CRYPT_SHARED_LIB}" = "true" ]; then
  CRYPT_SHARED_LIB_PATH=""
  echo "crypt_shared library is skipped"
elif [ -z "${CRYPT_SHARED_LIB_PATH}" ]; then
  echo "crypt_shared library path is empty"
else
  CRYPT_SHARED_LIB_PATH=${CRYPT_SHARED_LIB_PATH}
  echo "crypt_shared library will be loaded from path: $CRYPT_SHARED_LIB_PATH"
fi

if [ -z ${MAKEFILE_TARGET+x} ]; then
  if [ "$(uname -s)" = "Darwin" ]; then
      # Run a subset of the tests on Darwin
      MAKEFILE_TARGET="evg-test-load-balancers"
  else
    MAKEFILE_TARGET="evg-test"
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
MONGO_GO_DRIVER_COMPRESSOR=${MONGO_GO_DRIVER_COMPRESSOR} \
BUILD_TAGS="${RACE} -tags=${GO_BUILD_TAGS}" \
CRYPT_SHARED_LIB_PATH=$CRYPT_SHARED_LIB_PATH \
PKG_CONFIG_PATH=$PKG_CONFIG_PATH \
LD_LIBRARY_PATH=$LD_LIBRARY_PATH \
MACOS_LIBRARY_PATH=$DYLD_FALLBACK_LIBRARY_PATH \
make $MAKEFILE_TARGET
