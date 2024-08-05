// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

var defaultRegistry = bson.NewRegistry()

// Collation allows users to specify language-specific rules for string comparison, such as
// rules for lettercase and accent marks.
type Collation struct {
	Locale          string `bson:",omitempty"` // The locale
	CaseLevel       bool   `bson:",omitempty"` // The case level
	CaseFirst       string `bson:",omitempty"` // The case ordering
	Strength        int    `bson:",omitempty"` // The number of comparison levels to use
	NumericOrdering bool   `bson:",omitempty"` // Whether to order numbers based on numerical order and not collation order
	Alternate       string `bson:",omitempty"` // Whether spaces and punctuation are considered base characters
	MaxVariable     string `bson:",omitempty"` // Which characters are affected by alternate: "shifted"
	Normalization   bool   `bson:",omitempty"` // Causes text to be normalized into Unicode NFD
	Backwards       bool   `bson:",omitempty"` // Causes secondary differences to be considered in reverse order, as it is done in the French language
}

// CursorType specifies whether a cursor should close when the last data is retrieved. See
// NonTailable, Tailable, and TailableAwait.
type CursorType int8

const (
	// NonTailable specifies that a cursor should close after retrieving the last data.
	NonTailable CursorType = iota
	// Tailable specifies that a cursor should not close when the last data is retrieved and can be resumed later.
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

// FullDocument specifies how a change stream should return the modified document.
type FullDocument string

const (
	// Default does not include a document copy.
	Default FullDocument = "default"
	// Off is the same as sending no value for fullDocumentBeforeChange.
	Off FullDocument = "off"
	// Required is the same as WhenAvailable but raises a server-side error if the post-image is not available.
	Required FullDocument = "required"
	// UpdateLookup includes a delta describing the changes to the document and a copy of the entire document that
	// was changed.
	UpdateLookup FullDocument = "updateLookup"
	// WhenAvailable includes a post-image of the modified document for replace and update change events
	// if the post-image for this event is available.
	WhenAvailable FullDocument = "whenAvailable"
)
