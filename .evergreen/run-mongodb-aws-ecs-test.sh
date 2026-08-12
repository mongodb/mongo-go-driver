#!/bin/bash

set -o errexit  # Exit the script with error if any of the commands fail

############################################
#            Main Program                  #
############################################

if [[ -z "$1" ]]; then
    echo "usage: $0 <MONGODB_URI>"
    exit 1
fi
export MONGODB_URI="$1"

echo "Running MONGODB-AWS ECS authentication tests"

if echo "$MONGODB_URI" | grep -q "@"; then
  echo "MONGODB_URI unexpectedly contains user credentials in ECS test!";
  exit 1
fi

# Run only the ECS scenario. This container runs the test binary directly rather
# than through etc/run-mongodb-aws-test.sh, so it has to apply the same filter
# that script does; without it every other scenario runs here and fails for want
# of credentials this environment does not provide.
./src/main -test.run TestAWSProse_3_ECSCredentials -test.v
