// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package writeconcern

import (
	"errors"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"github.com/mongodb/mongo-go-driver/x/bsonx/bsoncore"
)

// ErrInconsistent indicates that an inconsistent write concern was specified.
var ErrInconsistent = errors.New("a write concern cannot have both w=0 and j=true")

// ErrNegativeW indicates that a negative integer `w` field was specified.
var ErrNegativeW = errors.New("write concern `w` field cannot be a negative number")

// ErrNegativeWTimeout indicates that a negative WTimeout was specified.
var ErrNegativeWTimeout = errors.New("write concern `wtimeout` field cannot be negative")

// WriteConcern describes the level of acknowledgement requested from MongoDB for write operations
// to a standalone mongod or to replica sets or to sharded clusters.
type WriteConcern struct {
	w        interface{}
	j        bool
	wTimeout time.Duration
}

// Option is an option to provide when creating a ReadConcern.
type Option func(concern *WriteConcern)

// New constructs a new WriteConcern.
func New(options ...Option) *WriteConcern {
	concern := &WriteConcern{}

	for _, option := range options {
		option(concern)
	}

	return concern
}

// W requests acknowledgement that write operations propagate to the specified number of mongod
// instances.
func W(w int) Option {
	return func(concern *WriteConcern) {
		concern.w = w
	}
}

// WMajority requests acknowledgement that write operations propagate to the majority of mongod
// instances.
func WMajority() Option {
	return func(concern *WriteConcern) {
		concern.w = "majority"
	}
}

// WTagSet requests acknowledgement that write operations propagate to the specified mongod
// instance.
func WTagSet(tag string) Option {
	return func(concern *WriteConcern) {
		concern.w = tag
	}
}

// J requests acknowledgement from MongoDB that write operations are written to
// the journal.
func J(j bool) Option {
	return func(concern *WriteConcern) {
		concern.j = j
	}
}

// WTimeout specifies specifies a time limit for the write concern.
func WTimeout(d time.Duration) Option {
	return func(concern *WriteConcern) {
		concern.wTimeout = d
	}
}

// MarshalBSONElement marshals the write concern into a *bsonx.Element.
func (wc *WriteConcern) MarshalBSONElement() (bsonx.Elem, error) {
	if !wc.IsValid() {
		return bsonx.Elem{}, ErrInconsistent
	}

	elems := bsonx.Doc{}

	if wc.w != nil {
		switch t := wc.w.(type) {
		case int:
			if t < 0 {
				return bsonx.Elem{}, ErrNegativeW
			}

			elems = append(elems, bsonx.Elem{"w", bsonx.Int32(int32(t))})
		case string:
			elems = append(elems, bsonx.Elem{"w", bsonx.String(t)})
		}
	}

	if wc.j {
		elems = append(elems, bsonx.Elem{"j", bsonx.Boolean(wc.j)})
	}

	if wc.wTimeout < 0 {
		return bsonx.Elem{}, ErrNegativeWTimeout
	}

	if wc.wTimeout != 0 {
		elems = append(elems, bsonx.Elem{"wtimeout", bsonx.Int64(int64(wc.wTimeout / time.Millisecond))})
	}

	return bsonx.Elem{"writeConcern", bsonx.Document(elems)}, nil
}

// AcknowledgedElement returns true if a BSON element for a write concern represents an acknowledged write concern.
// The element's value must be a document representing a write concern.
func AcknowledgedElement(elem bsonx.Elem) bool {
	wcDoc := elem.Value.Document()
	wVal, err := wcDoc.LookupErr("w")
	if err != nil {
		// key w not found --> acknowledged
		return true
	}

	return wVal.Int32() != 0
}

// AcknowledgedElementRaw returns true if a BSON RawValue for a write concern represents an acknowledged write concern.
// The element's value must be a document representing a write concern.
func AcknowledgedElementRaw(rawv bson.RawValue) bool {
	doc, ok := bsoncore.Value{Type: rawv.Type, Data: rawv.Value}.DocumentOK()
	if !ok {
		return false
	}

	val, err := doc.LookupErr("w")
	if err != nil {
		// key w not found --> acknowledged
		return true
	}

	i32, ok := val.Int32OK()
	if !ok {
		return false
	}
	return i32 != 0
}

// Acknowledged indicates whether or not a write with the given write concern will be acknowledged.
func (wc *WriteConcern) Acknowledged() bool {
	if wc == nil || wc.j {
		return true
	}

	switch v := wc.w.(type) {
	case int:
		if v == 0 {
			return false
		}
	}

	return true
}

// IsValid checks whether the write concern is invalid.
func (wc *WriteConcern) IsValid() bool {
	if !wc.j {
		return true
	}

	switch v := wc.w.(type) {
	case int:
		if v == 0 {
			return false
		}
	}

	return true
}

// AckWrite returns true if a write concern represents an acknowledged write
func AckWrite(wc *WriteConcern) bool {
	return wc == nil || wc.Acknowledged()
}
