// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"github.com/mongodb/mongo-go-driver/x/bsonx"
)

// IndexOptionsBuilder constructs a BSON document for index options
type IndexOptionsBuilder struct {
	document bsonx.Doc
}

// NewIndexOptionsBuilder creates a new instance of IndexOptionsBuilder
func NewIndexOptionsBuilder() *IndexOptionsBuilder {
	var b IndexOptionsBuilder
	b.document = bsonx.Doc{}
	return &b
}

// Background sets the background option
func (iob *IndexOptionsBuilder) Background(background bool) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"background", bsonx.Boolean(background)})
	return iob
}

// ExpireAfterSeconds sets the expireAfterSeconds option
func (iob *IndexOptionsBuilder) ExpireAfterSeconds(expireAfterSeconds int32) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"expireAfterSeconds", bsonx.Int32(expireAfterSeconds)})
	return iob
}

// Name sets the name option
func (iob *IndexOptionsBuilder) Name(name string) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"name", bsonx.String(name)})
	return iob
}

// Sparse sets the sparse option
func (iob *IndexOptionsBuilder) Sparse(sparse bool) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"sparse", bsonx.Boolean(sparse)})
	return iob
}

// StorageEngine sets the storageEngine option
func (iob *IndexOptionsBuilder) StorageEngine(storageEngine bsonx.Doc) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"storageEngine", bsonx.Document(storageEngine)})
	return iob
}

// Unique sets the unique option
func (iob *IndexOptionsBuilder) Unique(unique bool) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"unique", bsonx.Boolean(unique)})
	return iob
}

// Version sets the version option
func (iob *IndexOptionsBuilder) Version(version int32) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"v", bsonx.Int32(version)})
	return iob
}

// DefaultLanguage sets the defaultLanguage option
func (iob *IndexOptionsBuilder) DefaultLanguage(defaultLanguage string) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"default_language", bsonx.String(defaultLanguage)})
	return iob
}

// LanguageOverride sets the languageOverride option
func (iob *IndexOptionsBuilder) LanguageOverride(languageOverride string) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"language_override", bsonx.String(languageOverride)})
	return iob
}

// TextVersion sets the textVersion option
func (iob *IndexOptionsBuilder) TextVersion(textVersion int32) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"textIndexVersion", bsonx.Int32(textVersion)})
	return iob
}

// Weights sets the weights option
func (iob *IndexOptionsBuilder) Weights(weights bsonx.Doc) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"weights", bsonx.Document(weights)})
	return iob
}

// SphereVersion sets the sphereVersion option
func (iob *IndexOptionsBuilder) SphereVersion(sphereVersion int32) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"2dsphereIndexVersion", bsonx.Int32(sphereVersion)})
	return iob
}

// Bits sets the bits option
func (iob *IndexOptionsBuilder) Bits(bits int32) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"bits", bsonx.Int32(bits)})
	return iob
}

// Max sets the max option
func (iob *IndexOptionsBuilder) Max(max float64) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"max", bsonx.Double(max)})
	return iob
}

// Min sets the min option
func (iob *IndexOptionsBuilder) Min(min float64) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"min", bsonx.Double(min)})
	return iob
}

// BucketSize sets the bucketSize option
func (iob *IndexOptionsBuilder) BucketSize(bucketSize int32) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"bucketSize", bsonx.Int32(bucketSize)})
	return iob
}

// PartialFilterExpression sets the partialFilterExpression option
func (iob *IndexOptionsBuilder) PartialFilterExpression(partialFilterExpression bsonx.Doc) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"partialFilterExpression", bsonx.Document(partialFilterExpression)})
	return iob
}

// Collation sets the collation option
func (iob *IndexOptionsBuilder) Collation(collation bsonx.Doc) *IndexOptionsBuilder {
	iob.document = append(iob.document, bsonx.Elem{"collation", bsonx.Document(collation)})
	return iob
}

// Build returns the BSON document from the builder
func (iob *IndexOptionsBuilder) Build() bsonx.Doc {
	return iob.document
}
