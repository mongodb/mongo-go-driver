#!/usr/bin/env bash
# 
# Set up environment and write .env file.
set -eux

# Set Golang environment vars. GOROOT is wherever current Go distribution is; GOPATH is always 3
# directories up from pwd; GOCACHE is under .cache in the pwd.
if [ -z "${GOROOT:-}" ]; then 
    echo "Must set $GOROOT!"
    exit 1
fi

GOPATH="$(dirname "$(dirname "$(dirname "`pwd`")")")"
export GOPATH
GOCACHE="$(pwd)/.cache"
OS="${OS:-""}"
EXTRA_PATH="$GOROOT/bin:$$GOPATH/bin:${GCC_PATH:-}"

# Set other relevant variables for Evergreen processes.
DRIVERS_TOOLS_DEFAULT="$(dirname "$(dirname "$(dirname "`pwd`")")")/drivers-tools"
DRIVERS_TOOLS=${DRIVERS_TOOLS:-$DRIVERS_TOOLS_DEFAULT}
PROJECT_DIRECTORY="$(pwd)"

# If on Windows, convert paths with cygpath. GOROOT should not be converted as Windows expects it
# to be separated with '\'.
if [ "Windows_NT" = "$OS" ]; then
    GOPATH=$(cygpath -w $GOPATH)
    GOCACHE=$(cygpath -m $GOCACHE)
    DRIVERS_TOOLS=$(cygpath -m $DRIVERS_TOOLS)
    PROJECT_DIRECTORY=$(cygpath -m $PROJECT_DIRECTORY)
    # Use the --path option to convert the path on Windows.
    EXTRA_PATH=$(cypath -mp $EXTRA_PATH)

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

# Setup libmongocrypt.
task install-libmongocrypt
if [ "Windows_NT" = "$OS" ]; then
    EXTRA_PATH=$EXTRA_PATH:/cygdrive/c/libmongocrypt/bin
fi
PKG_CONFIG_PATH=$(pwd)/install/libmongocrypt/lib64/pkgconfig
LD_LIBRARY_PATH=$(pwd)/install/libmongocrypt/lib64

if [ "$(uname -s)" = "Darwin" ]; then
  PKG_CONFIG_PATH=$(pwd)/install/libmongocrypt/lib/pkgconfig
  DYLD_FALLBACK_LIBRARY_PATH=$(pwd)/install/libmongocrypt/lib
fi

cat <<EOT > .env
GOROOT="$GOROOT"
GOPATH="$GOPATH"
GOCACHE="$GOCACHE"
DRIVERS_TOOLS="$DRIVERS_TOOLS"
PROJECT_DIRECTORY="$PROJECT_DIRECTORY"
PKG_CONFIG_PATH="$PKG_CONFIG_PATH"
LD_LIBRARY_PATH="$LD_LIBRARY_PATH"
MACOS_LIBRARY_PATH="${DYLD_FALLBACK_LIBRARY_PATH:-}"
EXTRA_PATH="$EXTRA_PATH"
EOT

# TODO REMOVE
cat .test.env