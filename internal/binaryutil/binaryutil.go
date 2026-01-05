// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package binaryutil provides utility functions for working with binary data.
package binaryutil

import (
	"bytes"
	"encoding/binary"
)

// Append32 appends a uint32 or int32 value to dst in little-endian byte order.
// Byte shifting is done directly to prevent overflow security errors, in
// compliance with gosec G115.
//
// See: https://cs.opensource.google/go/go/+/refs/tags/go1.19:src/encoding/binary/binary.go;l=92
func Append32[T ~uint32 | ~int32](dst []byte, v T) []byte {
	return append(dst,
		byte(v),
		byte(v>>8),
		byte(v>>16),
		byte(v>>24),
	)
}

// Append64 appends a uint64 or int64 value to dst in little-endian byte order.
// Byte shifting is done directly to prevent overflow security errors, in
// compliance with gosec G115.
//
// See: https://cs.opensource.google/go/go/+/refs/tags/go1.19:src/encoding/binary/binary.go;l=119
func Append64[T ~uint64 | ~int64](dst []byte, v T) []byte {
	return append(dst,
		byte(v),
		byte(v>>8),
		byte(v>>16),
		byte(v>>24),
		byte(v>>32),
		byte(v>>40),
		byte(v>>48),
		byte(v>>56),
	)
}

// ReadU32 reads a uint32 from src in little-endian byte order. ReadU32 and
// ReadI32 are separate functions to avoid unsafe casting between unsigned and
// signed integers.
func ReadU32(src []byte) (uint32, []byte, bool) {
	if len(src) < 4 {
		return 0, src, false
	}

	return binary.LittleEndian.Uint32(src), src[4:], true
}

// ReadI32 reads an int32 from src in little-endian byte order.
// Byte shifting is done directly to prevent overflow security errors, in
// compliance with gosec G115. ReadU32 and ReadI32 are separate functions to
// avoid unsafe casting between unsigned and signed integers.
//
// See: https://cs.opensource.google/go/go/+/refs/tags/go1.19:src/encoding/binary/binary.go;l=79
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

// ReadU64 reads a uint64 from src in little-endian byte order. ReadU64 and
// ReadI64 are separate functions to avoid unsafe casting between unsigned and
// signed integers.
func ReadU64(src []byte) (uint64, []byte, bool) {
	if len(src) < 8 {
		return 0, src, false
	}

	return binary.LittleEndian.Uint64(src), src[8:], true
}

// ReadI64 reads an int64 from src in little-endian byte order.
// Byte shifting is done directly to prevent overflow security errors, in
// compliance with gosec G115. ReadU64 and ReadI64 are separate functions to
// avoid unsafe casting between unsigned and signed integers.
//
// See: https://cs.opensource.google/go/go/+/refs/tags/go1.19:src/encoding/binary/binary.go;l=101
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

// ReadCStringBytes reads a null-terminated C string from src as a byte slice.
// This is the base implementation used by ReadCString to ensure a single source
// of truth for C string parsing logic.
func ReadCStringBytes(src []byte) ([]byte, []byte, bool) {
	idx := bytes.IndexByte(src, 0x00)
	if idx < 0 {
		return nil, src, false
	}
	return src[:idx], src[idx+1:], true
}

// ReadCString reads a null-terminated C string from src as a string.
// It delegates to ReadCStringBytes to maintain a single source of truth for
// C string parsing logic.
func ReadCString(src []byte) (string, []byte, bool) {
	cstr, rem, ok := ReadCStringBytes(src)
	if !ok {
		return "", src, false
	}
	return string(cstr), rem, true
}
