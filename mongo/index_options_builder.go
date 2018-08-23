// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import "github.com/mongodb/mongo-go-driver/bson"

// IndexOptionsBuilder constructs a BSON document for index options
type IndexOptionsBuilder struct {
	document *bson.Document
}

// NewIndexOptionsBuilder creates a new instance of IndexOptionsBuilder
func NewIndexOptionsBuilder() *IndexOptionsBuilder {
	var b IndexOptionsBuilder
	b.document = bson.NewDocument()
	return &b
}

// Background sets the background option
func (iob *IndexOptionsBuilder) Background(background bool) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Boolean("background", background))
	return iob
}

// ExpireAfterSeconds sets the expireAfterSeconds option
func (iob *IndexOptionsBuilder) ExpireAfterSeconds(expireAfterSeconds int32) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Int32("expireAfterSeconds", expireAfterSeconds))
	return iob
}

// Name sets the name option
func (iob *IndexOptionsBuilder) Name(name string) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.String("name", name))
	return iob
}

// Sparse sets the sparse option
func (iob *IndexOptionsBuilder) Sparse(sparse bool) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Boolean("sparse", sparse))
	return iob
}

// StorageEngine sets the storageEngine option
func (iob *IndexOptionsBuilder) StorageEngine(storageEngine *bson.Document) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.SubDocument("storageEngine", storageEngine))
	return iob
}

// Unique sets the unique option
func (iob *IndexOptionsBuilder) Unique(unique bool) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Boolean("unique", unique))
	return iob
}

// Version sets the version option
func (iob *IndexOptionsBuilder) Version(version int32) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Int32("v", version))
	return iob
}

// DefaultLanguage sets the defaultLanguage option
func (iob *IndexOptionsBuilder) DefaultLanguage(defaultLanguage string) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.String("default_language", defaultLanguage))
	return iob
}

// LanguageOverride sets the languageOverride option
func (iob *IndexOptionsBuilder) LanguageOverride(languageOverride string) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.String("language_override", languageOverride))
	return iob
}

// TextVersion sets the textVersion option
func (iob *IndexOptionsBuilder) TextVersion(textVersion int32) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Int32("textIndexVersion", textVersion))
	return iob
}

// Weights sets the weights option
func (iob *IndexOptionsBuilder) Weights(weights *bson.Document) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.SubDocument("weights", weights))
	return iob
}

// SphereVersion sets the sphereVersion option
func (iob *IndexOptionsBuilder) SphereVersion(sphereVersion int32) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Int32("2dsphereIndexVersion", sphereVersion))
	return iob
}

// Bits sets the bits option
func (iob *IndexOptionsBuilder) Bits(bits int32) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Int32("bits", bits))
	return iob
}

// Max sets the max option
func (iob *IndexOptionsBuilder) Max(max float64) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Double("max", max))
	return iob
}

// Min sets the min option
func (iob *IndexOptionsBuilder) Min(min float64) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Double("min", min))
	return iob
}

// BucketSize sets the bucketSize option
func (iob *IndexOptionsBuilder) BucketSize(bucketSize int32) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Int32("bucketSize", bucketSize))
	return iob
}

// PartialFilterExpression sets the partialFilterExpression option
func (iob *IndexOptionsBuilder) PartialFilterExpression(partialFilterExpression *bson.Document) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.SubDocument("partialFilterExpression", partialFilterExpression))
	return iob
}

// Collation sets the collation option
func (iob *IndexOptionsBuilder) Collation(collation *bson.Document) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.SubDocument("collation", collation))
	return iob
}

// Build returns the BSON document from the builder
func (iob *IndexOptionsBuilder) Build() *bson.Document {
	return iob.document
}
