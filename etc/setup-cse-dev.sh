#!/usr/bin/env bash
# setup-cse-dev.sh — set up the local environment for CSE/CSFLE integration testing.
#
# Installs libmongocrypt from source into install/libmongocrypt, downloads a
# host-platform crypt_shared library for automatic-encryption query analysis,
# and exports the variables needed to build and run CSE tests without a
# system-wide libmongocrypt installation (e.g. Homebrew).
#
# Must be sourced so that exports reach the calling shell:
#
#   source etc/setup-cse-dev.sh
#
# Requirements:
#   - DRIVERS_TOOLS must point at a clone of drivers-evergreen-tools (used to
#     download crypt_shared and to load KMS secrets).
#   - CRYPT_SHARED_VERSION selects the crypt_shared version to download
#     (default "latest"). It must be >= the query types you exercise: prefix
#     and suffix require 9.0+, substring requires 8.2+.

if [ -z "${DRIVERS_TOOLS:-}" ]; then
  echo "ERROR: DRIVERS_TOOLS is not set; point it at a clone of drivers-evergreen-tools." >&2
  return 1
fi

if [ ! -d "${DRIVERS_TOOLS}" ]; then
  echo "ERROR: DRIVERS_TOOLS does not exist: ${DRIVERS_TOOLS}" >&2
  return 1
fi

task --force install-libmongocrypt || { echo "ERROR: install-libmongocrypt failed." >&2; return 1; }

if [ "$(uname -s)" = "Darwin" ]; then
  libdir="$(pwd)/install/libmongocrypt/lib"
  PKG_CONFIG_PATH="${libdir}/pkgconfig"
else
  libdir="$(pwd)/install/libmongocrypt/lib64"
  PKG_CONFIG_PATH="${libdir}/pkgconfig"
fi
export PKG_CONFIG_PATH

PKG_CONFIG="$(pwd)/etc/libmongocrypt-pkg-config.sh"
export PKG_CONFIG

# The locally built libmongocrypt dylib resolves via @rpath but the cgo build
# adds no LC_RPATH, so at runtime the loader would either fail to find it or
# fall back to a system-wide copy (e.g. Homebrew), silently diverging the
# build-time and run-time libmongocrypt versions. DYLD_LIBRARY_PATH is not a
# reliable fix because macOS SIP strips DYLD_* env vars when `go test` execs
# the test binary. Baking an rpath into the binary via CGO_LDFLAGS resolves
# @rpath deterministically with no env dependency, and because CGO_LDFLAGS is
# part of the build cache key it also forces a relink after a version bump.
# Append (don't clobber) so any caller-provided CGO_LDFLAGS is preserved, and
# only add the rpath if it isn't already present so re-sourcing this script
# doesn't accumulate duplicate flags (which would also churn the build cache key).
rpath_flag="-Wl,-rpath,${libdir}"
if [[ " ${CGO_LDFLAGS:-} " != *" ${rpath_flag} "* ]]; then
  CGO_LDFLAGS="${CGO_LDFLAGS:+${CGO_LDFLAGS} }${rpath_flag}"
fi
export CGO_LDFLAGS

# Download a crypt_shared library matching the host platform. This is the
# automatic-encryption query analyzer used by auto-encrypting clients; without
# it the driver spawns mongocryptd, which may be a stale system copy that does
# not recognize newer query types (e.g. prefix/suffix). crypt_shared and
# mongocryptd ship with the enterprise server, NOT with libmongocrypt, so it is
# downloaded separately here. mongodl auto-detects the host OS and architecture.
CRYPT_SHARED_VERSION="${CRYPT_SHARED_VERSION:-latest}"
cryptSharedOut="$(pwd)/install/crypt_shared"
rm -rf "${cryptSharedOut}"
python3 "${DRIVERS_TOOLS}/.evergreen/mongodl.py" \
  --component crypt_shared \
  --version "${CRYPT_SHARED_VERSION}" \
  --out "${cryptSharedOut}" \
  --strip-path-components 1
CRYPT_SHARED_LIB_PATH="$(find "${cryptSharedOut}" -name 'mongo_crypt_v1.*' | head -1)"
export CRYPT_SHARED_LIB_PATH

echo "CSE environment ready."
echo "  PKG_CONFIG_PATH=${PKG_CONFIG_PATH}"
echo "  PKG_CONFIG=${PKG_CONFIG}"
echo "  CRYPT_SHARED_LIB_PATH=${CRYPT_SHARED_LIB_PATH}"
echo ""
echo "Run CSE tests with:"
echo "  go test -tags cse ./internal/integration -run <TestName>"

if [ -n "${AWS_PROFILE:-}" ]; then
  aws sso login --profile "${AWS_PROFILE}"
else
  aws sso login
fi

if ! bash "${DRIVERS_TOOLS}/.evergreen/csfle/setup-secrets.sh"; then
  echo "ERROR: setup-secrets.sh failed; KMS secrets not loaded." >&2
  return 1
fi

# shellcheck source=/dev/null
if ! source secrets-export.sh; then
  echo "ERROR: failed to source secrets-export.sh; KMS secrets not loaded." >&2
  return 1
fi
echo "KMS secrets loaded."
