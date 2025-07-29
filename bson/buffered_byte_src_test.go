// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"io"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
)

func TestBufferedvalueReader_discard(t *testing.T) {
	tests := []struct {
		name       string
		buf        []byte
		n          int
		want       int
		wantOffset int64
		wantErr    error
	}{
		{
			name:       "nothing",
			buf:        bytes.Repeat([]byte("a"), 1024),
			n:          0,
			want:       0,
			wantOffset: 0,
			wantErr:    nil,
		},
		{
			name:       "amount less than buffer size",
			buf:        bytes.Repeat([]byte("a"), 1024),
			n:          100,
			want:       100,
			wantOffset: 100,
			wantErr:    nil,
		},
		{
			name:       "amount greater than buffer size",
			buf:        bytes.Repeat([]byte("a"), 1024),
			n:          10000,
			want:       1024,
			wantOffset: 1024,
			wantErr:    io.EOF,
		},
		{
			name:       "exact buffer size",
			buf:        bytes.Repeat([]byte("a"), 1024),
			n:          1024,
			want:       1024,
			wantOffset: 1024,
			wantErr:    nil,
		},
		{
			name:       "from empty buffer",
			buf:        []byte{},
			n:          10,
			want:       0,
			wantOffset: 0,
			wantErr:    io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &bufferedByteSrc{buf: tt.buf, offset: 0}
			n, err := reader.discard(tt.n)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr, "Expected error %v, got %v", tt.wantErr, err)
			} else {
				require.NoError(t, err, "Expected no error when discarding %d bytes", tt.n)
			}

			assert.Equal(t, tt.want, n, "Expected to discard %d bytes, got %d", tt.want, n)
			assert.Equal(t, tt.wantOffset, reader.offset, "Expected offset to be %d, got %d", tt.wantOffset, reader.offset)
		})
	}
}

func TestBufferedvalueReader_peek(t *testing.T) {
	tests := []struct {
		name    string
		buf     []byte
		n       int
		offset  int64
		want    []byte
		wantErr error
	}{
		{
			name:    "nothing",
			buf:     bytes.Repeat([]byte("a"), 1024),
			n:       0,
			want:    []byte{},
			wantErr: nil,
		},
		{
			name:    "amount less than buffer size",
			buf:     bytes.Repeat([]byte("a"), 1024),
			n:       100,
			want:    bytes.Repeat([]byte("a"), 100),
			wantErr: nil,
		},
		{
			name:    "amount greater than buffer size",
			buf:     bytes.Repeat([]byte("a"), 1024),
			n:       10000,
			want:    bytes.Repeat([]byte("a"), 1024),
			wantErr: io.EOF,
		},
		{
			name:    "exact buffer size",
			buf:     bytes.Repeat([]byte("a"), 1024),
			n:       1024,
			want:    bytes.Repeat([]byte("a"), 1024),
			wantErr: nil,
		},
		{
			name:    "from empty buffer",
			buf:     []byte{},
			n:       10,
			want:    []byte{},
			wantErr: io.EOF,
		},
		{
			name:    "peek with offset",
			buf:     append(bytes.Repeat([]byte("a"), 100), bytes.Repeat([]byte("b"), 100)...),
			offset:  100,
			n:       100,
			want:    bytes.Repeat([]byte("b"), 100),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &bufferedByteSrc{buf: tt.buf, offset: tt.offset}
			n, err := reader.peek(tt.n)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr, "Expected error %v, got %v", tt.wantErr, err)
			} else {
				require.NoError(t, err, "Expected no error when peeking %d bytes", tt.n)
			}

			assert.Equal(t, tt.want, n, "Expected to peek %d bytes, got %d", len(tt.want), len(n))
			assert.Equal(t, tt.offset, reader.offset, "Expected offset to be %d, got %d", tt.offset, reader.offset)
		})
	}
}
