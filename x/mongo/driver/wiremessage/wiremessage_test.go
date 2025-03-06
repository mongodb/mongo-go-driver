// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package wiremessage

import (
	"math"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

func TestAppendHeaderStart(t *testing.T) {
	testCases := []struct {
		desc      string
		dst       []byte
		reqid     int32
		respto    int32
		opcode    OpCode
		wantIdx   int32
		wantBytes []byte
	}{
		{
			desc:      "OP_MSG",
			reqid:     2,
			respto:    1,
			opcode:    OpMsg,
			wantIdx:   0,
			wantBytes: []byte{0, 0, 0, 0, 2, 0, 0, 0, 1, 0, 0, 0, 221, 7, 0, 0},
		},
		{
			desc:      "OP_QUERY",
			reqid:     2,
			respto:    1,
			opcode:    OpQuery,
			wantIdx:   0,
			wantBytes: []byte{0, 0, 0, 0, 2, 0, 0, 0, 1, 0, 0, 0, 212, 7, 0, 0},
		},
		{
			desc:      "non-empty buffer",
			dst:       []byte{0, 99},
			reqid:     2,
			respto:    1,
			opcode:    OpMsg,
			wantIdx:   2,
			wantBytes: []byte{0, 99, 0, 0, 0, 0, 2, 0, 0, 0, 1, 0, 0, 0, 221, 7, 0, 0},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			idx, b := AppendHeaderStart(tc.dst, tc.reqid, tc.respto, tc.opcode)
			assert.Equal(t, tc.wantIdx, idx, "appended slice index does not match")
			assert.Equal(t, tc.wantBytes, b, "appended bytes do not match")
		})
	}
}

func TestReadHeader(t *testing.T) {
	testCases := []struct {
		desc           string
		src            []byte
		wantLength     int32
		wantRequestID  int32
		wantResponseTo int32
		wantOpcode     OpCode
		wantRem        []byte
		wantOK         bool
	}{
		{
			desc:           "OP_MSG",
			src:            []byte{0, 0, 0, 0, 2, 0, 0, 0, 1, 0, 0, 0, 221, 7, 0, 0},
			wantLength:     0,
			wantRequestID:  2,
			wantResponseTo: 1,
			wantOpcode:     OpMsg,
			wantRem:        []byte{},
			wantOK:         true,
		},
		{
			desc:           "OP_QUERY",
			src:            []byte{0, 0, 0, 0, 2, 0, 0, 0, 1, 0, 0, 0, 212, 7, 0, 0},
			wantLength:     0,
			wantRequestID:  2,
			wantResponseTo: 1,
			wantOpcode:     OpQuery,
			wantRem:        []byte{},
			wantOK:         true,
		},
		{
			desc:           "not enough bytes",
			src:            []byte{0, 99},
			wantLength:     0,
			wantRequestID:  0,
			wantResponseTo: 0,
			wantOpcode:     0,
			wantRem:        []byte{0, 99},
			wantOK:         false,
		},
		{
			desc:           "nil",
			src:            nil,
			wantLength:     0,
			wantRequestID:  0,
			wantResponseTo: 0,
			wantOpcode:     0,
			wantRem:        nil,
			wantOK:         false,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			length, requestID, responseTo, opcode, rem, ok := ReadHeader(tc.src)
			assert.Equal(t, tc.wantLength, length, "length does not match")
			assert.Equal(t, tc.wantRequestID, requestID, "requestID does not match")
			assert.Equal(t, tc.wantResponseTo, responseTo, "responseTo does not match")
			assert.Equal(t, tc.wantOpcode, opcode, "OpCode does not match")
			assert.Equal(t, tc.wantRem, rem, "remaining bytes do not match")
			assert.Equal(t, tc.wantOK, ok, "OK does not match")
		})
	}
}

func TestReadMsgSectionDocumentSequence(t *testing.T) {
	testCases := []struct {
		desc           string
		src            []byte
		wantIdentifier string
		wantDocs       []bsoncore.Document
		wantRem        []byte
		wantOK         bool
	}{
		{
			desc: "valid document sequence",
			// Data:              |  len=17   |    "id"    |  empty doc   |  empty doc   |
			src:            []byte{17, 0, 0, 0, 105, 100, 0, 5, 0, 0, 0, 0, 5, 0, 0, 0, 0},
			wantIdentifier: "id",
			wantDocs: []bsoncore.Document{
				{0x5, 0x0, 0x0, 0x0, 0x0},
				{0x5, 0x0, 0x0, 0x0, 0x0},
			},
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc: "valid document sequence with remaining bytes",
			// Data:              |  len=17   |    "id"    |  empty doc   |  empty doc   |  rem  |
			src:            []byte{17, 0, 0, 0, 105, 100, 0, 5, 0, 0, 0, 0, 5, 0, 0, 0, 0, 99, 99},
			wantIdentifier: "id",
			wantDocs: []bsoncore.Document{
				{0x5, 0x0, 0x0, 0x0, 0x0},
				{0x5, 0x0, 0x0, 0x0, 0x0},
			},
			wantRem: []byte{99, 99},
			wantOK:  true,
		},
		{
			desc:           "not enough bytes",
			src:            []byte{0, 1},
			wantIdentifier: "",
			wantDocs:       nil,
			wantRem:        []byte{0, 1},
			wantOK:         false,
		},
		{
			desc:           "incorrect size",
			src:            []byte{3, 0, 0},
			wantIdentifier: "",
			wantDocs:       nil,
			wantRem:        []byte{3, 0, 0},
			wantOK:         false,
		},
		{
			desc:           "insufficient size",
			src:            []byte{4, 0, 0},
			wantIdentifier: "",
			wantDocs:       nil,
			wantRem:        []byte{4, 0, 0},
			wantOK:         false,
		},
		{
			desc:           "nil",
			src:            nil,
			wantIdentifier: "",
			wantDocs:       nil,
			wantRem:        nil,
			wantOK:         false,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			identifier, docs, rem, ok := ReadMsgSectionDocumentSequence(tc.src)
			assert.Equal(t, tc.wantIdentifier, identifier, "identifier does not match")
			assert.Equal(t, tc.wantDocs, docs, "docs do not match")
			assert.Equal(t, tc.wantRem, rem, "responseTo does not match")
			assert.Equal(t, tc.wantOK, ok, "OK does not match")
		})
	}
}

