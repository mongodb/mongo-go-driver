// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package option

import "github.com/mongodb/mongo-go-driver/bson"

// Collation allows users to specify language-specific rules for string comparison, such as
// rules for lettercase and accent marks.
type Collation struct {
	Locale          string `bson:",omitempty"`
	CaseLevel       bool   `bson:",omitempty"`
	CaseFirst       string `bson:",omitempty"`
	Strength        int    `bson:",omitempty"`
	NumericOrdering bool   `bson:",omitempty"`
	Alternate       string `bson:",omitempty"`
	MaxVariable     string `bson:",omitempty"`
	Backwards       bool   `bson:",omitempty"`
}

func (co *Collation) toDocument() *bson.Document {
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

// MarshalBSONDocument implements the bson.DocumentMarshaler interface.
func (co *Collation) MarshalBSONDocument() (*bson.Document, error) {
	return co.toDocument(), nil
}
