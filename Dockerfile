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

RUN export DEBIAN_FRONTEND=noninteractive && \
  export TZ=Etc/UTC && \
  echo "deb https://ppa.launchpadcontent.net/longsleep/golang-backports/ubuntu jammy main" >> /etc/apt/sources.list && \
  echo "deb-src https://ppa.launchpadcontent.net/longsleep/golang-backports/ubuntu jammy main" > /etc/apt/sources.list  && \
  apt-get -qq update && \
  apt-get -qqy install --no-install-recommends \
  golang-go \
  pkg-config \
  tzdata \
  make \
 && rm -rf /var/lib/apt/lists/*

COPY ./etc/docker_entry.sh /root/docker_entry.sh

COPY --from=libmongocrypt /root/install /root/install

ENTRYPOINT ["/bin/bash", "/root/docker_entry.sh"]
