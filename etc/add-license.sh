#!/usr/bin/env bash

year=$(date +"%Y")
copyright=$"// Copyright (C) MongoDB, Inc. $year-present.
//
// Licensed under the Apache License, Version 2.0 (the \"License\"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
"

add_copyright() {
    # Check if first 24 bytes match first 24 bytes of copyright notice.
    local line=$(head -c 24 $1)
    if [ "$line" == "// Copyright (C) MongoDB" ]; then
        echo "$1 already has copyright notice" >&2
        return
    fi

    echo "$copyright" | cat - $1 > temp && mv temp $1
}

for file in "$@"
do
    add_copyright "$file"
done
