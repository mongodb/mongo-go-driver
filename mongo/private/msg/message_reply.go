// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package msg

import (
	"fmt"

	"github.com/skriptble/wilson/bson"
)

// Reply is a message received from the server.
type Reply struct {
	ReqID          int32
	RespTo         int32
	ResponseFlags  ReplyFlags
	CursorID       int64
	StartingFrom   int32
	NumberReturned int32
	DocumentsBytes []byte

	unmarshaller documentUnmarshaller
	partitioner  documentPartitioner
}

// ResponseTo gets the request id the message was in response to.
func (m *Reply) ResponseTo() int32 { return m.RespTo }

type documentUnmarshaller func([]byte, interface{}) error
type documentPartitioner func([]byte) (int, error)

// ReplyFlags are the flags in a Reply.
type ReplyFlags int32

// ReplayMessageFlags constants.
const (
	CursorNotFound ReplyFlags = 1 << iota
	QueryFailure
	_
	AwaitCapable
)

// Iter returns a ReplyIter to iterate over each document
// returned by the server.
func (m *Reply) Iter() *ReplyIter {
	return &ReplyIter{
		unmarshaller:   m.unmarshaller,
		partitioner:    m.partitioner,
		documentsBytes: m.DocumentsBytes,
	}
}

// ReplyIter iterates over the documents returned
// in a Reply.
type ReplyIter struct {
	unmarshaller   documentUnmarshaller
	partitioner    documentPartitioner
	documentsBytes []byte
	pos            int

	err error
}

// One reads a single document from the iterator.
func (i *ReplyIter) One(result interface{}) (bool, error) {
	if !i.Next(result) {
		return false, i.err
	}

	return true, nil
}

// Next marshals the next document into the provided result and returns
// a value indicating whether or not it was successful.
func (i *ReplyIter) Next(result interface{}) bool {
	if i.pos >= len(i.documentsBytes) {
		return false
	}
	n, err := i.partitioner(i.documentsBytes[i.pos:])
	if err != nil {
		i.err = err
		return false
	}

	if len(i.documentsBytes)-i.pos < n {
		i.err = fmt.Errorf("needed %d bytes to read document, but only had %d", n, len(i.documentsBytes)-i.pos)
		return false
	}

	err = i.unmarshaller(i.documentsBytes[i.pos:i.pos+n], result)
	if err != nil {
		i.err = err
		return false
	}

	i.pos += n
	return true
}

// TODO(skriptble): This method is pretty gross and is a mesh of Next and
// Decode. When we're fixing up this part of the library we should fix this.
func (i *ReplyIter) DecodeBytes() (bson.Reader, error) {
	if i.pos >= len(i.documentsBytes) {
		return nil, nil
	}
	n, err := i.partitioner(i.documentsBytes[i.pos:])
	if err != nil {
		return nil, err
	}

	if len(i.documentsBytes)-i.pos < n {
		return nil, fmt.Errorf("needed %d bytes to read document, but only had %d", n, len(i.documentsBytes)-i.pos)
	}

	// Making a copy since we essentially do that when we unmarshal.
	r := make(bson.Reader, n)
	copy(r, i.documentsBytes[i.pos:i.pos+n])

	_, err = r.Validate()
	if err != nil {
		return nil, err
	}

	i.pos += n
	return r, nil
}

// Err indicates if there was an error unmarshalling the last document
// attempted.
func (i *ReplyIter) Err() error {
	return i.err
}
