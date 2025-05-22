#!/bin/bash

set -eu

############################################
#            Main Program                  #
############################################

# Supported/used environment variables:
#  MONGODB_URI    Set the URI, including an optional username/password to use
#                 to connect to the server via MONGODB-AWS authentication
#                 mechanism.

echo "Running MONGODB-AWS authentication tests"

if [ "$AWS_TEST" == "ec2" ] && [ "${SKIP_EC2_AUTH_TEST:-}" == "true" ]; then
  echo "This platform does not support the EC2 auth test, skipping..."
  exit 0
fi

if [ "$AWS_TEST" == "web-identity" ] && [ "${SKIP_WEB_IDENTITY_AUTH_TEST:-}" == "true" ]; then
  echo "This platform does not support the web identity auth test, skipping..."
  exit 0
fi

# Handle credentials and environment setup.
. $DRIVERS_TOOLS/.evergreen/auth_aws/aws_setup.sh $AWS_TEST

# show test output
set -x

# For Go 1.16+, Go builds requires a go.mod file in the current working directory or a parent
# directory. Spawn a new subshell, "cd" to the project directory, then run "go run".
(cd ${PROJECT_DIRECTORY} && go run "./internal/cmd/testaws/main.go" | tee test.suite)
(cd ${PROJECT_DIRECTORY} && go run "./internal/cmd/testmongoaws/main.go" | tee test.suite)
