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

# The prose test for this scenario. The aws-auth-test task invokes this script
# once per scenario, so without a filter every invocation would report the five
# tests it skipped alongside the one it ran. The tests still check AWS_TEST
# themselves; this only keeps the skips out of the Evergreen Tests tab.
case "$AWS_TEST" in
  regular)                 RUN_TEST=TestAWSProse_1_RegularCredentials ;;
  ec2)                     RUN_TEST=TestAWSProse_2_EC2Credentials ;;
  ecs)                     RUN_TEST=TestAWSProse_3_ECSCredentials ;;
  assume-role)             RUN_TEST=TestAWSProse_4_AssumeRole ;;
  web-identity)            RUN_TEST=TestAWSProse_5_AssumeRoleWithWebIdentity ;;
  env-creds|session-creds) RUN_TEST=TestAWSProse_6_AWSLambda ;;
  *)
    echo "unknown AWS_TEST scenario: $AWS_TEST"
    exit 1
    ;;
esac

# Handle credentials and environment setup.
. $DRIVERS_TOOLS/.evergreen/auth_aws/aws_setup.sh $AWS_TEST

# show test output
set -x

# Run from PROJECT_DIRECTORY with "go test -C" so test.suite is written at the
# repository root. The gotest.parse_files glob in .evergreen/config.yml only
# matches "src/go.mongodb.org/mongo-driver/*.suite", so a suite file written
# inside the module directory is never parsed and the Evergreen Tests tab is
# empty.
(cd ${PROJECT_DIRECTORY} && go test -C ./internal/test/awsauth -timeout 30m -v -run "$RUN_TEST" ./... | tee -a test.suite)
