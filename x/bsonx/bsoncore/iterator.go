// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"errors"
	"fmt"
	"io"
)

// errCorruptedDocument is returned when a full document couldn't be read from
// the sequence.
var errCorruptedDocument = errors.New("invalid DocumentSequence: corrupted document")

// Iterator maintains a list of BSON values and keeps track of the current
// position in relation to its Next() method.
type Iterator struct {
	List Array // List of BSON values
	pos  int   // The position of the iterator in the list in reference to Next()
}

// Count returned the number of elements in the iterator's list.
func (iter *Iterator) Count() int {
	if iter == nil {
		return 0
	}

	_, rem, ok := ReadLength(iter.List)
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

// Empty returns true if the iterator's list is empty.
func (iter *Iterator) Empty() bool {
	return len(iter.List) <= 5
}

// Reset will reset the iteration point for the Next method to the beginning of
// the list.
func (iter *Iterator) Reset() {
	iter.pos = 0
}

// Documents traverses the list as documents and returns them. This method
// assumes that the underlying list is composed of documents and will return
// an error otherwise.
func (iter *Iterator) Documents() ([]Document, error) {
	if iter == nil || len(iter.List) == 0 {
		return nil, nil
	}

	vals, err := iter.List.Values()
	if err != nil {
		return nil, errCorruptedDocument
	}

	docs := make([]Document, 0, len(vals))
	for _, v := range vals {
		if v.Type != TypeEmbeddedDocument {
			return nil, fmt.Errorf("invalid DocumentSequence: a non-document value was found in sequence")
		}

		docs = append(docs, v.Data)
	}

	return docs, nil
}

// Next retrieves the next value from the list and returns it. This method will
// return io.EOF when it has reached the end of the list.
func (iter *Iterator) Next() (*Value, error) {
	if iter == nil || iter.pos >= len(iter.List) {
		return nil, io.EOF
	}

	if iter.pos < 4 {
		if len(iter.List) < 4 {
			return nil, errCorruptedDocument
		}

		iter.pos = 4 // Skip the length of the document
	}

	rem := iter.List[iter.pos:]
	if len(rem) == 1 && rem[0] == 0x00 {
		return nil, io.EOF // At the end of the document
	}

	elem, _, ok := ReadElement(rem)
	if !ok {
		return nil, errCorruptedDocument
	}

	iter.pos += len(elem)
	val := elem.Value()

	return &val, nil
}
