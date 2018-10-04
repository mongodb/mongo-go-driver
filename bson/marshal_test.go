// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/stretchr/testify/require"
)

func TestMarshalAppendWithRegistry(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			dst := make([]byte, 0, 1024)
			var reg *bsoncodec.Registry
			if tc.reg != nil {
				reg = tc.reg
			} else {
				reg = DefaultRegistry
			}
			got, err := MarshalAppendWithRegistry(reg, dst, tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", got, tc.want)
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}

func TestMarshalWithRegistry(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			var reg *bsoncodec.Registry
			if tc.reg != nil {
				reg = tc.reg
			} else {
				reg = DefaultRegistry
			}
			got, err := MarshalWithRegistry(reg, tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", got, tc.want)
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}

func TestMarshalAppend(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.reg != nil {
				t.Skip() // test requires custom registry
			}
			dst := make([]byte, 0, 1024)
			got, err := MarshalAppend(dst, tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", got, tc.want)
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}

func TestMarshal_roundtripFromBytes(t *testing.T) {
	before := []byte{
		// length
		0x1c, 0x0, 0x0, 0x0,

		// --- begin array ---

		// type - document
		0x3,
		// key - "foo"
		0x66, 0x6f, 0x6f, 0x0,

		// length
		0x12, 0x0, 0x0, 0x0,
		// type - string
		0x2,
		// key - "bar"
		0x62, 0x61, 0x72, 0x0,
		// value - string length
		0x4, 0x0, 0x0, 0x0,
		// value - "baz"
		0x62, 0x61, 0x7a, 0x0,

		// null terminator
		0x0,

		// --- end array ---

		// null terminator
		0x0,
	}

	doc := NewDocument()
	require.NoError(t, Unmarshal(before, doc))

	after, err := Marshal(doc)
	require.NoError(t, err)

	require.True(t, bytes.Equal(before, after))
}

func TestMarshal_roundtripFromDoc(t *testing.T) {
	before := NewDocument(
		EC.String("foo", "bar"),
		EC.Int32("baz", -27),
		EC.ArrayFromElements("bing", VC.Null(), VC.Regex("word", "i")),
	)

	b, err := Marshal(before)
	require.NoError(t, err)

	after := NewDocument()
	require.NoError(t, Unmarshal(b, &after))

	require.True(t, before.Equal(after))
}
