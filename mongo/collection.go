// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/mongo/options"
	"github.com/10gen/mongo-go-driver/mongo/private/cluster"
	"github.com/10gen/mongo-go-driver/mongo/private/ops"
	"github.com/10gen/mongo-go-driver/mongo/readconcern"
	"github.com/10gen/mongo-go-driver/mongo/readpref"
	"github.com/10gen/mongo-go-driver/mongo/writeconcern"
)

// Collection performs operations on a given collection.
type Collection struct {
	client         *Client
	db             *Database
	name           string
	readConcern    *readconcern.ReadConcern
	writeConcern   *writeconcern.WriteConcern
	readPreference *readpref.ReadPref
	readSelector   cluster.ServerSelector
	writeSelector  cluster.ServerSelector
}

func newCollection(db *Database, name string) *Collection {
	coll := &Collection{
		client:         db.client,
		db:             db,
		name:           name,
		readPreference: db.readPreference,
		readConcern:    db.readConcern,
		writeConcern:   db.writeConcern,
	}

	latencySelector := cluster.LatencySelector(db.client.localThreshold)

	coll.readSelector = cluster.CompositeSelector([]cluster.ServerSelector{
		readpref.Selector(coll.readPreference),
		latencySelector,
	})

	coll.writeSelector = readpref.Selector(readpref.Primary())

	return coll
}

// namespace returns the namespace of the collection.
func (coll *Collection) namespace() ops.Namespace {
	return ops.NewNamespace(coll.db.name, coll.name)
}

func (coll *Collection) getWriteableServer(ctx context.Context) (*ops.SelectedServer, error) {
	return coll.client.selectServer(ctx, coll.writeSelector, readpref.Primary())
}

func (coll *Collection) getReadableServer(ctx context.Context) (*ops.SelectedServer, error) {
	return coll.client.selectServer(ctx, coll.readSelector, coll.readPreference)
}