func TestAppendi32(t *testing.T) {
	testCases := []struct {
		desc string
		dst  []byte
		x    int32
		want []byte
	}{
		{
			desc: "0",
			x:    0,
			want: []byte{0, 0, 0, 0},
		},
		{
			desc: "1",
			x:    1,
			want: []byte{1, 0, 0, 0},
		},
		{
			desc: "-1",
			x:    -1,
			want: []byte{255, 255, 255, 255},
		},
		{
			desc: "max",
			x:    math.MaxInt32,
			want: []byte{255, 255, 255, 127},
		},
		{
			desc: "min",
			x:    math.MinInt32,
			want: []byte{0, 0, 0, 128},
		},
		{
			desc: "non-empty dst",
			dst:  []byte{0, 1, 2, 3},
			x:    1,
			want: []byte{0, 1, 2, 3, 1, 0, 0, 0},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			b := appendi32(tc.dst, tc.x)
			assert.Equal(t, tc.want, b, "bytes do not match")
		})
	}
}

func TestAppendi64(t *testing.T) {
	testCases := []struct {
		desc string
		dst  []byte
		x    int64
		want []byte
	}{
		{
			desc: "0",
			x:    0,
			want: []byte{0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			desc: "1",
			x:    1,
			want: []byte{1, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			desc: "-1",
			x:    -1,
			want: []byte{255, 255, 255, 255, 255, 255, 255, 255},
		},
		{
			desc: "max",
			x:    math.MaxInt64,
			want: []byte{255, 255, 255, 255, 255, 255, 255, 127},
		},
		{
			desc: "min",
			x:    math.MinInt64,
			want: []byte{0, 0, 0, 0, 0, 0, 0, 128},
		},
		{
			desc: "non-empty dst",
			dst:  []byte{0, 1, 2, 3},
			x:    1,
			want: []byte{0, 1, 2, 3, 1, 0, 0, 0, 0, 0, 0, 0},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			b := appendi64(tc.dst, tc.x)
			assert.Equal(t, tc.want, b, "bytes do not match")
		})
	}
}

func TestReadi32(t *testing.T) {
	testCases := []struct {
		desc    string
		src     []byte
		want    int32
		wantRem []byte
		wantOK  bool
	}{
		{
			desc:    "0",
			src:     []byte{0, 0, 0, 0},
			want:    0,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "1",
			src:     []byte{1, 0, 0, 0},
			want:    1,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "-1",
			src:     []byte{255, 255, 255, 255},
			want:    -1,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "max",
			src:     []byte{255, 255, 255, 127},
			want:    math.MaxInt32,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "min",
			src:     []byte{0, 0, 0, 128},
			want:    math.MinInt32,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "non-empty remaining",
			src:     []byte{1, 0, 0, 0, 0, 1, 2, 3},
			want:    1,
			wantRem: []byte{0, 1, 2, 3},
			wantOK:  true,
		},
		{
			desc:    "not enough bytes",
			src:     []byte{0, 1, 2},
			want:    0,
			wantRem: []byte{0, 1, 2},
			wantOK:  false,
		},
		{
			desc:    "nil",
			src:     nil,
			want:    0,
			wantRem: nil,
			wantOK:  false,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			x, rem, ok := readi32(tc.src)
			assert.Equal(t, tc.want, x, "int32 result does not match")
			assert.Equal(t, tc.wantRem, rem, "remaining bytes do not match")
			assert.Equal(t, tc.wantOK, ok, "OK does not match")
		})
	}
}

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
			src:     []byte{0, 0, 0, 0, 0, 0, 0, 0},
			want:    0,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "1",
			src:     []byte{1, 0, 0, 0, 0, 0, 0, 0},
			want:    1,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "-1",
			src:     []byte{255, 255, 255, 255, 255, 255, 255, 255},
			want:    -1,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "max",
			src:     []byte{255, 255, 255, 255, 255, 255, 255, 127},
			want:    math.MaxInt64,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "min",
			src:     []byte{0, 0, 0, 0, 0, 0, 0, 128},
			want:    math.MinInt64,
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			desc:    "non-empty remaining",
			src:     []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3},
			want:    1,
			wantRem: []byte{0, 1, 2, 3},
			wantOK:  true,
		},
		{
			desc:    "not enough bytes",
			src:     []byte{0, 1, 2, 3, 4, 5, 6},
			want:    0,
			wantRem: []byte{0, 1, 2, 3, 4, 5, 6},
			wantOK:  false,
		},
		{
			desc:    "not enough bytes",
			src:     []byte{0, 1, 2, 3, 4, 5, 6},
			want:    0,
			wantRem: []byte{0, 1, 2, 3, 4, 5, 6},
			wantOK:  false,
		},
		{
			desc:    "nil",
			src:     nil,
			want:    0,
			wantRem: nil,
			wantOK:  false,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			x, rem, ok := readi64(tc.src)
			assert.Equal(t, tc.want, x, "int64 result does not match")
			assert.Equal(t, tc.wantRem, rem, "remaining bytes do not match")
			assert.Equal(t, tc.wantOK, ok, "OK does not match")
		})
	}
}
