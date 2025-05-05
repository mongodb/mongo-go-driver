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
// any error raised. It returns the origenal input slice if nothing can be appends within the limits.
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
	var size int
	var n int
	for i := b.offset; i < len(b.Documents); i++ {
		if n == maxCount {
			break
		}
		doc := b.Documents[i]
		size += len(doc)
		if size > totalSize {
			break
		}
		dst = append(dst, doc...)
		n++
	}
	if n == 0 {
		return 0, dst[:l], nil
	}
	dst = bsoncore.UpdateLength(dst, idx, int32(len(dst[idx:])))
	return n, dst, nil
}

// AppendBatchArray appends dst with array of batches as long as the limits of max count, max document size, or
// total size allows. It returns the number of batches appended, the new appended slice, and any error raised. It
// returns the origenal input slice if nothing can be appends within the limits.
func (b *Batches) AppendBatchArray(dst []byte, maxCount, totalSize int) (int, []byte, error) {
	if b.Size() == 0 {
		return 0, dst, io.EOF
	}
	l := len(dst)
	aidx, dst := bsoncore.AppendArrayElementStart(dst, b.Identifier)
	var size int
	var n int
	for i := b.offset; i < len(b.Documents); i++ {
		if n == maxCount {
			break
		}
		doc := b.Documents[i]
		size += len(doc)
		if size > totalSize {
			break
		}
		dst = bsoncore.AppendDocumentElement(dst, strconv.Itoa(n), doc)
		n++
	}
	if n == 0 {
		return 0, dst[:l], nil
	}
	var err error
	dst, err = bsoncore.AppendArrayEnd(dst, aidx)
	if err != nil {
		return 0, nil, err
	}
	return n, dst, nil
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
