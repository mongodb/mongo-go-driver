// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"github.com/mongodb/mongo-go-driver/bson"
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
	iob.document = append(iob.document, bson.Elem{"background", bson.Boolean(background)})
	return iob
}

// ExpireAfterSeconds sets the expireAfterSeconds option
func (iob *IndexOptionsBuilder) ExpireAfterSeconds(expireAfterSeconds int32) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"expireAfterSeconds", bson.Int32(expireAfterSeconds)})
	return iob
}

// Name sets the name option
func (iob *IndexOptionsBuilder) Name(name string) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"name", bson.String(name)})
	return iob
}

// Sparse sets the sparse option
func (iob *IndexOptionsBuilder) Sparse(sparse bool) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"sparse", bson.Boolean(sparse)})
	return iob
}

// StorageEngine sets the storageEngine option
func (iob *IndexOptionsBuilder) StorageEngine(storageEngine bsonx.Doc) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"storageEngine", bsonx.Document(storageEngine)})
	return iob
}

// Unique sets the unique option
func (iob *IndexOptionsBuilder) Unique(unique bool) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"unique", bson.Boolean(unique)})
	return iob
}

// Version sets the version option
func (iob *IndexOptionsBuilder) Version(version int32) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"v", bson.Int32(version)})
	return iob
}

// DefaultLanguage sets the defaultLanguage option
func (iob *IndexOptionsBuilder) DefaultLanguage(defaultLanguage string) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"default_language", bson.String(defaultLanguage)})
	return iob
}

// LanguageOverride sets the languageOverride option
func (iob *IndexOptionsBuilder) LanguageOverride(languageOverride string) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"language_override", bson.String(languageOverride)})
	return iob
}

// TextVersion sets the textVersion option
func (iob *IndexOptionsBuilder) TextVersion(textVersion int32) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"textIndexVersion", bson.Int32(textVersion)})
	return iob
}

// Weights sets the weights option
func (iob *IndexOptionsBuilder) Weights(weights bsonx.Doc) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"weights", bsonx.Document(weights)})
	return iob
}

// SphereVersion sets the sphereVersion option
func (iob *IndexOptionsBuilder) SphereVersion(sphereVersion int32) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"2dsphereIndexVersion", bson.Int32(sphereVersion)})
	return iob
}

// Bits sets the bits option
func (iob *IndexOptionsBuilder) Bits(bits int32) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"bits", bson.Int32(bits)})
	return iob
}

// Max sets the max option
func (iob *IndexOptionsBuilder) Max(max float64) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"max", bson.Double(max)})
	return iob
}

// Min sets the min option
func (iob *IndexOptionsBuilder) Min(min float64) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"min", bson.Double(min)})
	return iob
}

// BucketSize sets the bucketSize option
func (iob *IndexOptionsBuilder) BucketSize(bucketSize int32) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"bucketSize", bson.Int32(bucketSize)})
	return iob
}

// PartialFilterExpression sets the partialFilterExpression option
func (iob *IndexOptionsBuilder) PartialFilterExpression(partialFilterExpression bsonx.Doc) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"partialFilterExpression", bsonx.Document(partialFilterExpression)})
	return iob
}

// Collation sets the collation option
func (iob *IndexOptionsBuilder) Collation(collation bsonx.Doc) *IndexOptionsBuilder {
	iob.document = append(iob.document, bson.Elem{"collation", bsonx.Document(collation)})
	return iob
}

// Build returns the BSON document from the builder
func (iob *IndexOptionsBuilder) Build() bsonx.Doc {
	return iob.document
}
