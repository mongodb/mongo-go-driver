// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"fmt"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"reflect"
)

// Collation allows users to specify language-specific rules for string comparison, such as
// rules for lettercase and accent marks.
type Collation struct {
	Locale          string `bson:",omitempty"` // The locale
	CaseLevel       bool   `bson:",omitempty"` // The case level
	CaseFirst       string `bson:",omitempty"` // The case ordering
	Strength        int    `bson:",omitempty"` // The number of comparision levels to use
	NumericOrdering bool   `bson:",omitempty"` // Whether to order numbers based on numerical order and not collation order
	Alternate       string `bson:",omitempty"` // Whether spaces and punctuation are considered base characters
	MaxVariable     string `bson:",omitempty"` // Which characters are affected by alternate: "shifted"
	Backwards       bool   `bson:",omitempty"` // Causes secondary differences to be considered in reverse order, as it is done in the French language
}

// ToDocument converts the Collation to a *bson.Document
func (co *Collation) ToDocument() *bson.Document {
	doc := bson.NewDocument()
	if co.Locale != "" {
		doc.Append(bson.EC.String("locale", co.Locale))
	}
	if co.CaseLevel {
		doc.Append(bson.EC.Boolean("caseLevel", true))
	}
	if co.CaseFirst != "" {
		doc.Append(bson.EC.String("caseFirst", co.CaseFirst))
	}
	if co.Strength != 0 {
		doc.Append(bson.EC.Int32("strength", int32(co.Strength)))
	}
	if co.NumericOrdering {
		doc.Append(bson.EC.Boolean("numericOrdering", true))
	}
	if co.Alternate != "" {
		doc.Append(bson.EC.String("alternate", co.Alternate))
	}
	if co.MaxVariable != "" {
		doc.Append(bson.EC.String("maxVariable", co.MaxVariable))
	}
	if co.Backwards {
		doc.Append(bson.EC.Boolean("backwards", true))
	}
	return doc
}


// CursorType specifies whether a cursor should close when the last data is retrieved. See
// NonTailable, Tailable, and TailableAwait.
type CursorType int8

const (
	// NonTailable specifies that a cursor should close after retrieving the last data.
	NonTailable CursorType = iota
	// Tailable specifies that a cursor should not close when the last data is retrieved.
	Tailable
	// TailableAwait specifies that a cursor should not close when the last data is retrieved and
	// that it should block for a certain amount of time for new data before returning no data.
	TailableAwait
)

// ReturnDocument specifies whether a findAndUpdate operation should return the document as it was
// before the update or as it is after the update.
type ReturnDocument int8

const (
	// Before specifies that findAndUpdate should return the document as it was before the update.
	Before ReturnDocument = iota
	// After specifies that findAndUpdate should return the document as it is after the update.
	After
)

// FullDocument specifies whether a change stream should include a copy of the entire document that was changed from
// some time after the change occurred.
type FullDocument string

const (
	// Default does not include a document copy
	Default FullDocument = "default"
	// UpdateLookup includes a delta describing the changes to the document and a copy of the entire document that
	// was changed
	UpdateLookup FullDocument = "updateLookup"
)

// ArrayFilters is for internal use.
type ArrayFilters struct {
	Registry *bsoncodec.Registry
	Filters  []interface{}
}

func (af *ArrayFilters) ToArray() (*bson.Array, error) {
	docs := make([]*bson.Document, 0, len(af.Filters))
	for _, f := range af.Filters {
		d, err := transformDocument(af.Registry, f)
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}

	arr := bson.NewArray()
	for _, doc := range docs {
		arr.Append(bson.VC.Document(doc))
	}

	return arr, nil
}

// MarshalError is returned when attempting to transform a value into a document
// results in an error.
type MarshalError struct {
	Value interface{}
	Err   error
}

// Error implements the error interface.
func (me MarshalError) Error() string {
	return fmt.Sprintf("cannot transform type %s to a *bson.Document", reflect.TypeOf(me.Value))
}

var defaultRegistry = bsoncodec.NewRegistryBuilder().Build()

func transformDocument(registry *bsoncodec.Registry, val interface{}) (*bson.Document, error) {
	if val == nil {
		return bson.NewDocument(), nil
	}
	reg := defaultRegistry
	if registry != nil {
		reg = registry
	}

	if bs, ok := val.([]byte); ok {
		// Slight optimization so we'll just use MarshalBSON and not go through the codec machinery.
		val = bson.Reader(bs)
	}

	// TODO(skriptble): Use a pool of these instead.
	buf := make([]byte, 0, 256)
	b, err := bsoncodec.MarshalAppendWithRegistry(reg, buf, val)
	if err != nil {
		return nil, MarshalError{Value: val, Err: err}
	}
	return bson.ReadDocument(b)
}
