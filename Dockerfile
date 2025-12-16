# Dockerfile for Go Driver local development.

# Build libmongocrypt in a separate build stage (use same distro/toolchain as final image).
FROM golang:1.25.5-bookworm AS libmongocrypt

RUN apt-get -qq update && \
  apt-get -qqy install --no-install-recommends \
    git \
    ca-certificates \
    curl \
    build-essential \
    libssl-dev \
    pkg-config \
    python3 \
    python-is-python3 && \
  rm -rf /var/lib/apt/lists/*

  COPY etc/install-libmongocrypt.sh /root/install-libmongocrypt.sh
RUN cd /root && bash ./install-libmongocrypt.sh


# Final dev image (already has Go 1.25.x).
FROM golang:1.25.5-bookworm

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
    make && \
  update-ca-certificates && \
  rm -rf /var/lib/apt/lists/*


COPY etc/docker_entry.sh /root/docker_entry.sh
COPY --from=libmongocrypt /root/install /root/install

ENV DOCKER_RUNNING=true
ENTRYPOINT ["/bin/bash", "/root/docker_entry.sh"]
