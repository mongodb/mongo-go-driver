// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"github.com/mongodb/mongo-go-driver/bson"
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

// Hint allows users to specify the index to use
type Hint interface {
	hint()
}

type HintString struct {
	S string
}

func NewHintString(s string) *HintString {
	return &HintString{s}
}

func (hs *HintString) hint() {}

type HintDocument struct {
	D *bson.Document
}

func NewHintDocument(d *bson.Document) *HintDocument {
	return &HintDocument{d}
}

func (hd *HintDocument) hint() {}
