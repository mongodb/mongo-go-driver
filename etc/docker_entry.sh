#!/usr/bin/env bash
#
# Entry point for Dockerfile for running a go test.
#
set -eux

# Prep files.
cd /src
rm -f test.suite
cp -r $HOME/install ./install

export PATH="$MONGODB_BINARIES:$HOME/go/bin:$PATH"

task setup-test

# Run the test.
task $TASKFILE_TARGET
