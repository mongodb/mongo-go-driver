#!/bin/sh
# PKG_CONFIG wrapper for locally installed libmongocrypt.
#
# The .pc file produced by install-libmongocrypt.sh bakes in a temp build
# directory as prefix=, which no longer exists after the build completes.
# This wrapper overrides prefix= with the actual install location so cgo
# can find the headers and libraries without editing the .pc file.
#
# Usage: export PKG_CONFIG=$(pwd)/etc/libmongocrypt-pkg-config.sh
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
exec pkg-config --define-variable=prefix="${REPO_ROOT}/install/libmongocrypt" "$@"
