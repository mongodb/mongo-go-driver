#!/usr/bin/env bash
# 
# Set up environment and write env.sh file.
set -eu

# Set Golang environment vars. GOROOT is wherever current Go distribution is; GOPATH is always 3
# directories up from pwd; GOCACHE is under .cache in the pwd.
if [ -z "${GOROOT:-}" ]; then 
    echo "Must set $GOROOT!"
    exit 1
fi
   
GOPATH="$(dirname "$(dirname "$(dirname "`pwd`")")")"
export GOPATH
GOCACHE="$(pwd)/.cache"
export GOCACHE
OS="${OS:-""}"
EXTRA_PATH="$GOROOT/bin:$GOPATH/bin:${GCC_PATH:-}"

# Set other relevant variables for Evergreen processes.
DRIVERS_TOOLS_DEFAULT="$(dirname "$(dirname "$(dirname "`pwd`")")")/drivers-tools"
DRIVERS_TOOLS=${DRIVERS_TOOLS:-$DRIVERS_TOOLS_DEFAULT}
PROJECT_DIRECTORY="$(pwd)"

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
    USERPROFILE=$(cygpath -w "$(dirname "$(dirname "$(dirname "`pwd`")")")")
    HOME=$USERPROFILE
fi

# Prepend the path.
PATH="$EXTRA_PATH:$PATH"

# Check Go installation.
go version
go env

# Install taskfile.
go install github.com/go-task/task/v3/cmd/task@latest

cat <<EOT > env.sh
GOROOT="$GOROOT"
GOPATH="$GOPATH"
DRIVERS_TOOLS="$DRIVERS_TOOLS"
PROJECT_DIRECTORY="$PROJECT_DIRECTORY"
PATH="$PATH"
EOT

if [ "Windows_NT" = "$OS" ]; then
    echo "USERPROFILE=$USERPROFILE" >> env.sh
    echo "HOME=$HOME" >> env.sh
fi