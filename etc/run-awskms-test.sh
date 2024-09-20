#!/usr/bin/env bash
# run-awskms-test
# Runs the awskms test.
set -eu

GO_BUILD_TAGS="cse" task setup-test
task build-kms-test

. ${DRIVERS_TOOLS}/.evergreen/secrets_handling/setup-secrets.sh drivers/atlas_connect
export MONGODB_URI="$ATLAS_FREE"

if [ -z "${EXPECT_ERROR:-}" ]; then
    . ${DRIVERS_TOOLS}/.evergreen/csfle/setup-secrets.sh
    export AWS_SECRET_ACCESS_KEY=$FLE_AWS_SECRET_ACCESS_KEY
    export AWS_ACCESS_KEY_ID=$FLE_AWS_ACCESS_KEY_ID
fi

LD_LIBRARY_PATH=./install/libmongocrypt/lib64 PROVIDER='aws' ./testkms
