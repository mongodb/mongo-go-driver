#!/usr/bin/env bash
#
# Entry point for Dockerfile for launching a server and running a go test.
#
set -eux

# Start the server.
bash $DRIVERS_TOOLS/.evergreen/run-orchestration.sh

# Prep files.
cd /src
rm -f test.suite
cp -r $HOME/install ./install

# Run the test.
bash ./.evergreen/run-tests.sh
