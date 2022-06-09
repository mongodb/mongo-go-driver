#!/bin/bash

set -o errexit

echo "TEST"

export GOPATH=$(dirname $(dirname $(dirname `pwd`)))
export GOCACHE="$(pwd)/.cache"
export DRIVERS_TOOLS="$(pwd)/../drivers-tools"

if [ "Windows_NT" = "$OS" ]; then
    export GOPATH=$(cygpath -m $GOPATH)
    export GOCACHE=$(cygpath -m $GOCACHE)
    export DRIVERS_TOOLS=$(cygpath -m $DRIVERS_TOOLS)

    if [ ! -d "c:/libmongocrypt/include" ]; then
        mkdir -p c:/libmongocrypt/include
        mkdir -p c:/libmongocrypt/bin
        curl https://s3.amazonaws.com/mciuploads/libmongocrypt/windows/latest_release/libmongocrypt.tar.gz --output libmongocrypt.tar.gz
        tar -xvzf libmongocrypt.tar.gz
        cp ./bin/mongocrypt.dll c:/libmongocrypt/bin
        cp ./include/mongocrypt/*.h c:/libmongocrypt/include
        export PATH=$PATH:/cygdrive/c/libmongocrypt/bin
    fi
else
    if [ ! -d "libmongocrypt" ]; then
        git clone https://github.com/mongodb/libmongocrypt
        ./libmongocrypt/.evergreen/compile.sh
    fi
fi

export GOROOT="${GOROOT}"
export PATH="${GOROOT}/bin:${GCC_PATH}:$GOPATH/bin:$PATH"
export PROJECT="${project}"
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

if [ -z ${AWS_ACCESS_KEY_ID+x} ]; then
  export AWS_ACCESS_KEY_ID="${cse_aws_access_key_id}"
  export AWS_SECRET_ACCESS_KEY="${cse_aws_secret_access_key}"
fi

# Set temp credentials for AWS if python3 is available.
#
# Using python3-venv in Ubuntu 14.04 (an OS required for legacy server version
# tasks) requires the use of apt-get, which we wish to avoid. So, we do not set
# a python3 binary on Ubuntu 14.04. Setting AWS temp credentials for legacy
# server version tasks is unneccesary, as temp credentials are only needed on 4.2+.
if [ ! -z ${PYTHON3_BINARY} ]; then
  export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID}"
  export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY}"
  export AWS_DEFAULT_REGION="us-east-1"
  ${PYTHON3_BINARY} -m venv ./venv

  # Set the PYTHON environment variable to point to the active python3 binary. This is used by the
  # set-temp-creds.sh script.
  if [ "Windows_NT" = "$OS" ]; then
    export PYTHON="$(pwd)/venv/Scripts/python"
  else
    export PYTHON="$(pwd)/venv/bin/python"
  fi

  ./venv/${VENV_BIN_DIR:-bin}/pip3 install boto3
  . ${DRIVERS_TOOLS}/.evergreen/csfle/set-temp-creds.sh
fi

# If GO_BUILD_TAGS is not set, set the default Go build tags to "cse" to enable
# client-side encryption, which requires linking the libmongocrypt C library.
if [ -z ${GO_BUILD_TAGS+x} ]; then
  GO_BUILD_TAGS="cse"
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
BUILD_TAGS="-tags ${GO_BUILD_TAGS}" \
AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID}" \
AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY}" \
AWS_DEFAULT_REGION="us-east-1" \
CSFLE_AWS_TEMP_ACCESS_KEY_ID="$CSFLE_AWS_TEMP_ACCESS_KEY_ID" \
CSFLE_AWS_TEMP_SECRET_ACCESS_KEY="$CSFLE_AWS_TEMP_SECRET_ACCESS_KEY" \
CSFLE_AWS_TEMP_SESSION_TOKEN="$CSFLE_AWS_TEMP_SESSION_TOKEN" \
AZURE_TENANT_ID="${cse_azure_tenant_id}" \
AZURE_CLIENT_ID="${cse_azure_client_id}" \
AZURE_CLIENT_SECRET="${cse_azure_client_secret}" \
GCP_EMAIL="${cse_gcp_email}" \
GCP_PRIVATE_KEY="${cse_gcp_private_key}" \
make evg-test \
PKG_CONFIG_PATH=$PKG_CONFIG_PATH \
LD_LIBRARY_PATH=$LD_LIBRARY_PATH \
CSFLE_TLS_CA_FILE="$DRIVERS_TOOLS/.evergreen/x509gen/ca.pem" \
CSFLE_TLS_CERTIFICATE_KEY_FILE="$DRIVERS_TOOLS/.evergreen/x509gen/client.pem"

# Ensure mock KMS servers are running before starting tests.
if [ "$CLIENT_SIDE_ENCRYPTION" = "on" ]; then
  echo "Waiting for mock KMS servers to start..."
  wait_for_kms_server() {
    for i in $(seq 300); do
        # Exit code 7: "Failed to connect to host".
        if curl -s "localhost:$1"; test $? -ne 7; then
          return 0
        else
          sleep 1
        fi
    done
    echo "Could not detect mock KMS server on port $1"
    return 1
  }
  wait_for_kms_server 5698

  echo "Waiting for mock KMS servers to start... done."
  if ! test -d /cygdrive/c; then
    # We have trouble with this test on Windows. only set cryptSharedLibPath on other platforms
    export MONGOC_TEST_CRYPT_SHARED_LIB_PATH="$(find . -wholename '*src/libmongoc/mongo_crypt_v1.*' -and -regex '.*\(.dll\|.dylib\|.so\)' | head -n1)"
    echo "Setting env cryptSharedLibPath: [$MONGOC_TEST_CRYPT_SHARED_LIB_PATH]"
  fi
fi
