#!/bin/bash
#
set -o errexit  # Exit the script with error if any of the commands fail.

VARLIST=(
	AWS_REGION
	DRIVERS_TOOLS
	DRIVERS_ATLAS_PUBLIC_API_KEY
	DRIVERS_ATLAS_PRIVATE_API_KEY
	DRIVERS_ATLAS_LAMBDA_USER
	DRIVERS_ATLAS_LAMBDA_PASSWORD
	DRIVERS_ATLAS_GROUP_ID
	LAMBDA_STACK_NAME
	PROJECT_DIRECTORY
	TEST_LAMBDA_DIRECTORY
)

# Ensure that all variables required to run the test are set, otherwise throw
# an error.
for VARNAME in ${VARLIST[*]}; do
	[[ -z "${!VARNAME}" ]] && echo "ERROR: $VARNAME not set" && exit 1;
done

echo "Starting deployment"
. ${DRIVERS_TOOLS}/.evergreen/aws_lambda/run-deployed-lambda-aws-tests.sh
