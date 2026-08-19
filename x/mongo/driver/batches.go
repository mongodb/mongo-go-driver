// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"io"
	"strconv"

	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/wiremessage"
)

// Batches contains the necessary information to batch split an operation. This is only used for write
// operations.
type Batches struct {
	Identifier string
	Documents  []bsoncore.Document
	Ordered    *bool

	offset int
}

var _ OperationBatches = &Batches{}

// AppendBatchSequence appends dst with document sequence of batches as long as the limits of max count, max
// document size, or total size allows. It returns the number of batches appended, the new appended slice, and
// any error raised. It returns the original input slice if nothing can be appended within the limits.
func (b *Batches) AppendBatchSequence(dst []byte, maxCount, totalSize int) (int, []byte, error) {
	if b.Size() == 0 {
		return 0, dst, io.EOF
	}
	l := len(dst)
	var idx int32
	dst = wiremessage.AppendMsgSectionType(dst, wiremessage.DocumentSequence)
	idx, dst = bsoncore.ReserveLength(dst)
	dst = append(dst, b.Identifier...)
	dst = append(dst, 0x00)

	// First pass: total the documents that fit within the limits so dst can be
	// grown once instead of reallocating on each append.
	size := len(dst)
	n, docsSize := 0, 0
	for n < maxCount && b.offset+n < len(b.Documents) {
		doc := b.Documents[b.offset+n]
		if size+len(doc) > totalSize {
			break
		}
		size += len(doc)
		docsSize += len(doc)
		n++
	}
	if n == 0 {
		return 0, dst[:l], nil
	}

	// Reserve once; no-op when dst already has room.
	if cap(dst) < len(dst)+docsSize {
		dst = append(make([]byte, 0, len(dst)+docsSize), dst...)
	}
	for _, doc := range b.Documents[b.offset : b.offset+n] {
		dst = append(dst, doc...)
	}

	dst = bsoncore.UpdateLength(dst, idx, int32(len(dst[idx:])))
	return n, dst, nil
}

// AppendBatchArray appends dst with array of batches as long as the limits of max count, max document size, or
// total size allows. It returns the number of batches appended, the new appended slice, and any error raised. It
// returns the original input slice if nothing can be appended within the limits.
func (b *Batches) AppendBatchArray(dst []byte, maxCount, totalSize int) (int, []byte, error) {
	if b.Size() == 0 {
		return 0, dst, io.EOF
	}
	l := len(dst)
	aidx, dst := bsoncore.AppendArrayElementStart(dst, b.Identifier)

	// First pass: total the bytes the elements that fit will need so dst can be
	// grown once. Each element is a type byte, the decimal index key, a null
	// terminator and the document; keyLen is the current index's digit count,
	// bumped as n crosses each power of ten.
	size := len(dst)
	n, appendSize := 0, 0
	keyLen, nextPow10 := 1, 10
	for n < maxCount && b.offset+n < len(b.Documents) {
		doc := b.Documents[b.offset+n]
		if size+len(doc) > totalSize {
			break
		}
		size += len(doc)
		if n == nextPow10 {
			keyLen++
			nextPow10 *= 10
		}
		appendSize += 2 + keyLen + len(doc)
		n++
	}
	if n == 0 {
		return 0, dst[:l], nil
	}
	appendSize++ // closing byte written by AppendArrayEnd

	// Reserve once; no-op when dst already has room.
	if cap(dst) < len(dst)+appendSize {
		dst = append(make([]byte, 0, len(dst)+appendSize), dst...)
	}
	for j := 0; j < n; j++ {
		dst = appendDocumentElementInt(dst, j, b.Documents[b.offset+j])
	}

	var err error
	dst, err = bsoncore.AppendArrayEnd(dst, aidx)
	if err != nil {
		return 0, nil, err
	}
	return n, dst, nil
}

// appendDocumentElementInt is bsoncore.AppendDocumentElement with an integer
// key, formatting the key in place to avoid allocating a string per element.
func appendDocumentElementInt(dst []byte, key int, doc []byte) []byte {
	dst = bsoncore.AppendType(dst, bsoncore.TypeEmbeddedDocument)
	dst = strconv.AppendInt(dst, int64(key), 10)
	dst = append(dst, 0x00)
	return bsoncore.AppendDocument(dst, doc)
}

// IsOrdered indicates if the batches are ordered.
func (b *Batches) IsOrdered() *bool {
	return b.Ordered
}

// AdvanceBatches advances the batches with the given input.
func (b *Batches) AdvanceBatches(n int) {
	b.offset += n
	if b.offset > len(b.Documents) {
		b.offset = len(b.Documents)
	}
}

// Size returns the size of batches remained.
func (b *Batches) Size() int {
	if b.offset > len(b.Documents) {
		return 0
	}
	return len(b.Documents) - b.offset
}
