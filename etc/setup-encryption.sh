#!/usr/bin/env bash
#
# Script to set up encryption assets and servers.
set -eux

if [ -z "$DRIVERS_TOOLS" ]; then
    echo "Please define DRIVERS_TOOLS variable"
    exit 1
fi

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
PARENT_DIR=$(dirname $SCRIPT_DIR)

# Handle the secrets
export CSFLE_TLS_CA_FILE="${PARENT_DIR}/testdata/kmip-certs/ca-ec.pem"
export CSFLE_TLS_CERT_FILE="${PARENT_DIR}/testdata/kmip-certs/server-ec.pem"
export CSFLE_TLS_CLIENT_CERT_FILE="${PARENT_DIR}/testdata/kmip-certs/client-ec.pem"

bash $DRIVERS_TOOLS/.evergreen/csfle/setup-secrets.sh
bash $DRIVERS_TOOLS/.evergreen/csfle/start-servers.sh
