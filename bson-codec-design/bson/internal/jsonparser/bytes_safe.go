// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package jsonparser

import (
	"strconv"
)

// See fastbytes_unsafe.go for explanation on why *[]byte is used (signatures must be consistent with those in that file)

func equalStr(b *[]byte, s string) bool {
	return string(*b) == s
}

func parseFloat(b *[]byte) (float64, error) {
	return strconv.ParseFloat(string(*b), 64)
}
