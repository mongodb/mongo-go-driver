#!/usr/bin/env bash
# run-gcpkms-test
# Runs gcpkms tests.
set -eu

GO_BUILD_TAGS="cse" task setup-test
task build-kms-test

if [ -n "${EXPECT_ERROR:-}" ]; then
  . ${DRIVERS_TOOLS}/.evergreen/csfle/azurekms/setup-secrets.sh
  LD_LIBRARY_PATH=./install/libmongocrypt/lib64 \
  MONGODB_URI='mongodb://localhost:27017' \
  EXPECT_ERROR='unable to retrieve azure credentials' \
  PROVIDER='azure' AZUREKMS_KEY_NAME=$AZUREKMS_KEYNAME AZUREKMS_KEY_VAULT_ENDPOINT=$AZUREKMS_KEYVAULTENDPOINT \
    ./testkms
  exit 0
fi

echo "Copying files ... begin"
source ${DRIVERS_TOOLS}/.evergreen/csfle/azurekms/secrets-export.sh
tar czf testazurekms.tgz ./testkms ./install/libmongocrypt/lib64/libmongocrypt.*
AZUREKMS_SRC=testazurekms.tgz AZUREKMS_DST=/tmp ${DRIVERS_TOOLS}/.evergreen/csfle/azurekms/copy-file.sh
echo "Copying files ... end"
echo "Untarring file ... begin"
AZUREKMS_CMD="tar xf /tmp/testazurekms.tgz" ${DRIVERS_TOOLS}/.evergreen/csfle/azurekms/run-command.sh
echo "Untarring file ... end"

AZUREKMS_CMD="LD_LIBRARY_PATH=./install/libmongocrypt/lib64 MONGODB_URI='mongodb://localhost:27017' PROVIDER='azure' AZUREKMS_KEY_NAME=$AZUREKMS_KEYNAME AZUREKMS_KEY_VAULT_ENDPOINT=$AZUREKMS_KEYVAULTENDPOINT ./testkms" ${DRIVERS_TOOLS}/.evergreen/csfle/azurekms/run-command.sh
