// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import "github.com/skriptble/wilson/bson"

// CollationOptions allows users to specify language-specific rules for string comparison, such as
// rules for lettercase and accent marks.
type CollationOptions struct {
	Locale          string ",omitempty"
	CaseLevel       bool   ",omitempty"
	CaseFirst       string ",omitempty"
	Strength        int    ",omitempty"
	NumericOrdering bool   ",omitempty"
	Alternate       string ",omitempty"
	MaxVariable     string ",omitempty"
	Backwards       bool   ",omitempty"
}

func (co *CollationOptions) toDocument() *bson.Document {
	doc := bson.NewDocument()
	if co.Locale != "" {
		doc.Append(bson.C.String("locale", co.Locale))
	}
	if co.CaseLevel {
		doc.Append(bson.C.Boolean("caseLevel", true))
	}
	if co.CaseFirst != "" {
		doc.Append(bson.C.String("caseFirst", co.CaseFirst))
	}
	if co.Strength != 0 {
		doc.Append(bson.C.Int32("strength", int32(co.Strength)))
	}
	if co.NumericOrdering {
		doc.Append(bson.C.Boolean("numericOrdering", true))
	}
	if co.Alternate != "" {
		doc.Append(bson.C.String("alternate", co.Alternate))
	}
	if co.MaxVariable != "" {
		doc.Append(bson.C.String("maxVariable", co.MaxVariable))
	}
	if co.Backwards {
		doc.Append(bson.C.Boolean("backwards", true))
	}
	return doc
}

func (co *CollationOptions) MarshalBSONDocument() (*bson.Document, error) {
	return co.toDocument(), nil
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
