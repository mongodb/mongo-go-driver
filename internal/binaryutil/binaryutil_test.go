// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package binaryutil

import (
	"math"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
)

func TestReadi64(t *testing.T) {
	testCases := []struct {
		desc    string
		src     []byte
		want    int64
		wantRem []byte
		wantOK  bool
	}{
		{
			desc:    "0",
			src:     []byte{0, 0, 0, 0, 0, 0, 0, 0}, // little-endian int64(0)
			want:    0,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "1",
			src:     []byte{1, 0, 0, 0, 0, 0, 0, 0}, // little-endian int64(1)
			want:    1,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "-1",
			src:     []byte{255, 255, 255, 255, 255, 255, 255, 255}, // little-endian int64(-1)
			want:    -1,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "max",
			src:     []byte{255, 255, 255, 255, 255, 255, 255, 127}, // little-endian max int64
			want:    math.MaxInt64,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "min",
			src:     []byte{0, 0, 0, 0, 0, 0, 0, 128}, // little-endian min int64
			want:    math.MinInt64,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "non-empty remaining",
			src:     []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3}, // little-endian int64(1) + extra bytes
			want:    1,
			wantRem: []byte{0, 1, 2, 3},
			wantOK:  true,
		},
		{
			desc:    "not enough bytes",
			src:     []byte{0, 1, 2, 3, 4, 5, 6}, // only 7 bytes, need 8
			want:    0,
			wantRem: []byte{0, 1, 2, 3, 4, 5, 6},
			wantOK:  false,
		},
		{
			desc:    "nil",
			src:     nil, // nil slice
			want:    0,
			wantRem: nil,
			wantOK:  false,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			x, rem, ok := ReadI64(tc.src)
			assert.Equal(t, tc.want, x, "int64 result does not match")
			assert.Equal(t, tc.wantRem, rem, "remaining bytes do not match")
			assert.Equal(t, tc.wantOK, ok, "OK does not match")
		})
	}
}

func TestAppendI64(t *testing.T) {
	testCases := []struct {
		desc string
		dst  []byte
		x    int64
		want []byte
	}{
		{
			desc: "0",
			dst:  []byte{},
			x:    0,
			want: []byte{0, 0, 0, 0, 0, 0, 0, 0}, // little-endian int64(0)
		},
		{
			desc: "1",
			dst:  []byte{},
			x:    1,
			want: []byte{1, 0, 0, 0, 0, 0, 0, 0}, // little-endian int64(1)
		},
		{
			desc: "-1",
			dst:  []byte{},
			x:    -1,
			want: []byte{255, 255, 255, 255, 255, 255, 255, 255}, // little-endian int64(-1)
		},
		{
			desc: "max",
			dst:  []byte{},
			x:    math.MaxInt64,
			want: []byte{255, 255, 255, 255, 255, 255, 255, 127}, // little-endian max int64
		},
		{
			desc: "min",
			dst:  []byte{},
			x:    math.MinInt64,
			want: []byte{0, 0, 0, 0, 0, 0, 0, 128}, // little-endian min int64
		},
		{
			desc: "non-empty dst",
			dst:  []byte{0, 1, 2, 3},
			x:    1,
			want: []byte{0, 1, 2, 3, 1, 0, 0, 0, 0, 0, 0, 0}, // existing bytes + little-endian int64(1)
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			b := AppendI64(tc.dst, tc.x)
			assert.Equal(t, tc.want, b, "bytes do not match")
		})
	}
}

func TestAppendI32(t *testing.T) {
	testCases := []struct {
		desc string
		dst  []byte
		x    int32
		want []byte
	}{
		{
			desc: "0",
			x:    0,
			want: []byte{0, 0, 0, 0}, // little-endian int32(0)
		},
		{
			desc: "1",
			x:    1,
			want: []byte{1, 0, 0, 0}, // little-endian int32(1)
		},
		{
			desc: "-1",
			x:    -1,
			want: []byte{255, 255, 255, 255}, // little-endian int32(-1)
		},
		{
			desc: "max",
			x:    math.MaxInt32,
			want: []byte{255, 255, 255, 127}, // little-endian max int32
		},
		{
			desc: "min",
			x:    math.MinInt32,
			want: []byte{0, 0, 0, 128}, // little-endian min int32
		},
		{
			desc: "non-empty dst",
			dst:  []byte{0, 1, 2, 3},
			x:    1,
			want: []byte{0, 1, 2, 3, 1, 0, 0, 0}, // existing bytes + little-endian int32(1)
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			b := AppendI32(tc.dst, tc.x)
			assert.Equal(t, tc.want, b, "bytes do not match")
		})
	}
}

