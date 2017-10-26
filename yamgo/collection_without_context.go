// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package yamgo

import (
	"context"

	"github.com/10gen/mongo-go-driver/yamgo/options"
)

// InsertOne inserts a single document into the collection with a default context of
// context.Background.
//
// See InsertOneContext for details and options.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) InsertOne(document interface{},
	options ...options.InsertOption) (*InsertOneResult, error) {

	return coll.InsertOneContext(context.Background(), document, options...)
}

// DeleteOne deletes a single document from the collection with a default context of
// context.Background.
//
// See DeleteOneContext for details and options.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) DeleteOne(filter interface{},
	options ...options.DeleteOption) (*DeleteOneResult, error) {

	return coll.DeleteOneContext(context.Background(), filter, options...)
}

// UpdateOne updates a single document in the collection with a default context of
// context.Background.
//
// See UpdateOneContext for details and options.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) UpdateOne(filter interface{}, update interface{},
	options ...options.UpdateOption) (*UpdateOneResult, error) {

	return coll.UpdateOneContext(context.Background(), filter, update, options...)
}

// ReplaceOne replaces a single document in the collection with a default context of
// context.Background.
//
// See ReplaceOneContext for details and options.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) ReplaceOne(filter interface{}, replacement interface{},
	options ...options.UpdateOption) (*UpdateOneResult, error) {

	return coll.ReplaceOneContext(context.Background(), filter, replacement, options...)
}

// Aggregate runs an aggregation framework pipeline with a default context of context.Background.
//
// See AggregateContext for details and options.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) Aggregate(pipeline interface{},
	options ...options.AggregateOption) (Cursor, error) {

	return coll.AggregateContext(context.Background(), pipeline, options...)
}

// Count gets the number of documents matching the filter with a default context of
// context.Background.
//
// See CountContext for details and options.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) Count(filter interface{},
	options ...options.CountOption) (int64, error) {

	return coll.CountContext(context.Background(), filter, options...)
}

// Distinct finds the distinct values for a specified field across a single collection with a
// default context of context.Background.
//
// See DistinctContext for details and options.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) Distinct(fieldName string, filter interface{},
	options ...options.DistinctOption) ([]interface{}, error) {

	return coll.DistinctContext(context.Background(), fieldName, filter, options...)
}

// Find finds the documents matching the model with a default context of context.Background.
//
// See FindContext for details and options.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) Find(filter interface{},
	options ...options.FindOption) (Cursor, error) {

	return coll.FindContext(context.Background(), filter, options...)
}

// FindOne returns up to one document that matches the model with a default context of
// context.Background.
//
// See FindOneContext for details and options.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) FindOne(filter interface{}, result interface{},
	options ...options.FindOption) (bool, error) {

	return coll.FindOneContext(context.Background(), filter, result, options...)
}

// FindOneAndDelete find a single document and deletes it, returning the original in result.
// The document to return may be nil. It uses a default context of context.Background.
//
// See FindOneAndDeleteContext for details and options.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) FindOneAndDelete(filter interface{},
	result interface{}, opts ...options.FindOneAndDeleteOption) (bool, error) {

	return coll.FindOneAndDeleteContext(context.Background(), filter, result, opts...)
}

// FindOneAndReplace finds a single document and replaces it, returning either the original
// or the replaced document. The document to return may be nil. It uses a default context of
// context.Background.
//
// See FindOneAndReplaceContext for details and options.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) FindOneAndReplace(filter interface{},
	replacement interface{}, result interface{}, opts ...options.FindOneAndReplaceOption) (bool, error) {

	return coll.FindOneAndReplaceContext(context.Background(), filter, replacement, result, opts...)
}

// FindOneAndUpdate finds a single document an updates it, returning either the original
// or the updated document. The document to return may be nil. It uses a default context of
// context.Background.
//
// See FindOneAndUpdateContext for details and options.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) FindOneAndUpdate(filter interface{},
	update interface{}, result interface{}, opts ...options.FindOneAndUpdateOption) (bool, error) {

	return coll.FindOneAndUpdateContext(context.Background(), filter, update, result, opts...)
}
