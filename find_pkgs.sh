#!/bin/sh
# find_pkg <directory> <suffix_pattern>
directory="$1"
pattern="$2"
if [ -z "$directory" ]; then
    directory="."
fi
find "$directory" -iname "*${pattern}.go" | perl -ple 's{(.*)/[^/]+}{$1}' | sort -u
