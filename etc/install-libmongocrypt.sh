#!/usr/bin/env bash
# install libmongocrypt
# This script installs libmongocrypt into an "install" directory.
set -eux

# TODO(GODRIVER-4028): Remove UV_CONSTRAINT=<(echo "cmake<4.4") from the
# compile.sh call (see below) when we upgrade to libmongocrypt 1.21.0
LIBMONGOCRYPT_TAG="1.20.0"

# Install libmongocrypt based on OS.
if [ "Windows_NT" = "${OS:-}" ]; then
  mkdir -p c:/libmongocrypt/include
  mkdir -p c:/libmongocrypt/bin
  echo "fetching build for Windows ... begin"
  mkdir libmongocrypt-all
  cd libmongocrypt-all

  # Download the prebuilt Windows tarball and its detached PGP signature.
  base=https://github.com/mongodb/libmongocrypt/releases/download/$LIBMONGOCRYPT_TAG
  curl -LO $base/libmongocrypt-windows-x86_64-$LIBMONGOCRYPT_TAG.tar.gz
  curl -LO $base/libmongocrypt-windows-x86_64-$LIBMONGOCRYPT_TAG.asc

  # Download the MongoDB libmongocrypt public key, import it into an
  # isolated GNUPGHOME, and verify the tarball signature.
  curl -LO https://pgp.mongodb.com/libmongocrypt.pub
  GNUPGHOME=$(mktemp -d)
  export GNUPGHOME
  trap 'rm -rf "$GNUPGHOME"' EXIT
  gpg --batch --import libmongocrypt.pub
  gpg --batch --verify \
    libmongocrypt-windows-x86_64-$LIBMONGOCRYPT_TAG.asc \
    libmongocrypt-windows-x86_64-$LIBMONGOCRYPT_TAG.tar.gz

  tar -xf libmongocrypt-windows-x86_64-$LIBMONGOCRYPT_TAG.tar.gz
  cd ..
  cp libmongocrypt-all/bin/mongocrypt.dll c:/libmongocrypt/bin
  cp libmongocrypt-all/include/mongocrypt/*.h c:/libmongocrypt/include

  rm -rf libmongocrypt-all
  echo "fetching build for Windows ... end"
else
  rm -rf libmongocrypt
  git clone https://github.com/mongodb/libmongocrypt --depth=1 --branch $LIBMONGOCRYPT_TAG 2>/dev/null
  if ! (UV_CONSTRAINT=<(echo "cmake<4.4") ./libmongocrypt/.evergreen/compile.sh >|output.txt 2>&1); then
    cat output.txt 1>&2
    exit 1
  fi
  mv output.txt install
  rm -rf libmongocrypt
fi
