// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/wiremessage"
)

func newTestBatches(t *testing.T) *Batches {
	t.Helper()
	return &Batches{
		Identifier: "foobar",
		Documents: []bsoncore.Document{
			[]byte("Lorem ipsum dolor sit amet"),
			[]byte("consectetur adipiscing elit"),
		},
	}
}

func TestAdvancing(t *testing.T) {
	batches := newTestBatches(t)
	batches.AdvanceBatches(3)
	size := batches.Size()
	assert.Equal(t, 0, size, "expected Size(): %d, got: %d", 1, size)
}

func TestAppendBatchSequence(t *testing.T) {
	batches := newTestBatches(t)

	got := []byte{42}
	var n int
	var err error
	n, got, err = batches.AppendBatchSequence(got, 2, len(batches.Documents[0]))
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	var idx int32
	dst := []byte{42}
	dst = wiremessage.AppendMsgSectionType(dst, wiremessage.DocumentSequence)
	idx, dst = bsoncore.ReserveLength(dst)
	dst = append(dst, "foobar"...)
	dst = append(dst, 0x00)
	dst = append(dst, "Lorem ipsum dolor sit amet"...)
	dst = bsoncore.UpdateLength(dst, idx, int32(len(dst[idx:])))
	assert.Equal(t, dst, got)
}

func TestAppendBatchArray(t *testing.T) {
	batches := newTestBatches(t)

	got := []byte{42}
	var n int
	var err error
	n, got, err = batches.AppendBatchArray(got, 2, len(batches.Documents[0]))
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	var idx int32
	dst := []byte{42}
	idx, dst = bsoncore.AppendArrayElementStart(dst, "foobar")
	dst = bsoncore.AppendDocumentElement(dst, "0", []byte("Lorem ipsum dolor sit amet"))
	dst, err = bsoncore.AppendArrayEnd(dst, idx)
	assert.NoError(t, err)
	assert.Equal(t, dst, got)
}
