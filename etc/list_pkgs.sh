#!/bin/sh
# list_pkgs <directory>
directory="$1"
if [ -z "$directory" ]; then
    directory="."
fi
go list $directory/... | sed -e "s/^drivers.mongodb.org\/go/./"
