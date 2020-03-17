#!/bin/bash

set -o xtrace
set -o errexit  # Exit the script with error if any of the commands fail

############################################
#            Main Program                  #
############################################

# Supported/used environment variables:
#  MONGODB_URI    Set the URI, including an optional username/password to use
#                 to connect to the server via MONGODB-AWS authentication
#                 mechanism.

echo "Running MONGODB-AWS authentication tests"
# ensure no secrets are printed in log files
set +x

# load the script
shopt -s expand_aliases # needed for `urlencode` alias
[ -s "${PROJECT_DIRECTORY}/prepare_mongodb_aws.sh" ] && source "${PROJECT_DIRECTORY}/prepare_mongodb_aws.sh"

MONGODB_URI=${MONGODB_URI:-"mongodb://localhost"}
MONGODB_URI="${MONGODB_URI}/aws?authMechanism=MONGODB-AWS"
if [[ -n ${SESSION_TOKEN} ]]; then
    MONGODB_URI="${MONGODB_URI}&authMechanismProperties=AWS_SESSION_TOKEN:${SESSION_TOKEN}"
fi

export MONGODB_URI="$MONGODB_URI"

# show test output
set -x

go run "${PROJECT_DIRECTORY}/mongo/testaws/main.go"
