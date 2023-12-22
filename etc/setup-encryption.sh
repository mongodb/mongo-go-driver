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

if [ "Windows_NT" = "$OS" ]; then
  export CSFLE_TLS_CA_FILE=$(cygpath -m $CSFLE_TLS_CA_FILE)
  export CSFLE_TLS_CERT_FILE=$(cygpath -m $CSFLE_TLS_CERT_FILE)
  export CSFLE_TLS_CLIENT_CERT_FILE=$(cygpath -m $CSFLE_TLS_CLIENT_CERT_FILE)
fi

bash $DRIVERS_TOOLS/.evergreen/csfle/setup_secrets.sh

# Map some of the secrets to expected environment variables.
source ./secrets-export.sh
echo "export AZURE_TENANT_ID=\"$FLE_AZURE_TENANTID\"" >> secrets-export.sh
echo "export AZURE_CLIENT_ID=\"$FLE_AZURE_CLIENTID\"" >> secrets-export.sh
echo "export AZURE_CLIENT_SECRET=\"$FLE_AZURE_CLIENTSECRET\"" >> secrets-export.sh
echo "export GCP_EMAIL=\"$FLE_GCP_EMAIL\"" >> secrets-export.sh
echo "export GCP_PRIVATE_KEY=\"$FLE_GCP_PRIVATEKEY\"" >> secrets-export.sh
echo "export CSFLE_TLS_CA_FILE=\"$CSFLE_TLS_CA_FILE\"" >> secrets-export.sh
echo "export CSFLE_TLS_CERT_FILE=\"$CSFLE_TLS_CERT_FILE\"" >> secrets-export.sh

# Start the servers.
bash $DRIVERS_TOOLS/.evergreen/csfle/start_servers.sh
