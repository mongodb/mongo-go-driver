// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package binaryutil

// TODO(GODRIVER-3707): Consolidate remaining duplicate binary utility functions
// from bsoncore and wiremessage packages.

// ReadI64 reads an 8-byte little-endian int64 from src returning the value,
// remaining bytes, and ok flag.
func ReadI64(src []byte) (int64, []byte, bool) {
	if len(src) < 8 {
		return 0, src, false
	}

	_ = src[7] // bounds check hint to compiler

	// Do arithmetic in uint64, then convert to int64
	value := uint64(src[0]) |
		uint64(src[1])<<8 |
		uint64(src[2])<<16 |
		uint64(src[3])<<24 |
		uint64(src[4])<<32 |
		uint64(src[5])<<40 |
		uint64(src[6])<<48 |
		uint64(src[7])<<56 // MSB contains sign bit (bit 63)

	return int64(value), src[8:], true
}

// AppendI64 appends an int64 to dst in little-endian byte order.
func AppendI64(dst []byte, x int64) []byte {
	return append(dst,
		byte(x),
		byte(x>>8),
		byte(x>>16),
		byte(x>>24),
		byte(x>>32),
		byte(x>>40),
		byte(x>>48),
		byte(x>>56),
	)
}

// AppendI32 appends an int32 to dst in little-endian byte order.
func AppendI32(dst []byte, x int32) []byte {
	return append(dst,
		byte(x),
		byte(x>>8),
		byte(x>>16),
		byte(x>>24),
	)
}

// ReadI32 reads a 32-bit little-endian int32 from src returning the value,
// remaining bytes, and ok flag.
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

// ReadI32Unsafe reads a 32-bit little-endian int32 from src without length
// checks. The caller must ensure src has at least 4 bytes.
func ReadI32Unsafe(src []byte) int32 {
	_ = src[3] // bounds check hint to compiler

	return int32(src[0]) | int32(src[1])<<8 | int32(src[2])<<16 | int32(src[3])<<24
}

// PutI32 writes a little-endian int32 into dst starting at offset. Caller must
// ensure capacity.
func PutI32(dst []byte, offset int, value int32) {
	dst[offset] = byte(value)
	dst[offset+1] = byte(value >> 8)
	dst[offset+2] = byte(value >> 16)
	dst[offset+3] = byte(value >> 24)
}
