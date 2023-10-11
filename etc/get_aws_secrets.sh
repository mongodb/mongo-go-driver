#!/usr/bin/env bash
# get-aws-secrets
# Gets AWS secrets from the vault
set -eu

if [ -z "$DRIVERS_TOOLS" ]; then
    echo "Please define DRIVERS_TOOLS variable"
    exit 1
fi

bash $DRIVERS_TOOLS/.evergreen/auth_aws/setup_secrets.sh "$@"
. ./secrets-export.sh