func TestReadI32(t *testing.T) {
	testCases := []struct {
		desc    string
		src     []byte
		want    int32
		wantRem []byte
		wantOK  bool
	}{
		{
			desc:    "0",
			src:     []byte{0, 0, 0, 0}, // little-endian int32(0)
			want:    0,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "1",
			src:     []byte{1, 0, 0, 0}, // little-endian int32(1)
			want:    1,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "-1",
			src:     []byte{255, 255, 255, 255}, // little-endian int32(-1)
			want:    -1,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "max",
			src:     []byte{255, 255, 255, 127}, // little-endian max int32
			want:    math.MaxInt32,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "min",
			src:     []byte{0, 0, 0, 128}, // little-endian min int32
			want:    math.MinInt32,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "non-empty remaining",
			src:     []byte{1, 0, 0, 0, 0, 1, 2, 3}, // little-endian int32(1) + extra bytes
			want:    1,
			wantRem: []byte{0, 1, 2, 3},
			wantOK:  true,
		},
		{
			desc:    "not enough bytes",
			src:     []byte{0, 1, 2}, // only 3 bytes, need 4
			want:    0,
			wantRem: []byte{0, 1, 2},
			wantOK:  false,
		},
		{
			desc:    "nil",
			src:     nil, // nil slice
			want:    0,
			wantRem: nil,
			wantOK:  false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			x, rem, ok := ReadI32(tc.src)
			assert.Equal(t, tc.want, x, "int32 result does not match")
			assert.Equal(t, tc.wantRem, rem, "remaining bytes do not match")
			assert.Equal(t, tc.wantOK, ok, "OK does not match")
		})
	}
}

func TestReadI32Unsafe(t *testing.T) {
	testCases := []struct {
		desc string
		src  []byte
		want int32
	}{
		{
			desc: "0",
			src:  []byte{0, 0, 0, 0}, // little-endian int32(0)
			want: 0,
		},
		{
			desc: "1",
			src:  []byte{1, 0, 0, 0}, // little-endian int32(1)
			want: 1,
		},
		{
			desc: "-1",
			src:  []byte{255, 255, 255, 255}, // little-endian int32(-1)
			want: -1,
		},
		{
			desc: "max",
			src:  []byte{255, 255, 255, 127}, // little-endian max int32
			want: math.MaxInt32,
		},
		{
			desc: "min",
			src:  []byte{0, 0, 0, 128}, // little-endian min int32
			want: math.MinInt32,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			x := ReadI32Unsafe(tc.src)
			assert.Equal(t, tc.want, x, "int32 result does not match")
		})
	}
}

func TestPutI32(t *testing.T) {
	testCases := []struct {
		desc   string
		dst    []byte
		offset int
		value  int32
		want   []byte
	}{
		{
			desc:   "0",
			dst:    make([]byte, 4),
			offset: 0,
			value:  0,
			want:   []byte{0, 0, 0, 0}, // little-endian int32(0)
		},
		{
			desc:   "1",
			dst:    make([]byte, 4),
			offset: 0,
			value:  1,
			want:   []byte{1, 0, 0, 0}, // little-endian int32(1)
		},
		{
			desc:   "-1",
			dst:    make([]byte, 4),
			offset: 0,
			value:  -1,
			want:   []byte{255, 255, 255, 255}, // little-endian int32(-1)
		},
		{
			desc:   "max",
			dst:    make([]byte, 4),
			offset: 0,
			value:  math.MaxInt32,
			want:   []byte{255, 255, 255, 127}, // little-endian max int32
		},
		{
			desc:   "min",
			dst:    make([]byte, 4),
			offset: 0,
			value:  math.MinInt32,
			want:   []byte{0, 0, 0, 128}, // little-endian min int32
		},
		{
			desc:   "with offset",
			dst:    []byte{99, 99, 0, 0, 0, 0, 99, 99},
			offset: 2,
			value:  1,
			want:   []byte{99, 99, 1, 0, 0, 0, 99, 99}, // little-endian int32(1) at offset 2
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			PutI32(tc.dst, tc.offset, tc.value)
			assert.Equal(t, tc.want, tc.dst, "bytes do not match")
		})
	}
}
