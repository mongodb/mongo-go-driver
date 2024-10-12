// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"io"
	"strconv"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

// Batches contains the necessary information to batch split an operation. This is only used for write
// operations.
type Batches struct {
	Identifier string
	Documents  []bsoncore.Document
	Ordered    *bool

	offset int
}

func (b *Batches) AppendBatchSequence(dst []byte, maxCount, maxDocSize, totalSize int) (int, []byte, error) {
	if b.End() {
		return 0, dst, io.EOF
	}
	l := len(dst)
	var idx int32
	dst = wiremessage.AppendMsgSectionType(dst, wiremessage.DocumentSequence)
	idx, dst = bsoncore.ReserveLength(dst)
	dst = append(dst, b.Identifier...)
	dst = append(dst, 0x00)
	size := len(dst) - l
	var n int
	for i := b.offset; i < len(b.Documents); i++ {
		if n == maxCount {
			break
		}
		doc := b.Documents[i]
		if len(doc) > maxDocSize {
			break
		}
		size += len(doc)
		if size >= totalSize {
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

func (b *Batches) AppendBatchArray(dst []byte, maxCount, maxDocSize, totalSize int) (int, []byte, error) {
	if b.End() {
		return 0, dst, io.EOF
	}
	l := len(dst)
	aidx, dst := bsoncore.AppendArrayElementStart(dst, b.Identifier)
	size := len(dst) - l
	var n int
	for i := b.offset; i < len(b.Documents); i++ {
		if n == maxCount {
			break
		}
		doc := b.Documents[i]
		if len(doc) > maxDocSize {
			break
		}
		size += len(doc)
		if size >= totalSize {
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

func (b *Batches) IsOrdered() *bool {
	return b.Ordered
}

func (b *Batches) AdvanceBatches(n int) {
	b.offset += n
}

func (b *Batches) End() bool {
	return len(b.Documents) <= b.offset
}
