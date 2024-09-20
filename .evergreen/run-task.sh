#!/usr/bin/env bash
# 
# Source the env.sh file and run the given task
set -eu

ls -a
source ./env.sh
task "$@"