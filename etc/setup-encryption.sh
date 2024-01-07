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

bash $DRIVERS_TOOLS/.evergreen/csfle/setup_secrets.sh

# Map some of the secrets to expected environment variables.
source ./secrets-export.sh
cat <<EOT >> ./secrets-export.sh

export AZURE_TENANT_ID="$FLE_AZURE_TENANTID"
export AZURE_CLIENT_ID="$FLE_AZURE_CLIENTID"
export AZURE_CLIENT_SECRET="$FLE_AZURE_CLIENTSECRET"
export GCP_EMAIL="$FLE_GCP_EMAIL"
export GCP_PRIVATE_KEY="$FLE_GCP_PRIVATEKEY"

EOT

bash $DRIVERS_TOOLS/.evergreen/csfle/start_servers.sh
