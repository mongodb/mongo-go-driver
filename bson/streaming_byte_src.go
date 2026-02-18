// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bufio"
	"bytes"
	"io"
)

// streamingByteSrc reads from an ioReader wrapped in a bufio.Reader. It
// first reads the BSON length header, then ensures it only ever reads exactly
// that many bytes.
//
// Note: this approach trades memory usage for extra buffering and reader calls,
// so it is less performanted than the in-memory bufferedValueReader.
type streamingByteSrc struct {
	br     *bufio.Reader
	offset int64 // offset is the current read position in the buffer
}

var _ byteSrc = (*streamingByteSrc)(nil)

// Read reads up to len(p) bytes from the underlying bufio.Reader, advancing
// the offset by the number of bytes read.
func (s *streamingByteSrc) readExact(p []byte) (int, error) {
	n, err := io.ReadFull(s.br, p)
	if err == nil {
		s.offset += int64(n)
	}

	return n, err
}

// ReadByte returns the single byte at buf[offset] and advances offset by 1.
func (s *streamingByteSrc) ReadByte() (byte, error) {
	c, err := s.br.ReadByte()
	if err == nil {
		s.offset++
	}
	return c, err
}

// peek returns buf[offset:offset+n] without advancing offset.
func (s *streamingByteSrc) peek(n int) ([]byte, error) {
	return s.br.Peek(n)
}

// discard advances offset by n bytes, returning the number of bytes discarded.
func (s *streamingByteSrc) discard(n int) (int, error) {
	m, err := s.br.Discard(n)
	s.offset += int64(m)
	return m, err
}

// readSlice reads until the first occurrence of delim, returning a slice
// containing the data up to and including the delimiter, and advances offset
// past it; errors if delim not found.
func (s *streamingByteSrc) readSlice(delim byte) ([]byte, error) {
	var full [][]byte
	var frag []byte
	var err error
	var n int

	for {
		if len(frag) > 0 {
			// Make a copy of the fragment to accumulate full buffers.
			buf := bytes.Clone(frag)
			full = append(full, buf)
		}
		frag, err = s.br.ReadSlice(delim)
		n += len(frag)
		if err != bufio.ErrBufferFull {
			break
		}
	}
	s.offset += int64(n)

	// If ReadSlice is only called once, we can return the fragment directly.
	if len(full) == 0 {
		return frag, err
	}

	// Allocate new buffer to hold the full buffers and the fragment.
	buf := make([]byte, n)
	n = 0
	for i := range full {
		n += copy(buf[n:], full[i])
	}
	copy(buf[n:], frag)
	return buf, err
}

// pos returns the current read position in the buffer.
func (s *streamingByteSrc) pos() int64 {
	return s.offset
}

// regexLength will return the total byte length of a BSON regex value.
func (s *streamingByteSrc) regexLength() (int32, error) {
	var (
		count    int32
		nulCount int
	)

	for nulCount < 2 {
		buf, err := s.br.Peek(int(count) + 1)
		if err != nil {
			return 0, err
		}

		b := buf[count]
		count++
		if b == 0x00 {
			nulCount++
		}
	}

	return count, nil
}

func (*streamingByteSrc) streamable() bool {
	return true
}

func (s *streamingByteSrc) reset() {
	s.offset = 0
}
