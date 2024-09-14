#!/usr/bin/env bash
# Prepares perf data.
set -eux
    
if [ -d /testdata/perf/ ]; then 
    echo "/testdata/perf/ already exists, skipping download"
    exit 0
fi

curl --retry 5 "https://s3.amazonaws.com/boxes.10gen.com/build/driver-test-data.tar.gz" -o driver-test-data.tar.gz --silent --max-time 120

if [ $(uname -s) == "Darwin" ]; then 
    tar -zxf driver-test-data.tar.gz -s /testdata/perf/
else
    tar -zxf driver-test-data.tar.gz  --transform=s/testdata/perf/
fi 