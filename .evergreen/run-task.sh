#!/usr/bin/env bash
# 
# Source the env.sh file and run the given task
set -eu

source ./env.sh
task "$@"
