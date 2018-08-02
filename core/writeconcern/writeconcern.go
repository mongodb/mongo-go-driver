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

// MarshalBSONElement marshals the write concern into a *bson.Element.
func (wc *WriteConcern) MarshalBSONElement() (*bson.Element, error) {
	if !wc.IsValid() {
		return nil, ErrInconsistent
	}

	var elems []*bson.Element

	if wc.w != nil {
		switch t := wc.w.(type) {
		case int:
			if t < 0 {
				return nil, ErrNegativeW
			}

			elems = append(elems, bson.EC.Int32("w", int32(t)))
		case string:
			elems = append(elems, bson.EC.String("w", t))
		}
	}

	if wc.j {
		elems = append(elems, bson.EC.Boolean("j", wc.j))
	}

	if wc.wTimeout < 0 {
		return nil, ErrNegativeWTimeout
	}

	if wc.wTimeout != 0 {
		elems = append(elems, bson.EC.Int64("wtimeout", int64(wc.wTimeout/time.Millisecond)))
	}

	return bson.EC.SubDocumentFromElements("writeConcern", elems...), nil
}

// AcknowledgedElement returns true if a BSON element for a write concern represents an acknowledged write concern.
// The element's value must be a document representing a write concern.
func AcknowledgedElement(elem *bson.Element) bool {
	wcDoc := elem.Value().MutableDocument()
	wVal, err := wcDoc.LookupErr("w")
	if err != nil {
		// key w not found --> acknowledged
		return true
	}

	return wVal.Int32() != 0
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
