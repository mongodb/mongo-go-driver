#!/usr/bin/env bash
#
# Entry point for Dockerfile for launching running a go test.
#
set -eux

# Prep files.
cd /src
rm -f test.suite
cp -r $HOME/install ./install

# Run the test.
bash ./.evergreen/run-tests.sh