// InsertOne inserts a single document into the collection. A user can supply
// a custom context to this method, or nil to default to context.Background().
//
// This method uses TransformDocument to turn the document parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// document.
//
// TODO(skriptble): Determine if we should unwrap the value for the
// InsertOneResult or just return the bson.Element or a bson.Value.
func (coll *Collection) InsertOne(ctx context.Context, document interface{},
	options ...options.InsertOptioner) (*InsertOneResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	doc, err := TransformDocument(document)
	if err != nil {
		return nil, err
	}

	insertedID, err := ensureID(doc)
	if err != nil {
		return nil, err
	}

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	insert := func() (bson.Reader, error) {
		return ops.Insert(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			[]*bson.Document{doc},
			options...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		go func() { _, _ = insert() }()
		return nil, nil
	}

	_, err = insert()

	if err != nil {
		return nil, err
	}

	return &InsertOneResult{InsertedID: insertedID}, nil
}

// InsertMany inserts the provided documents. A user can supply a custom context to this
// method.
//
// Currently, batching is not implemented for this operation. Because of this, extremely large
// sets of documents will not fit into a single BSON document to be sent to the server, so the
// operation will fail.
//
// This method uses TransformDocument to turn the documents parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// documents.
func (coll *Collection) InsertMany(ctx context.Context, documents []interface{},
	options ...options.InsertOptioner) (*InsertManyResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	result := make([]interface{}, len(documents))
	docs := make([]*bson.Document, len(documents))

	for i, doc := range documents {
		bdoc, err := TransformDocument(doc)
		if err != nil {
			return nil, err
		}
		insertedID, err := ensureID(bdoc)
		if err != nil {
			return nil, err
		}

		docs[i] = bdoc
		result[i] = insertedID
	}

	// TODO GODRIVER-27: write concern

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	insert := func() (bson.Reader, error) {
		return ops.Insert(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			docs,
			options...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		go func() { _, _ = insert() }()
		return nil, nil
	}

	_, err = insert()

	if err != nil {
		return nil, err
	}

	return &InsertManyResult{InsertedIDs: result}, nil
}

// DeleteOne deletes a single document from the collection. A user can supply
// a custom context to this method, or nil to default to context.Background().
//
// This method uses TransformDocument to turn the filter parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// filter.
func (coll *Collection) DeleteOne(ctx context.Context, filter interface{},
	options ...options.DeleteOptioner) (*DeleteResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := TransformDocument(filter)
	if err != nil {
		return nil, err
	}
	deleteDocs := []*bson.Document{
		bson.NewDocument(
			bson.C.SubDocument("q", f),
			bson.C.Int32("limit", 1)),
	}

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	var result DeleteResult
	doDelete := func() (bson.Reader, error) {
		return ops.Delete(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			deleteDocs,
			options...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		go func() { _, _ = doDelete() }()
		return nil, nil
	}

	rdr, err := doDelete()
	if err != nil {
		return nil, err
	}

	err = bson.NewDecoder(bytes.NewReader(rdr)).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteMany deletes multiple documents from the collection. A user can
// supply a custom context to this method, or nil to default to
// context.Background().
//
// This method uses TransformDocument to turn the filter parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// filter.
func (coll *Collection) DeleteMany(ctx context.Context, filter interface{},
	options ...options.DeleteOptioner) (*DeleteResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := TransformDocument(filter)
	if err != nil {
		return nil, err
	}
	deleteDocs := []*bson.Document{bson.NewDocument(bson.C.SubDocument("q", f), bson.C.Int32("limit", 0))}

	// TODO GODRIVER-27: write concern

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	var result DeleteResult
	doDelete := func() (bson.Reader, error) {
		return ops.Delete(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			deleteDocs,
			options...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		go func() { _, _ = doDelete() }()
		return nil, nil
	}

	rdr, err := doDelete()
	if err != nil {
		return nil, err
	}

	err = bson.NewDecoder(bytes.NewReader(rdr)).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (coll *Collection) updateOrReplaceOne(ctx context.Context, filter,
	update *bson.Document, options ...options.UpdateOptioner) (*UpdateResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	updateDocs := []*bson.Document{
		bson.NewDocument(
			bson.C.SubDocument("q", filter),
			bson.C.SubDocument("u", update),
			bson.C.Boolean("multi", false),
		),
	}

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	var result UpdateResult
	doUpdate := func() (bson.Reader, error) {
		return ops.Update(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			updateDocs,
			options...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		go func() { _, _ = doUpdate() }()
		return nil, nil
	}

	rdr, err := doUpdate()
	if err != nil {
		return nil, err
	}

	err = bson.NewDecoder(bytes.NewReader(rdr)).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateOne updates a single document in the collection. A user can supply a
// custom context to this method, or nil to default to context.Background().
//
// This method uses TransformDocument to turn the filter and update parameter
// into a *bson.Document. See TransformDocument for the list of valid types for
// filter and update.
func (coll *Collection) UpdateOne(ctx context.Context, filter interface{}, update interface{},
	options ...options.UpdateOptioner) (*UpdateResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := TransformDocument(filter)
	if err != nil {
		return nil, err
	}

	u, err := TransformDocument(update)
	if err != nil {
		return nil, err
	}

	if err := ensureDollarKey(u); err != nil {
		return nil, err
	}

	return coll.updateOrReplaceOne(ctx, f, u, options...)
}

// UpdateMany updates multiple documents in the collection. A user can supply
// a custom context to this method, or nil to default to context.Background().
//
// This method uses TransformDocument to turn the filter and update parameter
// into a *bson.Document. See TransformDocument for the list of valid types for
// filter and update.
func (coll *Collection) UpdateMany(ctx context.Context, filter interface{}, update interface{},
	options ...options.UpdateOptioner) (*UpdateResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := TransformDocument(filter)
	if err != nil {
		return nil, err
	}

	u, err := TransformDocument(update)
	if err != nil {
		return nil, err
	}

	if err = ensureDollarKey(u); err != nil {
		return nil, err
	}

	updateDocs := []*bson.Document{
		bson.NewDocument(
			bson.C.SubDocument("q", f),
			bson.C.SubDocument("u", u),
			bson.C.Boolean("multi", true),
		),
	}

	// TODO GODRIVER-27: write concern

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	var result UpdateResult
	doUpdate := func() (bson.Reader, error) {
		return ops.Update(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			updateDocs,
			options...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		go func() { _, _ = doUpdate() }()
		return nil, nil
	}

	rdr, err := doUpdate()
	if err != nil {
		return nil, err
	}

	err = bson.NewDecoder(bytes.NewReader(rdr)).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// ReplaceOne replaces a single document in the collection. A user can supply
// a custom context to this method, or nil to default to context.Background().
//
// This method uses TransformDocument to turn the filter and replacement
// parameter into a *bson.Document. See TransformDocument for the list of
// valid types for filter and replacement.
func (coll *Collection) ReplaceOne(ctx context.Context, filter interface{},
	replacement interface{}, options ...options.UpdateOptioner) (*UpdateResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := TransformDocument(filter)
	if err != nil {
		return nil, err
	}

	r, err := TransformDocument(replacement)
	if err != nil {
		return nil, err
	}

	elem, err := r.ElementAt(0)
	if err != nil && err != bson.ErrOutOfBounds {
		return nil, err
	}
	if strings.HasPrefix(elem.Key(), "$") {
		return nil, errors.New("replacement document cannot contains keys beginning with '$")
	}
	return coll.updateOrReplaceOne(ctx, f, r, options...)
}

// Aggregate runs an aggregation framework pipeline. A user can supply a custom context to
// this method.
//
// See https://docs.mongodb.com/manual/aggregation/.
//
// This method uses TransformDocument to turn the pipeline parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// pipeline.
func (coll *Collection) Aggregate(ctx context.Context, pipeline interface{},
	options ...options.AggregateOptioner) (Cursor, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	var pipelineArr *bson.Array
	switch t := pipeline.(type) {
	case *bson.Array:
		pipelineArr = t
	default:
		p, err := TransformDocument(pipeline)
		if err != nil {
			return nil, err
		}

		pipelineArr = bson.ArrayFromDocument(p)
	}

	var dollarOut bool
	for i := 0; i < pipelineArr.Len(); i++ {
		val, err := pipelineArr.Lookup(uint(i))
		if err != nil {
			return nil, err
		}
		if val.Type() != bson.TypeEmbeddedDocument {
			return nil, errors.New("pipeline contains non-document element")
		}
		doc := val.MutableDocument()
		if doc.Len() != 1 {
			return nil, fmt.Errorf("pipeline document of incorrect length %d", doc.Len())
		}

		elem, err := doc.ElementAt(0)
		if err != nil {
			return nil, err
		}
		if elem.Key() == "$out" {
			dollarOut = true
		}
	}
	// TODO GODRIVER-95: Check for $out and use readable server/read preference if not found
	var cursor Cursor
	var err error
	if dollarOut {
		var s *ops.SelectedServer
		s, err = coll.getWriteableServer(ctx)
		if err != nil {
			return nil, err
		}
		cursor, err = ops.Aggregate(ctx, s, coll.namespace(), nil, coll.writeConcern, pipelineArr, options...)
	} else {
		var s *ops.SelectedServer
		s, err = coll.getReadableServer(ctx)
		if err != nil {
			return nil, err
		}
		cursor, err = ops.Aggregate(ctx, s, coll.namespace(), coll.readConcern, nil, pipelineArr, options...)
	}

	if err != nil {
		return nil, err
	}

	return cursor, nil
}

// Count gets the number of documents matching the filter. A user can supply a
// custom context to this method, or nil to default to context.Background().
//
// This method uses TransformDocument to turn the filter parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// filter.
func (coll *Collection) Count(ctx context.Context, filter interface{},
	options ...options.CountOptioner) (int64, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := TransformDocument(filter)
	if err != nil {
		return 0, err
	}

	s, err := coll.getReadableServer(ctx)
	if err != nil {
		return 0, err
	}

	count, err := ops.Count(ctx, s, coll.namespace(), coll.readConcern, f, options...)
	if err != nil {
		return 0, err
	}

	return int64(count), nil
}

// Distinct finds the distinct values for a specified field across a single
// collection. A user can supply a custom context to this method, or nil to
// default to context.Background().
//
// This method uses TransformDocument to turn the filter parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// filter.
func (coll *Collection) Distinct(ctx context.Context, fieldName string, filter interface{},
	options ...options.DistinctOptioner) ([]interface{}, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	var f *bson.Document
	var err error
	if filter != nil {
		f, err = TransformDocument(filter)
		if err != nil {
			return nil, err
		}
	}

	s, err := coll.getReadableServer(ctx)

	if err != nil {
		return nil, err
	}

	values, err := ops.Distinct(
		ctx,
		s,
		coll.namespace(),
		coll.readConcern,
		fieldName,
		f,
		options...,
	)

	if err != nil {
		return nil, err
	}

	return values, nil
}

// Find finds the documents matching a model. A user can supply a custom context to this
// method.
//
// This method uses TransformDocument to turn the filter parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// filter.
func (coll *Collection) Find(ctx context.Context, filter interface{},
	options ...options.FindOptioner) (Cursor, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	var f *bson.Document
	var err error
	if filter != nil {
		f, err = TransformDocument(filter)
		if err != nil {
			return nil, err
		}
	}

	s, err := coll.getReadableServer(ctx)
	if err != nil {
		return nil, err
	}

	cursor, err := ops.Find(ctx, s, coll.namespace(), coll.readConcern, f, options...)
	if err != nil {
		return nil, err
	}

	return cursor, nil
}

// FindOne returns up to one document that matches the model. A user can
// supply a custom context to this method, or nil to default to
// context.Background().
//
// This method uses TransformDocument to turn the filter parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// filter.
func (coll *Collection) FindOne(ctx context.Context, filter interface{},
	options ...options.FindOptioner) *DocumentResult {

	if ctx == nil {
		ctx = context.Background()
	}

	options = append(options, Limit(1))

	var f *bson.Document
	var err error
	if filter != nil {
		f, err = TransformDocument(filter)
		if err != nil {
			return &DocumentResult{err: err}
		}
	}

	cursor, err := coll.Find(ctx, f, options...)
	if err != nil {
		return &DocumentResult{err: err}
	}

	return &DocumentResult{cur: cursor}
}

// FindOneAndDelete find a single document and deletes it, returning the
// original in result.  The document to return may be nil.
//
// A user can supply a custom context to this method, or nil to default to
// context.Background().
//
// This method uses TransformDocument to turn the filter parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// filter.
func (coll *Collection) FindOneAndDelete(ctx context.Context, filter interface{},
	opts ...options.FindOneAndDeleteOptioner) *DocumentResult {

	if ctx == nil {
		ctx = context.Background()
	}

	var f *bson.Document
	var err error
	if filter != nil {
		f, err = TransformDocument(filter)
		if err != nil {
			return &DocumentResult{err: err}
		}
	}

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return &DocumentResult{err: err}
	}

	findOneAndDelete := func() (Cursor, error) {
		return ops.FindOneAndDelete(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			f,
			opts...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		// TODO(skriptble): This is probably wrong. We should be returning
		// ErrUnacknowledged but we don't have that yet  ¯\_(ツ)_/¯
		go func() { _, _ = findOneAndDelete() }()
		return nil
	}

	cur, err := findOneAndDelete()
	return &DocumentResult{cur: cur, err: err}
}

// FindOneAndReplace finds a single document and replaces it, returning either
// the original or the replaced document. The document to return may be nil.
//
// A user can supply a custom context to this method, or nil to default to
// context.Background().
//
// This method uses TransformDocument to turn the filter and replacement
// parameter into a *bson.Document. See TransformDocument for the list of
// valid types for filter and replacement.
func (coll *Collection) FindOneAndReplace(ctx context.Context, filter interface{},
	replacement interface{}, opts ...options.FindOneAndReplaceOptioner) *DocumentResult {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := TransformDocument(filter)
	if err != nil {
		return &DocumentResult{err: err}
	}

	r, err := TransformDocument(replacement)
	if err != nil {
		return &DocumentResult{err: err}
	}

	elem, err := r.ElementAt(0)
	if err != nil && err != bson.ErrOutOfBounds {
		return &DocumentResult{err: err}
	}

	if strings.HasPrefix(elem.Key(), "$") {
		return &DocumentResult{err: errors.New("replacement document cannot contains keys beginning with '$")}
	}

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return &DocumentResult{err: err}
	}

	findOneAndReplace := func() (Cursor, error) {
		return ops.FindOneAndReplace(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			f,
			r,
			opts...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		// TODO(skriptble): This is probably wrong. We should be returning
		// ErrUnacknowledged but we don't have that yet  ¯\_(ツ)_/¯
		go func() { _, _ = findOneAndReplace() }()
		return nil
	}

	cur, err := findOneAndReplace()
	return &DocumentResult{cur: cur, err: err}
}

// FindOneAndUpdate finds a single document and updates it, returning either
// the original or the updated. The document to return may be nil.
//
// A user can supply a custom context to this method, or nil to default to
// context.Background().
//
// This method uses TransformDocument to turn the filter and update parameter
// into a *bson.Document. See TransformDocument for the list of valid types for
// filter and update.
func (coll *Collection) FindOneAndUpdate(ctx context.Context, filter interface{},
	update interface{}, opts ...options.FindOneAndUpdateOptioner) *DocumentResult {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := TransformDocument(filter)
	if err != nil {
		return &DocumentResult{err: err}
	}

	u, err := TransformDocument(update)
	if err != nil {
		return &DocumentResult{err: err}
	}

	elem, err := u.ElementAt(0)
	if err != nil && err != bson.ErrOutOfBounds {
		return &DocumentResult{err: err}
	}

	if !strings.HasPrefix(elem.Key(), "$") {
		return &DocumentResult{err: errors.New("update document must contain key beginning with '$")}
	}

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return &DocumentResult{err: err}
	}

	findOneAndUpdate := func() (Cursor, error) {
		return ops.FindOneAndUpdate(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			f,
			u,
			opts...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		// TODO(skriptble): This is probably wrong. We should be returning
		// ErrUnacknowledged but we don't have that yet  ¯\_(ツ)_/¯
		go func() { _, _ = findOneAndUpdate() }()
		return nil
	}

	cur, err := findOneAndUpdate()
	return &DocumentResult{cur: cur, err: err}
}
