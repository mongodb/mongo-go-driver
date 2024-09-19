#!/usr/bin/env bash
# 
# Set up environment and write env.sh and expansion.yml files.
set -eu

# Get the current unique version of this checkout.
if [ "${IS_PATCH}" = "true" ]; then
    CURRENT_VERSION=$(git describe)-patch-${VERSION_ID}
else
    CURRENT_VERSION=latest
fi

# Set general variables.
HERE=$(pwd)
ROOT_PATH="$(dirname "$(dirname "$(dirname "$HERE")")")"
OS="${OS:-""}"

# Set Golang environment vars. GOROOT is wherever current Go distribution is, and is set in evergreen config.
# GOPATH is always 3 directories up from pwd; GOCACHE is under .cache in the pwd.
GOPATH=$ROOT_PATH
export GOPATH
GOCACHE="$HERE/.cache"
export GOCACHE

# Set other relevant variables for Evergreen processes.
DRIVERS_TOOLS="$ROOT_PATH/drivers-tools"
PROJECT_DIRECTORY="$HERE"

# If on Windows, convert paths with cygpath. GOROOT should not be converted as Windows expects it
# to be separated with '\'.
if [ "Windows_NT" = "$OS" ]; then
    GOPATH=$(cygpath -m $GOPATH)
    GOCACHE=$(cygpath -w $GOCACHE)
    DRIVERS_TOOLS=$(cygpath -m $DRIVERS_TOOLS)
    PROJECT_DIRECTORY=$(cygpath -m $PROJECT_DIRECTORY)
    # Convert all Windows-style paths (e.g. C:/) to Bash-style Cygwin paths
    # (e.g. /cygdrive/c/...) because PATH is interpreted by Bash, which uses ":" as a
    # separator so doesn't support Windows-style paths. Other scripts or binaries that
    # aren't part of Cygwin still need the environment variables to use Windows-style
    # paths, so only convert them when setting PATH. Note that GCC_PATH is already a
    # Bash-style Cygwin path for all Windows tasks.
    EXTRA_PATH="$(cygpath $GOROOT/bin):$(cygpath $GOPATH/bin):${GCC_PATH:-}:/cygdrive/c/libmongocrypt/bin"

    # Set home variables for Windows, too.
    USERPROFILE=$(cygpath -w "$ROOT_PATH")
    HOME=$USERPROFILE
else 
    EXTRA_PATH="$GOROOT/bin:$GOPATH/bin:${GCC_PATH:-}"
fi

# Prepend the path.
PATH="$EXTRA_PATH:$PATH"

# Check Go installation.
go version
go env

# Install taskfile.
go install github.com/go-task/task/v3/cmd/task@v3.39.1

cat <<EOT > env.sh
export GOROOT="$GOROOT"
export GOPATH="$GOPATH"
export GOCACHE="$GOCACHE"
export DRIVERS_TOOLS="$DRIVERS_TOOLS"
export PROJECT_DIRECTORY="$PROJECT_DIRECTORY"
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
RUN_TASK: "$HERE/.evergreen/run-task.sh"
EOT

cat env.sh