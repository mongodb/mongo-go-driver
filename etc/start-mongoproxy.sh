#!/usr/bin/env bash
#
# Set up environment and write env.sh and expansion.yml files.
set -eux

bash $DRIVERS_TOOLS/.evergreen/start-mongoproxy.sh
