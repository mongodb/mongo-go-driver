// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package wiremessage

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// Query represents the OP_QUERY message of the MongoDB wire protocol.
type Query struct {
	MsgHeader            Header
	Flags                QueryFlag
	FullCollectionName   string
	NumberToSkip         int32
	NumberToReturn       int32
	Query                bson.Reader
	ReturnFieldsSelector bson.Reader
}

// MarshalWireMessage implements the Marshaler and WireMessage interfaces.
//
// See AppendWireMessage for a description of the rules this method follows.
func (q Query) MarshalWireMessage() ([]byte, error) {
	b := make([]byte, 0, q.Len())
	return q.AppendWireMessage(b)
}

// ValidateWireMessage implements the Validator and WireMessage interfaces.
func (q Query) ValidateWireMessage() error {
	if int(q.MsgHeader.MessageLength) != q.Len() {
		return errors.New("incorrect header: message length is not correct")
	}
	if q.MsgHeader.OpCode != OpQuery {
		return errors.New("incorrect header: op code is not OpQuery")
	}
	if strings.Index(q.FullCollectionName, ".") == -1 {
		return errors.New("incorrect header: collection name does not contain a dot")
	}
	if q.Query != nil && len(q.Query) > 0 {
		_, err := q.Query.Validate()
		if err != nil {
			return err
		}
	}

	if q.ReturnFieldsSelector != nil && len(q.ReturnFieldsSelector) > 0 {
		_, err := q.ReturnFieldsSelector.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

// AppendWireMessage implements the Appender and WireMessage interfaces.
//
// AppendWireMessage will set the MessageLength property of the MsgHeader
// if it is zero. It will also set the OpCode to OpQuery if the OpCode is
// zero. If either of these properties are non-zero and not correct, this
// method will return both the []byte with the wire message appended to it
// and an invalid header error.
func (q Query) AppendWireMessage(b []byte) ([]byte, error) {
	var err error
	err = q.MsgHeader.SetDefaults(q.Len(), OpQuery)

	b = q.MsgHeader.AppendHeader(b)
	b = appendInt32(b, int32(q.Flags))
	b = appendCString(b, q.FullCollectionName)
	b = appendInt32(b, q.NumberToSkip)
	b = appendInt32(b, q.NumberToReturn)
	b = append(b, q.Query...)
	b = append(b, q.ReturnFieldsSelector...)
	return b, err
}

// String implements the fmt.Stringer interface.
func (q Query) String() string {
	return fmt.Sprintf(
		`OP_QUERY{MsgHeader: %s, Flags: %s, FullCollectionname: %s, NumberToSkip: %d, NumberToReturn: %d, Query: %s, ReturnFieldsSelector: %s}`,
		q.MsgHeader, q.Flags, q.FullCollectionName, q.NumberToSkip, q.NumberToReturn, q.Query, q.ReturnFieldsSelector,
	)
}

// Len implements the WireMessage interface.
func (q Query) Len() int {
	// Header + Flags + CollectionName + Null Byte + Skip + Return + Query + ReturnFieldsSelector
	return 16 + 4 + len(q.FullCollectionName) + 1 + 4 + 4 + len(q.Query) + len(q.ReturnFieldsSelector)
}

// UnmarshalWireMessage implements the Unmarshaler interface.
func (q *Query) UnmarshalWireMessage(b []byte) error {
	var err error
	q.MsgHeader, err = ReadHeader(b, 0)
	if err != nil {
		return err
	}
	if len(b) < int(q.MsgHeader.MessageLength) {
		return Error{Type: ErrOpQuery, Message: "[]byte too small"}
	}

	q.Flags = QueryFlag(readInt32(b, 16))
	q.FullCollectionName, err = readCString(b, 20)
	if err != nil {
		return err
	}
	pos := 20 + len(q.FullCollectionName) + 1
	q.NumberToSkip = readInt32(b, int32(pos))
	pos += 4
	q.NumberToReturn = readInt32(b, int32(pos))
	pos += 4

	var size int
	var wmerr Error
	q.Query, size, wmerr = readDocument(b, int32(pos))
	if wmerr.Message != "" {
		wmerr.Type = ErrOpQuery
		return wmerr
	}
	pos += size
	if pos < len(b) {
		q.ReturnFieldsSelector, size, wmerr = readDocument(b, int32(pos))
		if wmerr.Message != "" {
			wmerr.Type = ErrOpQuery
			return wmerr
		}
		pos += size
	}

	return nil
}

// AcknowledgedWrite returns true if this command represents an acknowledged write
func (q *Query) AcknowledgedWrite() bool {
	wcElem, err := q.Query.Lookup("writeConcern")
	if err != nil {
		// no wc --> ack
		return true
	}

	return writeconcern.AcknowledgedElement(wcElem)
}

// QueryFlag represents the flags on an OP_QUERY message.
type QueryFlag int32

// These constants represent the individual flags on an OP_QUERY message.
const (
	_ QueryFlag = 1 << iota
	TailableCursor
	SlaveOK
	OplogReplay
	NoCursorTimeout
	AwaitData
	Exhaust
	Partial
)

// String implements the fmt.Stringer interface.
func (qf QueryFlag) String() string {
	strs := make([]string, 0)
	if qf&TailableCursor == TailableCursor {
		strs = append(strs, "TailableCursor")
	}
	if qf&SlaveOK == SlaveOK {
		strs = append(strs, "SlaveOK")
	}
	if qf&OplogReplay == OplogReplay {
		strs = append(strs, "OplogReplay")
	}
	if qf&NoCursorTimeout == NoCursorTimeout {
		strs = append(strs, "NoCursorTimeout")
	}
	if qf&AwaitData == AwaitData {
		strs = append(strs, "AwaitData")
	}
	if qf&Exhaust == Exhaust {
		strs = append(strs, "Exhaust")
	}
	if qf&Partial == Partial {
		strs = append(strs, "Partial")
	}
	str := "["
	str += strings.Join(strs, ", ")
	str += "]"
	return str
}
