#!/bin/bash

AUTH=${AUTH:-noauth} 
SSL=${SSL:-nossl} 
MONGODB_URI=${MONGODB_URI:-}
TOPOLOGY=${TOPOLOGY:-server}
COMPRESSORS=${COMPRESSORS:-}

if [ "$COMPRESSORS" != "" ]; then
     if [[ "$MONGODB_URI" == *"?"* ]]; then
       export MONGODB_URI="${MONGODB_URI}&compressors=${COMPRESSORS}"
     else
       export MONGODB_URI="${MONGODB_URI}/?compressors=${COMPRESSORS}"
     fi
fi

make evg-test