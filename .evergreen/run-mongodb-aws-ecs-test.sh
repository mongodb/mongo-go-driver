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

# The tests select their scenario from AWS_TEST. It is set on the Evergreen host
# but is not propagated into the ECS container, so set it here; without it every
# scenario skips and the task passes without testing anything.
export AWS_TEST=ecs

./src/main
