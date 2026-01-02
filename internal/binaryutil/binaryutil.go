// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package binaryutil provides utility functions for working with binary data.
package binaryutil

import (
	"bytes"
)

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

func ReadU32(src []byte) (uint32, []byte, bool) {
	if len(src) < 4 {
		return 0, src, false
	}

	_ = src[3] // bounds check hint to compiler

	value := uint32(src[0]) |
		uint32(src[1])<<8 |
		uint32(src[2])<<16 |
		uint32(src[3])<<24

	return value, src[4:], true
}

func ReadI32(src []byte) (int32, []byte, bool) {
	if len(src) < 4 {
		return 0, src, false
	}

	_ = src[3] // bounds check hint to compiler

	value := int32(src[0]) |
		int32(src[1])<<8 |
		int32(src[2])<<16 |
		int32(src[3])<<24

	return value, src[4:], true
}

func ReadU64(src []byte) (uint64, []byte, bool) {
	if len(src) < 8 {
		return 0, src, false
	}

	_ = src[7] // bounds check hint to compiler

	value := uint64(src[0]) |
		uint64(src[1])<<8 |
		uint64(src[2])<<16 |
		uint64(src[3])<<24 |
		uint64(src[4])<<32 |
		uint64(src[5])<<40 |
		uint64(src[6])<<48 |
		uint64(src[7])<<56

	return value, src[8:], true
}

func ReadI64(src []byte) (int64, []byte, bool) {
	if len(src) < 8 {
		return 0, src, false
	}

	_ = src[7] // bounds check hint to compiler

	value := int64(src[0]) |
		int64(src[1])<<8 |
		int64(src[2])<<16 |
		int64(src[3])<<24 |
		int64(src[4])<<32 |
		int64(src[5])<<40 |
		int64(src[6])<<48 |
		int64(src[7])<<56

	return value, src[8:], true
}

func ReadCStringBytes(src []byte) ([]byte, []byte, bool) {
	idx := bytes.IndexByte(src, 0x00)
	if idx < 0 {
		return nil, src, false
	}
	return src[:idx], src[idx+1:], true
}

func ReadCString(src []byte) (string, []byte, bool) {
	cstr, rem, ok := ReadCStringBytes(src)
	if !ok {
		return "", src, false
	}
	return string(cstr), rem, true
}
