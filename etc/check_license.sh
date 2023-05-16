#!/usr/bin/env bash

year=$(date +"%Y")
copyright=$"// Copyright (C) MongoDB, Inc. $year-present.
//
// Licensed under the Apache License, Version 2.0 (the \"License\"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
"

add_copyright() {
    file=$1

    # Check if first 24 bytes match first 24 bytes of copyright notice.
    local line=$(head -c 24 $file)
    if [ "$line" == "// Copyright (C) MongoDB" ]; then
        if [ ! -z "$verbose" ]; then
            echo "$file already has copyright notice" >&2
        fi
        return
    fi

    # Check if first 14 bytes matches the prefix "// Copied from"
    local line=$(head -c 14 $file)
    if [ "$line" == "// Copied from" ]; then
        if [ ! -z "$verbose" ]; then
            echo "$file has a third-party copyright notice" >&2
        fi
        return
    fi

    if [ ! -z "$add" ]; then
        echo "$copyright" | cat - $file > temp && mv temp $file
        return
    fi

    echo "Missing copyright notice in \"$file\". Run \"make add-license\" to add missing licenses."
    exit 1
}

# Options are:
# -a : Add licenses that are missing.
# -v : Verbose. Print all files as they're checked.
while getopts at:vt: flag
do
    case "${flag}" in
        a) add=1;;
        v) verbose=1;;
    esac
done

# Find all .go files not in the vendor directory and try to write a license notice.
GO_FILES=$(find . -path ./vendor -prune -o -type f -name "*.go" -print)

for file in $GO_FILES
do
    add_copyright "$file"
done
