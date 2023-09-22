FROM drivers-evergreen-tools

RUN export DEBIAN_FRONTEND=noninteractive && \
  export TZ=Etc/UTC && \
  add-apt-repository ppa:longsleep/golang-backports && \
  apt-get -qq update && \
  apt-get -qqy install --no-install-recommends \
  gnupg \
  golang-go \
  pkg-config \
  libssl-dev \
  build-essential \
  tzdata \
  make \
 && rm -rf /var/lib/apt/lists/*

RUN export LIBMONGOCRYPT_TAG="1.8.2" && \
    cd $HOME && \
    git clone https://github.com/mongodb/libmongocrypt --depth=1 --branch $LIBMONGOCRYPT_TAG && \
    PKG_CONFIG_PATH=$HOME/install/libmongocrypt/lib/pkgconfig:$HOME/install/mongo-c-driver/lib/pkgconfig \
    LD_LIBRARY_PATH=$HOME/install/libmongocrypt/lib \
    ./libmongocrypt/.evergreen/compile.sh

COPY ./etc/docker_entry.sh /root/docker_entry.sh

ENTRYPOINT ["/bin/bash", "/root/docker_entry.sh"]
