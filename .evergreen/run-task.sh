#!/usr/bin/env bash
#
# Source the env.sh file and run the given task
set -exu

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
PROJECT_DIRECTORY=$(dirname $SCRIPT_DIR)
pushd ${PROJECT_DIRECTORY} >/dev/null

source env.sh
task "$@"

popd >/dev/null
