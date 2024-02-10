#!/usr/bin/env bash
# run-enterprise-gssapi-test
# Runs the enterprise auth tests with gssapi credentials.
set -eu

. ${DRIVERS_TOOLS}/.evergreen/secrets_handling/setup-secrets.sh drivers/enterprise_auth
if [ "Windows_NT" = "${OS:-}" ]; then
    export MONGODB_URI="mongodb://${PRINCIPAL/@/%40}:${SASL_PASS}@${SASL_HOST}:${SASL_PORT}/kerberos?authMechanism=GSSAPI"
else
    echo "${KEYTAB_BASE64}" > /tmp/drivers.keytab.base64
    base64 --decode /tmp/drivers.keytab.base64 > ${PROJECT_DIRECTORY}/.evergreen/drivers.keytab
    mkdir -p ~/.krb5
    cat .evergreen/krb5.config | tee -a ~/.krb5/config
    kinit -k -t ${PROJECT_DIRECTORY}/.evergreen/drivers.keytab -p "${PRINCIPAL}"
    export MONGODB_URI="mongodb://${PRINCIPAL/@/%40}@${SASL_HOST}:${SASL_PORT}/kerberos?authMechanism=GSSAPI"
fi
export MONGO_GO_DRIVER_COMPRESSOR="${MONGO_GO_DRIVER_COMPRESSOR:-}"

make -s evg-test-enterprise-auth
