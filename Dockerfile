FROM ubuntu:22.04

RUN apt-get -qq update && apt-get -qqy -o DPkg::Lock::Timeout=-1 install --no-install-recommends \
  git \
  ca-certificates \
  curl \
  wget \
  sudo \
  gnupg \
  python3 \
  python3.10-venv \
  build-essential \
  golang-go \
  pkg-config \
  libssl-dev \
  make \
  lsof \
  net-tools \
 && rm -rf /var/lib/apt/lists/*

RUN export LIBMONGOCRYPT_TAG="1.8.2" && \
    cd $HOME && \
    git clone https://github.com/mongodb/libmongocrypt --depth=1 --branch $LIBMONGOCRYPT_TAG && \
    PKG_CONFIG_PATH=$HOME/install/libmongocrypt/lib/pkgconfig:$HOME/install/mongo-c-driver/lib/pkgconfig \
    LD_LIBRARY_PATH=$HOME/install/libmongocrypt/lib \
    ./libmongocrypt/.evergreen/compile.sh

COPY ./etc/docker_entry.sh /root/docker_entry.sh

ENTRYPOINT ["/bin/bash", "/root/docker_entry.sh"]
