#!/usr/bin/env bash
#
# Script to wait for the kmip servers to start
set -eux

if [ -z "$DRIVERS_TOOLS" ]; then
    echo "Please define DRIVERS_TOOLS variable"
    exit 1
fi

source ./etc/secrets-export.sh
bash $DRIVERS_TOOLS/.evergreen/csfle/await_servers.sh
