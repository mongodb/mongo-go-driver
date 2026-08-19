// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"strconv"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/wiremessage"
)

const testIdentifier = "foobar"

var testDocs = [][]byte{
	[]byte("Lorem ipsum dolor sit amet"),
	[]byte("consectetur adipiscing elit"),
}

// newBatches builds a Batches with the test identifier from docs.
func newBatches(docs ...[]byte) *Batches {
	b := &Batches{Identifier: testIdentifier}
	for _, doc := range docs {
		b.Documents = append(b.Documents, bsoncore.Document(doc))
	}
	return b
}

// wantSequence builds the expected AppendBatchSequence output: prefix followed by
// a document sequence section holding docs.
func wantSequence(prefix []byte, docs [][]byte) []byte {
	dst := append([]byte(nil), prefix...)
	dst = wiremessage.AppendMsgSectionType(dst, wiremessage.DocumentSequence)
	idx, dst := bsoncore.ReserveLength(dst)
	dst = append(dst, testIdentifier...)
	dst = append(dst, 0x00)
	for _, doc := range docs {
		dst = append(dst, doc...)
	}
	return bsoncore.UpdateLength(dst, idx, int32(len(dst[idx:])))
}

// wantArray builds the expected AppendBatchArray output using the straightforward
// bsoncore.AppendDocumentElement / strconv.Itoa construction. Comparing against it
// also verifies AppendBatchArray's in-place integer-key encoding.
func wantArray(prefix []byte, docs [][]byte) []byte {
	dst := append([]byte(nil), prefix...)
	idx, dst := bsoncore.AppendArrayElementStart(dst, testIdentifier)
	for i, doc := range docs {
		dst = bsoncore.AppendDocumentElement(dst, strconv.Itoa(i), doc)
	}
	dst, _ = bsoncore.AppendArrayEnd(dst, idx)
	return dst
}

// benchBatches builds a Batches of numDocs identical documents of docSize bytes.
func benchBatches(numDocs, docSize int) *Batches {
	b := &Batches{Identifier: "documents"}
	doc := bsoncore.Document(make([]byte, docSize))
	for i := 0; i < numDocs; i++ {
		b.Documents = append(b.Documents, doc)
	}
	return b
}

func TestAdvancing(t *testing.T) {
	batches := newBatches(testDocs...)
	batches.AdvanceBatches(3)
	size := batches.Size()
	assert.Equal(t, 0, size, "expected Size(): %d, got: %d", 0, size)
}

func TestAppendBatchSequence(t *testing.T) {
	tests := []struct {
		name      string
		docs      [][]byte
		advance   int
		maxCount  int
		sizeLimit int
		want      [][]byte
	}{
		{
			name:      "size limit fits only the first document",
			docs:      testDocs,
			maxCount:  2,
			sizeLimit: len(testDocs[0]) + len(testDocs[1]),
			want:      testDocs[:1],
		},
		{
			name:      "all documents fit",
			docs:      testDocs,
			maxCount:  10,
			sizeLimit: 16 * 1024 * 1024,
			want:      testDocs,
		},
		{
			name:      "offset skips leading documents",
			docs:      testDocs,
			advance:   1,
			maxCount:  10,
			sizeLimit: 16 * 1024 * 1024,
			want:      testDocs[1:],
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batches := newBatches(tt.docs...)
			batches.AdvanceBatches(tt.advance)

			n, got, err := batches.AppendBatchSequence([]byte{42}, tt.maxCount, tt.sizeLimit)
			assert.NoError(t, err)
			assert.Equal(t, len(tt.want), n)
			assert.Equal(t, wantSequence([]byte{42}, tt.want), got)
		})
	}
}

func TestAppendBatchArray(t *testing.T) {
	manyDocs := make([][]byte, 150) // exercises multi-digit array keys
	for i := range manyDocs {
		manyDocs[i] = testDocs[0]
	}

	tests := []struct {
		name      string
		docs      [][]byte
		advance   int
		maxCount  int
		sizeLimit int
		want      [][]byte
	}{
		{
			name:      "size limit fits only the first document",
			docs:      testDocs,
			maxCount:  2,
			sizeLimit: len(testDocs[0]) + len(testDocs[1]),
			want:      testDocs[:1],
		},
		{
			name:      "all documents fit",
			docs:      testDocs,
			maxCount:  10,
			sizeLimit: 16 * 1024 * 1024,
			want:      testDocs,
		},
		{
			name:      "offset skips leading documents",
			docs:      testDocs,
			advance:   1,
			maxCount:  10,
			sizeLimit: 16 * 1024 * 1024,
			want:      testDocs[1:],
		},
		{
			name:      "many elements with multi-digit keys",
			docs:      manyDocs,
			maxCount:  len(manyDocs),
			sizeLimit: 16 * 1024 * 1024,
			want:      manyDocs,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batches := newBatches(tt.docs...)
			batches.AdvanceBatches(tt.advance)

			n, got, err := batches.AppendBatchArray([]byte{42}, tt.maxCount, tt.sizeLimit)
			assert.NoError(t, err)
			assert.Equal(t, len(tt.want), n)
			assert.Equal(t, wantArray([]byte{42}, tt.want), got)
		})
	}
}

func BenchmarkAppendBatch(b *testing.B) {
	batches := benchBatches(1000, 256)

	benchmarks := []struct {
		name string
		fn   func(*Batches, []byte, int, int) (int, []byte, error)
	}{
		{"Sequence", (*Batches).AppendBatchSequence},
		{"Array", (*Batches).AppendBatchArray},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				batches.offset = 0
				// A nil buffer measures the growth a BulkWrite pays per operation.
				if _, _, err := bm.fn(batches, nil, len(batches.Documents), 16*1024*1024); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
