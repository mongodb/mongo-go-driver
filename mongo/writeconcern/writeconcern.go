// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package writeconcern

import (
	"errors"
	"time"

	"github.com/10gen/mongo-go-driver/bson"
)

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
func WMajority(w int) Option {
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

// GetBSON is used by the BSON library to serialize the WriteConcern into a bson.M.
func (wc *WriteConcern) GetBSON() (interface{}, error) {
	if !wc.IsValid() {
		return nil, errors.New("a write concern cannot have both w=0 and j=true")
	}

	doc := make(bson.M)

	if wc.w != nil {
		doc["w"] = wc.w
	}

	if wc.j {
		doc["j"] = wc.j
	}

	if wc.wTimeout != 0 {
		doc["wtimeout"] = wc.wTimeout / time.Millisecond
	}

	return doc, nil
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
