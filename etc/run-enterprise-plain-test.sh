#!/usr/bin/env bash
# run-enterprise-gssapi-test
# Runs the enterprise auth tests with plain credentials.
set -eu
. ${DRIVERS_TOOLS}/.evergreen/secrets_handling/setup-secrets.sh drivers/enterprise_auth
export MONGODB_URI="mongodb://${SASL_USER}:${SASL_PASS}@${SASL_HOST}:${SASL_PORT}/ldap?authMechanism=PLAIN"
export MONGO_GO_DRIVER_COMPRESSOR="${MONGO_GO_DRIVER_COMPRESSOR:-}"
make -s evg-test-enterprise-auth
