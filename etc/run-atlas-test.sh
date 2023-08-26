#!/usr/bin/env bash
# run-atlas-test
# Run atlas connectivity tests.
set -eux

bash etc/get_aws_secrets.sh drivers/atlas_connect
. ./secrets-export.sh

set +x  # Disable xtrace
echo "Running cmd/testatlas/main.go"
go run ./cmd/testatlas/main.go "$ATLAS_REPL" "$ATLAS_SHRD" "$ATLAS_FREE" "$ATLAS_TLS11" "$ATLAS_TLS12" "$ATLAS_SERVERLESS" "$ATLAS_SRV_REPL" "$ATLAS_SRV_SHRD" "$ATLAS_SRV_FREE" "$ATLAS_SRV_TLS11" "$ATLAS_SRV_TLS12" "$ATLAS_SRV_SERVERLESS"
