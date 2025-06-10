#!/usr/bin/env bash

year=$(date +"%Y")
copyright=$"// Copyright (C) MongoDB, Inc. $year-present.
//
// Licensed under the Apache License, Version 2.0 (the \"License\"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
"

check_or_add_copyright() {
    file=$1

    # Check if first 24 bytes match first 24 bytes of copyright notice.
    local line
    line=$(head -c 24 $file)
    if [ "$line" == "// Copyright (C) MongoDB" ]; then
        if [ ! -z "$verbose" ]; then
            echo "$file already has copyright notice" >&2
        fi
        return
    fi

    # Check if first 14 bytes matches the prefix "// Copied from"
    line=$(head -c 14 $file)
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

    echo "Missing copyright notice in \"$file\". Run \"task add-license\" to add missing licenses."
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
        *)
            echo "flag not recognized"
            exit 1
            ;;
    esac
done

# Shift script arguments so $1 contains only optional filename args.
# E.g. check_license.sh -a ./mongo/cursor.go
shift "$((OPTIND - 1))"
FILES=$1

# If no filenames were passed, find all .go files and try to write a license
# notice.
if [ -z "$FILES" ]; then
    FILES=$(find . -type f -name "*.go" -print)
fi

for file in $FILES
do
    # Skip any files that aren't .go files.
    if [[ ! "$file" =~ .go$ ]]; then
        continue
    fi

    check_or_add_copyright "$file"
done
