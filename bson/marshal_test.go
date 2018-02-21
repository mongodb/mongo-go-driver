// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

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

	bson, err := Marshal(before)
	require.NoError(t, err)

	after := NewDocument()
	require.NoError(t, Unmarshal(bson, after))

	require.True(t, before.Equal(after))
}

func TestMarshal_roundtripWithUnmarshalDoc(t *testing.T) {
	before := NewDocument(
		EC.String("foo", "bar"),
		EC.Int32("baz", -27),
		EC.ArrayFromElements("bing", VC.Null(), VC.Regex("word", "i")),
	)

	bson, err := Marshal(before)
	require.NoError(t, err)

	after, err := UnmarshalDocument(bson)
	require.NoError(t, err)

	require.True(t, before.Equal(after))
}
