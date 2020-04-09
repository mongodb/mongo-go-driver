// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package readpref // import "go.mongodb.org/mongo-driver/mongo/readpref"

import (
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/tag"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var (
	errInvalidReadPreference = errors.New("can not specify tags, max staleness, or hedge with mode primary")
)

var primary = ReadPref{mode: PrimaryMode}

// Primary constructs a read preference with a PrimaryMode.
func Primary() *ReadPref {
	return &primary
}

// PrimaryPreferred constructs a read preference with a PrimaryPreferredMode.
func PrimaryPreferred(opts ...Option) *ReadPref {
	// New only returns an error with a mode of Primary
	rp, _ := New(PrimaryPreferredMode, opts...)
	return rp
}

// SecondaryPreferred constructs a read preference with a SecondaryPreferredMode.
func SecondaryPreferred(opts ...Option) *ReadPref {
	// New only returns an error with a mode of Primary
	rp, _ := New(SecondaryPreferredMode, opts...)
	return rp
}

// Secondary constructs a read preference with a SecondaryMode.
func Secondary(opts ...Option) *ReadPref {
	// New only returns an error with a mode of Primary
	rp, _ := New(SecondaryMode, opts...)
	return rp
}

// Nearest constructs a read preference with a NearestMode.
func Nearest(opts ...Option) *ReadPref {
	// New only returns an error with a mode of Primary
	rp, _ := New(NearestMode, opts...)
	return rp
}

// New creates a new ReadPref.
func New(mode Mode, opts ...Option) (*ReadPref, error) {
	rp := &ReadPref{
		mode:  mode,
		hedge: new(hedge),
	}

	if mode == PrimaryMode && len(opts) != 0 {
		return nil, errInvalidReadPreference
	}

	for _, opt := range opts {
		err := opt(rp)
		if err != nil {
			return nil, err
		}
	}

	// Create the hedge document if necessary.
	if rp.hedge != nil {
		hedgeRaw, err := rp.hedge.toDocument()
		if err != nil {
			return nil, fmt.Errorf("error creating hedge document: %v", err)
		}

		rp.hedgeRaw = hedgeRaw
	}

	return rp, nil
}

// hedge is an internal representation of the document used to configure hedged reads.
type hedge struct {
	enabled *bool
}

func (h *hedge) toDocument() (bson.Raw, error) {
	idx, doc := bsoncore.AppendDocumentStart(nil)
	if h.enabled != nil {
		doc = bsoncore.AppendBooleanElement(doc, "enabled", *h.enabled)
	}
	doc, err := bsoncore.AppendDocumentEnd(doc, idx)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// ReadPref determines which servers are considered suitable for read operations.
type ReadPref struct {
	maxStaleness    time.Duration
	maxStalenessSet bool
	mode            Mode
	tagSets         []tag.Set

	// Store a *hedge instance instead of storing hedge configurations as separate parameters so we can easily tell if
	// a hedge document needs to be created by checking rp.hedge != nil. The hedgeRaw field is the transformed bson.Raw
	// document. It's set on read preference construction and cached here rather than being re-constructed for each
	// operation.
	hedge    *hedge
	hedgeRaw bson.Raw
}

// MaxStaleness is the maximum amount of time to allow
// a server to be considered eligible for selection. The
// second return value indicates if this value has been set.
func (r *ReadPref) MaxStaleness() (time.Duration, bool) {
	return r.maxStaleness, r.maxStalenessSet
}

// Mode indicates the mode of the read preference.
func (r *ReadPref) Mode() Mode {
	return r.mode
}

// TagSets are multiple tag sets indicating
// which servers should be considered.
func (r *ReadPref) TagSets() []tag.Set {
	return r.tagSets
}

// Hedge returns the document used to configure server hedged reads. If no hedge options were specified when creating
// the read preference, nil is returned.
func (r *ReadPref) Hedge() bson.Raw {
	return r.hedgeRaw
}
