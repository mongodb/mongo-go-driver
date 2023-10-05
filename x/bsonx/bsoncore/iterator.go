// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"errors"
	"io"

	"go.mongodb.org/mongo-driver/bson/bsontype"
)

// ErrCorruptedDocument is returned when a full document couldn't be read from the sequence.
var ErrCorruptedDocument = errors.New("invalid DocumentSequence: corrupted document")

// ErrNonDocument is returned when a DocumentSequence contains a non-document BSON value.
var ErrNonDocument = errors.New("invalid DocumentSequence: a non-document value was found in sequence")

// ErrInvalidDocumentSequenceStyle is returned when an unknown DocumentSequenceStyle is set on a
// DocumentSequence.
var ErrInvalidDocumentSequenceStyle = errors.New("invalid DocumentSequenceStyle")

type Iterator struct {
	Data Array
	pos  int
}

func (iter *Iterator) Count() int {
	if iter == nil {
		return 0
	}

	_, rem, ok := ReadLength(iter.Data)
	if !ok {
		return 0
	}

	var count int
	for len(rem) > 1 {
		_, rem, ok = ReadElement(rem)
		if !ok {
			return 0
		}
		count++
	}
	return count
}

func (iter *Iterator) Empty() bool {
	return len(iter.Data) <= 5
}

func (iter *Iterator) Reset() {
	iter.pos = 0
}

// TODO: This will be removed, just a convenience ATM to test change streams
func (iter *Iterator) Documents() ([]Document, error) {
	if len(iter.Data) == 0 {
		return nil, nil
	}

	vals, err := iter.Data.Values()
	if err != nil {
		return nil, ErrCorruptedDocument
	}

	docs := make([]Document, 0, len(vals))
	for _, v := range vals {
		if v.Type != bsontype.EmbeddedDocument {
			return nil, ErrNonDocument
		}
		docs = append(docs, v.Data)
	}

	return docs, nil
}

func (iter *Iterator) Next() (*Value, error) {
	if iter == nil || iter.pos >= len(iter.Data) {
		return nil, io.EOF
	}

	if iter.pos < 4 {
		if len(iter.Data) < 4 {
			return nil, ErrCorruptedDocument
		}

		iter.pos = 4 // Skip the length of the document
	}

	if len(iter.Data[iter.pos:]) == 1 && iter.Data[iter.pos] == 0x00 {
		return nil, io.EOF // At the end of the document
	}

	elem, _, ok := ReadElement(iter.Data[iter.pos:])
	if !ok {
		return nil, ErrCorruptedDocument
	}

	iter.pos += len(elem)
	val := elem.Value()

	return &val, nil
}
