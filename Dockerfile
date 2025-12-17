# Dockerfile for Go Driver local development.

# Build libmongocrypt in a separate build stage.
FROM golang:1.25.5-trixie AS libmongocrypt

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
FROM golang:1.25.5-trixie

RUN export DEBIAN_FRONTEND=noninteractive && \
  export TZ=Etc/UTC && \
  apt-get -qq update && \
  apt-get -qqy install --reinstall --no-install-recommends \
    git \
    ca-certificates \
    curl \
    wget \
    sudo \
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


COPY etc/docker_entry.sh /root/docker_entry.sh
COPY --from=libmongocrypt /root/install /root/install

# Copy the Go driver source for local development and compile checks.
COPY . /mongo-go-driver

ENV DOCKER_RUNNING=true
ENTRYPOINT ["/bin/bash", "/root/docker_entry.sh"]
