#!/usr/bin/env bash
# run-gcpkms-test
# Runs gcpkms tests.
set -eu

GO_BUILD_TAGS="cse" task setup-test
task build-kms-test

if [ -n "${EXPECT_ERROR:-}" ]; then
  LD_LIBRARY_PATH=./install/libmongocrypt/lib64 \
  MONGODB_URI='mongodb://localhost:27017/' \
  EXPECT_ERROR='unable to retrieve GCP credentials' \
  PROVIDER='gcp' \
    ./testkms
  exit 0
fi

source ${DRIVERS_TOOLS}/.evergreen/csfle/gcpkms/secrets-export.sh
echo "Copying files ... begin"
tar czf testgcpkms.tgz ./testkms ./install/libmongocrypt/lib64/libmongocrypt.*
GCPKMS_SRC=testgcpkms.tgz GCPKMS_DST=$GCPKMS_INSTANCENAME: ${DRIVERS_TOOLS}/.evergreen/csfle/gcpkms/copy-file.sh
echo "Copying files ... end"

echo "Untarring file ... begin"
GCPKMS_CMD="tar xf testgcpkms.tgz" ${DRIVERS_TOOLS}/.evergreen/csfle/gcpkms/run-command.sh
echo "Untarring file ... end"

GCPKMS_CMD="LD_LIBRARY_PATH=./install/libmongocrypt/lib64 MONGODB_URI='mongodb://localhost:27017' PROVIDER='gcp' ./testkms" ${DRIVERS_TOOLS}/.evergreen/csfle/gcpkms/run-command.sh
