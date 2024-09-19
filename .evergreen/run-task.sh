#!/usr/bin/env bash
# 
# Source the env.sh file and run the given task
set -eux

source ./env.sh
task "$@"
