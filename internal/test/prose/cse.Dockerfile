# Dockerfile for running Client-Side Encryption (CSE) tests.
#
# Builds libmongocrypt so cgo can build with the "cse" tag. The driver source
# is not copied in: the repository root is bind-mounted at /mongo-go-driver
# when the container starts, so driver changes do not require a rebuild.
#
# sha found via this command: docker inspect --format='{{index .RepoDigests 0}}' golang:1.26.4-trixie
FROM golang:1.26.4-trixie@sha256:76a29248dedcd75870e95cbd90cc8cb356db082404ac7d3a5803f276c3ba79c9

RUN apt-get -qq update && \
  apt-get -qqy install --no-install-recommends \
  git \
  ca-certificates \
  curl \
  build-essential \
  libssl-dev \
  pkg-config \
  python3 \
  python3-packaging \
  python-is-python3 && \
  rm -rf /var/lib/apt/lists/*

COPY install-libmongocrypt.sh /root/install-libmongocrypt.sh
RUN cd /root && bash ./install-libmongocrypt.sh

# The .pc file produced by install-libmongocrypt.sh bakes in a temp build
# directory as prefix=. Override it with the install location inside the
# image; etc/libmongocrypt-pkg-config.sh would resolve to the host's install
# directory through the bind mount.
RUN printf '#!/bin/sh\nexec pkg-config --define-variable=prefix=/root/install/libmongocrypt "$@"\n' \
  > /usr/local/bin/libmongocrypt-pkg-config && \
  chmod +x /usr/local/bin/libmongocrypt-pkg-config

ENV PKG_CONFIG=/usr/local/bin/libmongocrypt-pkg-config
ENV PKG_CONFIG_PATH=/root/install/libmongocrypt/lib64/pkgconfig:/root/install/libmongocrypt/lib/pkgconfig
ENV LD_LIBRARY_PATH=/root/install/libmongocrypt/lib64:/root/install/libmongocrypt/lib

ARG CRYPT_SHARED_VERSION=latest-stable
COPY mongodl.py /root/mongodl.py
RUN python3 /root/mongodl.py \
  --component crypt_shared \
  --version "${CRYPT_SHARED_VERSION}" \
  --target debian12 \
  --out /root/install/crypt_shared \
  --strip-path-components 1

ENV CRYPT_SHARED_LIB_PATH=/root/install/crypt_shared/lib/mongo_crypt_v1.so

WORKDIR /mongo-go-driver
