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
# It also logs into AWS SSO and loads the KMS secrets, both inside a container,
# so no host AWS CLI, AWS profile or Python environment is needed. The login is
# interactive the first time: approve the printed URL and code.
#
# Requirements:
#   - Docker, for the AWS SSO login and the KMS secrets fetch.
#   - DRIVERS_TOOLS must point at a clone of drivers-evergreen-tools (used to
#     download crypt_shared).
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

# This replaces a host "aws sso login" plus drivers-evergreen-tools'
# setup-secrets.sh. Both needed host setup that this does not: a configured
# AWS profile, a host AWS CLI, and a Python 3.10+ venv for boto3. The image
# hardcodes the SSO settings and fetches drivers/csfle with the AWS CLI it
# already ships, so a clean machine needs only Docker.
#
# The login is interactive the first time: it prints a verification URL and a
# code to approve. The SSO token cache is shared with the host, so later runs
# reuse a live session.
proseDir="$(pwd)/internal/test/prose"

if ! docker build -q -t aws-sso-login -f "${proseDir}/docker/aws-sso-login.Dockerfile" "${proseDir}/docker"; then
  echo "ERROR: failed to build the aws-sso-login image; KMS secrets not loaded." >&2
  return 1
fi

# Write the credentials to a temporary directory rather than the repo, so they
# are not left lying around after the shell exits.
secretsDir="$(mktemp -d)"

ssoCacheDir="${HOME}/.aws/sso/cache"
mkdir -p "${ssoCacheDir}"

if ! docker run --rm -i \
  -v "${ssoCacheDir}:/root/.aws/sso/cache" \
  -v "${secretsDir}:/secrets" \
  -e "SECRET_VAULTS=drivers/csfle" \
  ${AWS_PROFILE:+-e "AWS_PROFILE=${AWS_PROFILE}"} \
  aws-sso-login; then
  rm -rf "${secretsDir}"
  echo "ERROR: the AWS SSO login container failed; KMS secrets not loaded." >&2
  return 1
fi

# shellcheck source=/dev/null
set -a
if ! source "${secretsDir}/secrets-export.sh"; then
  set +a
  rm -rf "${secretsDir}"
  echo "ERROR: failed to source secrets-export.sh; KMS secrets not loaded." >&2
  return 1
fi
set +a

rm -rf "${secretsDir}"

echo "KMS secrets loaded."
