#!/usr/bin/env bash
#
# Set up environment and write env.sh and expansion.yml files.
set -eu

# Set up default environment variables.
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
PROJECT_DIRECTORY=$(dirname $SCRIPT_DIR)
pushd $PROJECT_DIRECTORY
ROOT_DIR=$(dirname $PROJECT_DIRECTORY)
DRIVERS_TOOLS=${DRIVERS_TOOLS:-${ROOT_DIR}/drivers-evergreen-tools}
MONGO_ORCHESTRATION_HOME="${DRIVERS_TOOLS}/.evergreen/orchestration"
MONGODB_BINARIES="${DRIVERS_TOOLS}/mongodb/bin"
OS="${OS:-""}"

# Set Golang environment vars. GOROOT is wherever current Go distribution is, and is set in evergreen config.
# GOPATH is always 3 directories up from pwd on EVG; GOCACHE is under .cache in the pwd.
GOROOT=${GOROOT:-$(dirname "$(dirname "$(which go)")")}
export GOPATH=${GOPATH:-$ROOT_DIR}
export GOCACHE="${GO_CACHE:-$PROJECT_DIRECTORY/.cache}"

# Handle paths on Windows.
if [ "Windows_NT" = "${OS:-}" ]; then # Magic variable in cygwin
  GOPATH=$(cygpath -m $GOPATH)
  GOCACHE=$(cygpath -w $GOCACHE)
  DRIVERS_TOOLS=$(cygpath -m $DRIVERS_TOOLS)
  PROJECT_DIRECTORY=$(cygpath -m $PROJECT_DIRECTORY)
  EXTRA_PATH=/cygdrive/c/libmongocrypt/bin
  MONGO_ORCHESTRATION_HOME=$(cygpath -m $MONGO_ORCHESTRATION_HOME)
  MONGODB_BINARIES=$(cygpath -m $MONGODB_BINARIES)
  # Set home variables for Windows, too.
  USERPROFILE=$(cygpath -w "$ROOT_DIR")
  HOME=$USERPROFILE
else
  EXTRA_PATH=${GCC:-}
fi

# Add binaries to the path.
PATH="${GOROOT}/bin:${GOPATH}/bin:${MONGODB_BINARIES}:${EXTRA_PATH}:${PATH}"

# Get the current unique version of this checkout.
if [ "${IS_PATCH:-}" = "true" ]; then
    CURRENT_VERSION=$(git describe)-patch-${VERSION_ID}
else
    CURRENT_VERSION=latest
fi

# Ensure a checkout of drivers-tools.
if [ ! -d "$DRIVERS_TOOLS" ]; then
  git clone https://github.com/mongodb-labs/drivers-evergreen-tools $DRIVERS_TOOLS
fi

# Write the .env file for drivers-tools.
cat <<EOT > ${DRIVERS_TOOLS}/.env
SKIP_LEGACY_SHELL=1
DRIVERS_TOOLS="$DRIVERS_TOOLS"
MONGO_ORCHESTRATION_HOME="$MONGO_ORCHESTRATION_HOME"
MONGODB_BINARIES="$MONGODB_BINARIES"
TMPDIR="$MONGO_ORCHESTRATION_HOME/db"
EOT

# Check Go installation.
go version
go env

# Install taskfile.
go install github.com/go-task/task/v3/cmd/task@v3.39.1

# Write our own env file.
cat <<EOT > env.sh
export GOROOT="$GOROOT"
export GOPATH="$GOPATH"
export GOCACHE="$GOCACHE"
export DRIVERS_TOOLS="$DRIVERS_TOOLS"
export PROJECT_DIRECTORY="$PROJECT_DIRECTORY"
export MONGODB_BINARIES="$MONGODB_BINARIES"
export PATH="$PATH"
EOT

if [ "Windows_NT" = "$OS" ]; then
    echo "export USERPROFILE=$USERPROFILE" >> env.sh
    echo "export HOME=$HOME" >> env.sh
fi

# source the env.sh file and write the expansion file.
cat <<EOT > expansion.yml
CURRENT_VERSION: "$CURRENT_VERSION"
DRIVERS_TOOLS: "$DRIVERS_TOOLS"
PROJECT_DIRECTORY: "$PROJECT_DIRECTORY"
RUN_TASK: "$PROJECT_DIRECTORY/.evergreen/run-task.sh"
EOT

cat env.sh
popd
