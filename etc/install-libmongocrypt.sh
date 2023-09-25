#!/usr/bin/env bash
# install libmongocrypt
# This script installs libmongocrypt into an "install" directory.
set -eux

LIBMONGOCRYPT_TAG="1.8.2"

# Install libmongocrypt based on OS.
if [ "Windows_NT" = "${OS:-}" ]; then
    INSTALL_DIR=c:/libmongocrypt/bin
    rm -rf $INSTALL_DIR
    mkdir -p $INSTALL_DIR/include
    mkdir -p $INSTALL_DIR/bin
    echo "fetching build for Windows ... begin"
    mkdir libmongocrypt-all
    pushd libmongocrypt-all
    # The following URL is published from the upload-all task in the libmongocrypt Evergreen project.
    curl https://mciuploads.s3.amazonaws.com/libmongocrypt/all/$LIBMONGOCRYPT_TAG/libmongocrypt-all.tar.gz -o libmongocrypt-all.tar.gz
    tar -xf libmongocrypt-all.tar.gz
    popd
    cp libmongocrypt-all/windows-test/bin/mongocrypt.dll $INSTALL_DIR/bin
    cp libmongocrypt-all/windows-test/include/mongocrypt/*.h $INSTALL_DIR/include
    rm -rf libmongocrypt-all
    echo "fetching build for Windows ... end"
    exit 0
fi

rm -rf libmongocrypt
git clone https://github.com/mongodb/libmongocrypt --depth=1 --branch $LIBMONGOCRYPT_TAG 2> /dev/null
if ! ( ./libmongocrypt/.evergreen/compile.sh >| output.txt 2>&1 ); then
    cat output.txt 1>&2
    exit 1
fi
mv output.txt install
rm -rf libmongocrypt
