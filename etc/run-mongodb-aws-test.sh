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

# Run the MONGODB-AWS prose tests. Exactly one AWS_TEST scenario is live per
# invocation; the rest skip.
#
# Run from PROJECT_DIRECTORY with "go test -C" so test.suite is written at the
# repository root. The gotest.parse_files glob in .evergreen/config.yml only
# matches "src/go.mongodb.org/mongo-driver/*.suite", so a suite file written
# inside the module directory is never parsed and the Evergreen Tests tab is
# empty.
(cd ${PROJECT_DIRECTORY} && go test -C ./internal/test/awsauth -timeout 30m -v ./... | tee -a test.suite)
