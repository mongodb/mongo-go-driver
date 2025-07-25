// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"io"
)

// bufferedByteSrc implements the low-level byteSrc interface by reading
// directly from an in-memory byte slice. It provides efficient, zero-copy
// access for parsing BSON when the entire document is buffered in memory.
type bufferedByteSrc struct {
	buf    []byte // entire BSON document
	offset int64  // Current read index into buf
}

var _ byteSrc = (*bufferedByteSrc)(nil)

// Read reads up to len(p) bytes from the in-memory buffer, advancing the offset
// by the number of bytes read.
func (b *bufferedByteSrc) readExact(p []byte) (int, error) {
	if b.offset >= int64(len(b.buf)) {
		return 0, io.EOF
	}
	n := copy(p, b.buf[b.offset:])
	b.offset += int64(n)
	return n, nil
}

// ReadByte returns the single byte at buf[offset] and advances offset by 1.
func (b *bufferedByteSrc) ReadByte() (byte, error) {
	if b.offset >= int64(len(b.buf)) {
		return 0, io.EOF
	}
	b.offset++
	return b.buf[b.offset-1], nil
}

// peek returns buf[offset:offset+n] without advancing offset.
func (b *bufferedByteSrc) peek(n int) ([]byte, error) {
	// Ensure we don't read past the end of the buffer.
	if int64(n)+b.offset > int64(len(b.buf)) {
		return b.buf[b.offset:], io.EOF
	}

	// Return the next n bytes without advancing the offset
	return b.buf[b.offset : b.offset+int64(n)], nil
}

// discard advances offset by n bytes, returning the number of bytes discarded.
func (b *bufferedByteSrc) discard(n int) (int, error) {
	// Ensure we don't read past the end of the buffer.
	if int64(n)+b.offset > int64(len(b.buf)) {
		// If we have exceeded the buffer length, discard only up to the end.
		left := len(b.buf) - int(b.offset)
		b.offset = int64(len(b.buf))

		return left, io.EOF
	}

	// Advance the read position
	b.offset += int64(n)
	return n, nil
}

// readSlice scans buf[offset:] for the first occurrence of delim, returns
// buf[offset:idx+1], and advances offset past it; errors if delim not found.
func (b *bufferedByteSrc) readSlice(delim byte) ([]byte, error) {
	// Ensure we don't read past the end of the buffer.
	if b.offset >= int64(len(b.buf)) {
		return nil, io.EOF
	}

	// Look for the delimiter in the remaining bytes
	rem := b.buf[b.offset:]
	idx := bytes.IndexByte(rem, delim)
	if idx < 0 {
		return nil, io.EOF
	}

	// Build the result slice up through the delimiter.
	result := rem[:idx+1]

	// Advance the offset past the delimiter.
	b.offset += int64(idx + 1)

	return result, nil
}

// pos returns the current read position in the buffer.
func (b *bufferedByteSrc) pos() int64 {
	return b.offset
}

// regexLength will return the total byte length of a BSON regex value.
func (b *bufferedByteSrc) regexLength() (int32, error) {
	rem := b.buf[b.offset:]

	// Find end of the first C-string (pattern).
	i := bytes.IndexByte(rem, 0x00)
	if i < 0 {
		return 0, io.EOF
	}

	// Find end of second C-string (options).
	j := bytes.IndexByte(rem[i+1:], 0x00)
	if j < 0 {
		return 0, io.EOF
	}

	// Total length = first C-string length (pattern) + second C-string length
	// (options) + 2 null terminators
	return int32(i + j + 2), nil
}

func (*bufferedByteSrc) streamable() bool {
	return false
}

func (b *bufferedByteSrc) reset() {
	b.buf = nil
	b.offset = 0
}
