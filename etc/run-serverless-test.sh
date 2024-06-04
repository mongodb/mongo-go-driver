#!/usr/bin/env bash
# run-serverless-test
# Runs the serverless tests.
source ${DRIVERS_TOOLS}/.evergreen/serverless/secrets-export.sh
AUTH="auth" \
    SSL="ssl" \
    MONGODB_URI="${SERVERLESS_URI}" \
    IS_SERVERLESS_PROXY="${IS_SERVERLESS_PROXY}" \
    SERVERLESS="serverless" \
    MAKEFILE_TARGET=evg-test-serverless \
    sh ${PROJECT_DIRECTORY}/.evergreen/run-tests.sh
