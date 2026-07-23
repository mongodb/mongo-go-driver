# Dockerfile for Go Driver local development.
# sha found via this command: docker inspect --format='{{index .RepoDigests 0}}' golang:1.25.6-trixie
FROM golang:1.25.6-trixie@sha256:fb4b74a39c7318d53539ebda43ccd3ecba6e447a78591889c0efc0a7235ea8b3 AS base

# Build libmongocrypt in a separate build stage.
FROM base AS libmongocrypt

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

COPY etc/install-libmongocrypt.sh /root/install-libmongocrypt.sh
RUN cd /root && bash ./install-libmongocrypt.sh


# Final dev image (already has Go 1.25.x).
FROM base

RUN export DEBIAN_FRONTEND=noninteractive && \
  export TZ=Etc/UTC && \
  apt-get -qq update && \
  apt-get -qqy install --reinstall --no-install-recommends \
  git \
  ca-certificates \
  curl \
  wget \
  tzdata \
  pkg-config \
  gpg \
  apt-utils \
  libc6-dev \
  gcc \
  make \
  libkrb5-dev && \
  update-ca-certificates && \
  rm -rf /var/lib/apt/lists/*

# Install taskfile
RUN go install github.com/go-task/task/v3/cmd/task@v3.39.2

# Pre-download the Go toolchains used by internal/test/compilecheck so the
# compile check does not download them at runtime (the largest cost of that
# test). The parent Go can fetch and run older toolchain modules, and a single
# linux-amd64 toolchain per version cross-compiles for every GOARCH. Placed
# before "COPY . /mongo-go-driver" so repo changes don't invalidate this layer.
# Keep this list in sync with the GoVersions in
# internal/test/compilecheck/compile_check_test.go.
RUN for v in 1.19.0 1.20.0 1.21.0 1.22.0 1.23.0 1.24.0 1.25.0; do \
      GOTOOLCHAIN=go$v go version; \
    done

COPY etc/docker_entry.sh /root/docker_entry.sh
COPY --from=libmongocrypt /root/install /root/install

# Copy the Go driver source for local development and compile checks.
COPY . /mongo-go-driver

ENV DOCKER_RUNNING=true
ENTRYPOINT ["/bin/bash", "/root/docker_entry.sh"]
