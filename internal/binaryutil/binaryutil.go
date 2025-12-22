// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package binaryutil provides utility functions for working with binary data.
package binaryutil

import "encoding/binary"

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

func Append64[T ~uint64 | ~int64](dst []byte, v T) []byte {
	n := len(dst)
	dst = append(dst, 0, 0, 0, 0, 0, 0, 0, 0)

	b := dst[n:]
	_ = b[7] // early bounds check

	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
	b[4] = byte(v >> 32)
	b[5] = byte(v >> 40)
	b[6] = byte(v >> 48)
	b[7] = byte(v >> 56)

	return dst
}

func read32[T ~uint32 | ~int32](src []byte) (T, []byte, bool) {
	if len(src) < 4 {
		return 0, src, false
	}
	return T(binary.LittleEndian.Uint32(src)), src[4:], true
}

func ReadU32(src []byte) (uint32, []byte, bool) {
	return read32[uint32](src)
}

func ReadI32(src []byte) (int32, []byte, bool) {
	return read32[int32](src)
}
