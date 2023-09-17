#!/usr/bin/env bash
#
# Entry point for Dockerfile for launching a server and running a go test.
#
set -eux

# Handle env variables.
export DRIVERS_TOOLS=$HOME/drivers-evergreen-tools
export PROJECT_ORCHESTRATION_HOME=$DRIVERS_TOOLS/.evergreen/orchestration
export MONGO_ORCHESTRATION_HOME=$HOME

# Clone DRIVERS_TOOLS if necessary.
if [ ! -d $DRIVERS_TOOLS ]; then
    git clone https://github.com/mongodb-labs/drivers-evergreen-tools.git $DRIVERS_TOOLS
fi

# Disable ipv6.
sed -i "s/\"ipv6\": true,/\"ipv6\": false,/g" $PROJECT_ORCHESTRATION_HOME/configs/${TOPOLOGY}s/$ORCHESTRATION_FILE

# Start the server.
bash $DRIVERS_TOOLS/.evergreen/run-orchestration.sh

# Prep files.
cd /src
rm -f test.suite
cp -r $HOME/install ./install

# Run the test.
bash ./.evergreen/run-tests.sh
