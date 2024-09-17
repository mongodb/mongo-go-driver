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
GOCACHE="$(pwd)/.cache"

# Set other relevant variables for Evergreen processes.
DRIVERS_TOOLS_DEFAULT="$(dirname "$(dirname "$(dirname "`pwd`")")")/drivers-tools"
DRIVERS_TOOLS=${DRIVERS_TOOLS:-$DRIVERS_TOOLS_DEFAULT}
PROJECT_DIRECTORY="$(pwd)"

# If on Windows, convert paths with cygpath. GOROOT should not be converted as Windows expects it
# to be separated with '\'.
if [ "Windows_NT" = "$OS" ]; then
    GOPATH=$(cygpath -m $GOPATH)
    GOCACHE=$(cygpath -m $GOCACHE)
    DRIVERS_TOOLS=$(cygpath -m $DRIVERS_TOOLS)
    PROJECT_DIRECTORY=$(cygpath -m $PROJECT_DIRECTORY)

    # Set home variables for Windows, too.
    USERPROFILE=$(cygpath -w "$(dirname "$(dirname "$(dirname "`pwd`")")")")
    HOME=$USERPROFILE
fi

# Set up extra path needed for running executables
EXTRA_PATH=""
GOROOTBIN="$GOROOT/bin"
 GOPATHBIN="$GOPATH/bin"
if [ "Windows_NT" = "$OS" ]; then
    # Convert all Windows-style paths (e.g. C:/) to Bash-style Cygwin paths
    # (e.g. /cygdrive/c/...) because PATH is interpreted by Bash, which uses ":" as a
    # separator so doesn't support Windows-style paths. Other scripts or binaries that
    # aren't part of Cygwin still need the environment variables to use Windows-style
    # paths, so only convert them when setting PATH. Note that GCC_PATH is already a
    # Bash-style Cygwin path for all Windows tasks.
    EXTRA_PATH="$(cygpath $GOROOTBIN):$(cygpath $GOPATHBIN):${GCC_PATH:-}:$EXTRA_PATH"
else
    EXTRA_PATH="$GOROOTBIN:$GOPATHBIN:${GCC_PATH:-}::$EXTRA_PATH"
fi

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