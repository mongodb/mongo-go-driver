#!/bin/sh
# list_pkgs <directory>
directory="$1"
if [ -z "$directory" ]; then
    directory="."
fi
go list -test -f '{{.ForTest}}' $directory/... | sed -e "s/^github.com\/mongodb\/mongo-go-driver/./"
