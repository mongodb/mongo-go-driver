#!/bin/bash
set -eu

if [ "${SKIP_ECS_AUTH_TEST:-}" = "true" ]; then
    echo "This platform does not support the ECS auth test, skipping..."
    exit 0
fi

task build-aws-ecs-test
    
AUTH_AWS_DIR=${DRIVERS_TOOLS}/.evergreen/auth_aws
ECS_SRC_DIR=$AUTH_AWS_DIR/src

# pack up project directory to ssh it to the container
mkdir -p $ECS_SRC_DIR
tar -czf $ECS_SRC_DIR/src.tgz -C ${PROJECT_DIRECTORY} .
PROJECT_DIRECTORY="${ECS_SRC_DIR}" $AUTH_AWS_DIR/aws_setup.sh ecs