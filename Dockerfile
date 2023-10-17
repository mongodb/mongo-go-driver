# Dockerfile for Go Driver local development.

# Build libmongocrypt in a separate build stage.
FROM ubuntu:20.04 as libmongocrypt

RUN apt-get -qq update && \
  apt-get -qqy install --no-install-recommends \
    git \
    ca-certificates \
    curl \
    build-essential \
    libssl-dev \
    python

COPY etc/install-libmongocrypt.sh /root/install-libmongocrypt.sh
RUN cd /root && bash ./install-libmongocrypt.sh


# Inherit from the drivers-evergreen-tools image and copy in the files
# from the libmongocrypt build stage.
FROM drivers-evergreen-tools

# Install common deps.
RUN export DEBIAN_FRONTEND=noninteractive && \
  export TZ=Etc/UTC && \
  apt-get -qq update && \
  apt-get -qqy install --reinstall --no-install-recommends \
    tzdata \
    ca-certificates \
    pkg-config \
    software-properties-common \
    gpg \
    apt-utils \
    make && \
  sudo update-ca-certificates && \
  rm -rf /var/lib/apt/lists/*
  
# Install golang from the golang-backports ppa.
RUN export DEBIAN_FRONTEND=noninteractive && \
  export TZ=Etc/UTC && \
  export LC_ALL=C.UTF-8 && \
  sudo -E apt-add-repository "ppa:longsleep/golang-backports" && \
  apt-get -qq update && \
  apt-get -qqy install --no-install-recommends golang-go && \
  rm -rf /var/lib/apt/lists/*

COPY ./etc/docker_entry.sh /root/docker_entry.sh

COPY --from=libmongocrypt /root/install /root/install

ENTRYPOINT ["/bin/bash", "/root/docker_entry.sh"]
