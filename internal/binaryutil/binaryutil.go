// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package binaryutil provides utility functions for working with binary data.
package binaryutil

func Append32[T ~uint32 | ~int32](dst []byte, v T) []byte {
	n := len(dst)
	dst = append(dst, 0, 0, 0, 0)

	b := dst[n:]
	_ = b[3] // early bounds check

	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)

	return dst
}
