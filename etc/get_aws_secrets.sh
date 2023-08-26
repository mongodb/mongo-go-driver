#!/usr/bin/env bash
# get-aws-secrets
# Gets AWS secrets from the vault
set -eu

if [ -z "$DRIVERS_TOOLS" ]; then
    echo "Please define DRIVERS_TOOLS variable"
    exit 1
fi

pushd ${DRIVERS_TOOLS}/.evergreen/auth_aws
. ./activate-authawsvenv.sh
popd

# TODO: Add a section to https://wiki.corp.mongodb.com/display/DRIVERS/Using+AWS+Secrets+Manager+to+Store+Testing+Secrets
# about using a bash script that can be run locally
# Add this note:
# Note: for local testing using AWS SSO credentials,
# you may need to set the AWS_PROFILE environment variable
# to point to your local profile name.
echo "Getting secrets: $@"
python ${DRIVERS_TOOLS}/.evergreen/auth_aws/setup_secrets.py $@
echo "Got secrets"
