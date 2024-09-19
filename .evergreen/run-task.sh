#!/usr/bin/env bash
# 
# Source the env.sh file and run the given task
set -eu

source ./env.sh
echo "GOCACHE1=${GOCACHE:-}"
cat env.sh
echo "GOCACHE2=$GOCACHE"
task "$@"
