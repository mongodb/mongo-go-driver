// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package msg

import (
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
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
		documentsBytes: m.DocumentsBytes,
	}
}

// ReplyIter iterates over the documents returned
// in a Reply.
type ReplyIter struct {
	documentsBytes []byte
	pos            int

	err error
}

// NextBytes reads the next document and returns it as a bson.Reader.
func (i *ReplyIter) NextBytes() (bson.Reader, error) {
	if i.pos >= len(i.documentsBytes) {
		return nil, nil
	}

	if i.pos+4 >= len(i.documentsBytes) {
		return nil, fmt.Errorf("malformed document, only %d bytes available for reading but need at least 4", len(i.documentsBytes)-i.pos)
	}

	n := int(readInt32(i.documentsBytes, int32(i.pos)))

	if len(i.documentsBytes)-i.pos < n {
		return nil, fmt.Errorf("needed %d bytes to read document, but only had %d", n, len(i.documentsBytes)-i.pos)
	}

	// Making a copy since we essentially do that when we unmarshal.
	r := make(bson.Reader, n)
	copy(r, i.documentsBytes[i.pos:i.pos+n])

	_, err := r.Validate()
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
