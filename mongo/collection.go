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

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/cluster"
	"github.com/mongodb/mongo-go-driver/mongo/private/ops"
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/readconcern"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
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
	opts ...options.InsertOneOptioner) (*InsertOneResult, error) {

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

	if coll.writeConcern != nil {
		wc, err := Opt.WriteConcern(coll.writeConcern)
		if err != nil {
			return nil, err
		}

		opts = append(opts, wc)
	}

	newOptions := make([]options.InsertOptioner, 0, len(opts))
	for _, opt := range opts {
		newOptions = append(newOptions, opt)
	}

	insert := func() (bson.Reader, error) {
		return ops.Insert(
			ctx,
			s,
			coll.namespace(),
			[]*bson.Document{doc},
			newOptions...,
		)
	}

	acknowledged := true

	for _, opt := range opts {
		if wc, ok := opt.(options.OptWriteConcern); ok {
			acknowledged = wc.Acknowledged
			break
		}
	}

	if !acknowledged {
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
	opts ...options.InsertManyOptioner) (*InsertManyResult, error) {

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

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	if coll.writeConcern != nil {
		wc, err := Opt.WriteConcern(coll.writeConcern)
		if err != nil {
			return nil, err
		}

		opts = append(opts, wc)
	}

	newOptions := make([]options.InsertOptioner, 0, len(opts))
	for _, opt := range opts {
		newOptions = append(newOptions, opt)
	}

	insert := func() (bson.Reader, error) {
		return ops.Insert(
			ctx,
			s,
			coll.namespace(),
			docs,
			newOptions...,
		)
	}

	acknowledged := true

	for _, opt := range opts {
		if wc, ok := opt.(options.OptWriteConcern); ok {
			acknowledged = wc.Acknowledged
			break
		}
	}

	if !acknowledged {
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
	opts ...options.DeleteOptioner) (*DeleteResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := TransformDocument(filter)
	if err != nil {
		return nil, err
	}
	deleteDocs := []*bson.Document{
		bson.NewDocument(
			bson.EC.SubDocument("q", f),
			bson.EC.Int32("limit", 1)),
	}

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	if coll.writeConcern != nil {
		wc, err := Opt.WriteConcern(coll.writeConcern)
		if err != nil {
			return nil, err
		}

		opts = append(opts, wc)
	}

	var result DeleteResult
	doDelete := func() (bson.Reader, error) {
		return ops.Delete(
			ctx,
			s,
			coll.namespace(),
			deleteDocs,
			opts...,
		)
	}

	acknowledged := true

	for _, opt := range opts {
		if wc, ok := opt.(options.OptWriteConcern); ok {
			acknowledged = wc.Acknowledged
			break
		}
	}

	if !acknowledged {
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
	opts ...options.DeleteOptioner) (*DeleteResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := TransformDocument(filter)
	if err != nil {
		return nil, err
	}
	deleteDocs := []*bson.Document{bson.NewDocument(bson.EC.SubDocument("q", f), bson.EC.Int32("limit", 0))}

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	if coll.writeConcern != nil {
		wc, err := Opt.WriteConcern(coll.writeConcern)
		if err != nil {
			return nil, err
		}

		opts = append(opts, wc)
	}

	var result DeleteResult
	doDelete := func() (bson.Reader, error) {
		return ops.Delete(
			ctx,
			s,
			coll.namespace(),
			deleteDocs,
			opts...,
		)
	}

	acknowledged := true

	for _, opt := range opts {
		if wc, ok := opt.(options.OptWriteConcern); ok {
			acknowledged = wc.Acknowledged
			break
		}
	}

	if !acknowledged {
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
	update *bson.Document, opts ...options.UpdateOptioner) (*UpdateResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	updateDocs := []*bson.Document{
		bson.NewDocument(
			bson.EC.SubDocument("q", filter),
			bson.EC.SubDocument("u", update),
			bson.EC.Boolean("multi", false),
		),
	}

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	if coll.writeConcern != nil {
		wc, err := Opt.WriteConcern(coll.writeConcern)
		if err != nil {
			return nil, err
		}

		opts = append(opts, wc)
	}

	var result UpdateResult
	doUpdate := func() (bson.Reader, error) {
		return ops.Update(
			ctx,
			s,
			coll.namespace(),
			updateDocs,
			opts...,
		)
	}

	acknowledged := true

	for _, opt := range opts {
		if wc, ok := opt.(options.OptWriteConcern); ok {
			acknowledged = wc.Acknowledged
			break
		}
	}

	if !acknowledged {
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

	if result.UpsertedID != nil {
		result.MatchedCount--
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
	opts ...options.UpdateOptioner) (*UpdateResult, error) {

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
			bson.EC.SubDocument("q", f),
			bson.EC.SubDocument("u", u),
			bson.EC.Boolean("multi", true),
		),
	}

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	if coll.writeConcern != nil {
		wc, err := Opt.WriteConcern(coll.writeConcern)
		if err != nil {
			return nil, err
		}

		opts = append(opts, wc)
	}

	var result UpdateResult
	doUpdate := func() (bson.Reader, error) {
		return ops.Update(
			ctx,
			s,
			coll.namespace(),
			updateDocs,
			opts...,
		)
	}

	acknowledged := true

	for _, opt := range opts {
		if wc, ok := opt.(options.OptWriteConcern); ok {
			acknowledged = wc.Acknowledged
			break
		}
	}

	if !acknowledged {
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

	if result.UpsertedID != nil {
		result.MatchedCount--
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
	replacement interface{}, opts ...options.ReplaceOptioner) (*UpdateResult, error) {

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

	if elem, ok := r.ElementAtOK(0); ok && strings.HasPrefix(elem.Key(), "$") {
		return nil, errors.New("replacement document cannot contains keys beginning with '$")
	}

	updateOptions := make([]options.UpdateOptioner, 0, len(opts))
	for _, opt := range opts {
		updateOptions = append(updateOptions, opt)
	}

	return coll.updateOrReplaceOne(ctx, f, r, updateOptions...)
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
	opts ...options.AggregateOptioner) (Cursor, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	pipelineArr, err := transformAggregatePipeline(pipeline)
	if err != nil {
		return nil, err
	}

	var dollarOut bool
	if pipelineArr.Len() > 0 {
		val, err := pipelineArr.Lookup(uint(pipelineArr.Len() - 1))
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

		if elem, ok := doc.ElementAtOK(0); ok && elem.Key() == "$out" {
			dollarOut = true
		}
	}

	var cursor Cursor
	if dollarOut {
		var s *ops.SelectedServer
		s, err = coll.getWriteableServer(ctx)
		if err != nil {
			return nil, err
		}

		if coll.writeConcern != nil {
			wc, err := Opt.WriteConcern(coll.writeConcern)
			if err != nil {
				return nil, err
			}

			opts = append(opts, wc)
		}

		doAggregate := func() (Cursor, error) {
			return ops.Aggregate(ctx, s, coll.namespace(), pipelineArr, true, opts...)
		}

		acknowledged := true

		for _, opt := range opts {
			if wc, ok := opt.(options.OptWriteConcern); ok {
				acknowledged = wc.Acknowledged
				break
			}
		}

		if !acknowledged {
			go func() { _, _ = doAggregate() }()
			return nil, nil
		}

		cursor, err = doAggregate()
	} else {
		var s *ops.SelectedServer
		s, err = coll.getReadableServer(ctx)
		if err != nil {
			return nil, err
		}

		cursor, err = coll.aggregateWithServer(ctx, s, pipelineArr, opts...)
	}

	if err != nil {
		return nil, err
	}

	return cursor, nil
}

func (coll *Collection) aggregateWithServer(ctx context.Context, server *ops.SelectedServer,
	pipeline *bson.Array, opts ...options.AggregateOptioner) (Cursor, error) {
	if coll.readConcern != nil {
		rc, err := Opt.ReadConcern(coll.readConcern)
		if err != nil {
			return nil, err
		}

		opts = append(opts, rc)
	}

	return ops.Aggregate(ctx, server, coll.namespace(), pipeline, false, opts...)
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

	if coll.readConcern != nil {
		rc, err := Opt.ReadConcern(coll.readConcern)
		if err != nil {
			return 0, err
		}

		options = append(options, rc)
	}

	count, err := ops.Count(ctx, s, coll.namespace(), f, options...)
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

	if coll.readConcern != nil {
		rc, err := Opt.ReadConcern(coll.readConcern)
		if err != nil {
			return nil, err
		}

		options = append(options, rc)
	}

	values, err := ops.Distinct(
		ctx,
		s,
		coll.namespace(),
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

	if coll.readConcern != nil {
		rc, err := Opt.ReadConcern(coll.readConcern)
		if err != nil {
			return nil, err
		}

		options = append(options, rc)
	}

	cursor, err := ops.Find(ctx, s, coll.namespace(), f, options...)
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
	opts ...options.FindOneOptioner) *DocumentResult {

	findOpts := make([]options.FindOptioner, 0, len(opts))
	for _, opt := range opts {
		findOpts = append(findOpts, opt.(options.FindOptioner))
	}

	findOpts = append(findOpts, Opt.Limit(1))

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

	if coll.readConcern != nil {
		rc, err := Opt.ReadConcern(coll.readConcern)
		if err != nil {
			return &DocumentResult{err: err}
		}

		opts = append(opts, rc)
	}

	cursor, err := coll.Find(ctx, f, findOpts...)
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

	if coll.writeConcern != nil {
		wc, err := Opt.WriteConcern(coll.writeConcern)
		if err != nil {
			return &DocumentResult{err: err}
		}

		opts = append(opts, wc)
	}

	findOneAndDelete := func() (Cursor, error) {
		return ops.FindOneAndDelete(
			ctx,
			s,
			coll.namespace(),
			f,
			opts...,
		)
	}

	acknowledged := true

	for _, opt := range opts {
		if wc, ok := opt.(options.OptWriteConcern); ok {
			acknowledged = wc.Acknowledged
			break
		}
	}

	if !acknowledged {
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

	if elem, ok := r.ElementAtOK(0); ok && strings.HasPrefix(elem.Key(), "$") {
		return &DocumentResult{err: errors.New("replacement document cannot contains keys beginning with '$")}
	}

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return &DocumentResult{err: err}
	}

	if coll.writeConcern != nil {
		wc, err := Opt.WriteConcern(coll.writeConcern)
		if err != nil {
			return &DocumentResult{err: err}
		}

		opts = append(opts, wc)
	}

	findOneAndReplace := func() (Cursor, error) {
		return ops.FindOneAndReplace(
			ctx,
			s,
			coll.namespace(),
			f,
			r,
			opts...,
		)
	}

	acknowledged := true

	for _, opt := range opts {
		if wc, ok := opt.(options.OptWriteConcern); ok {
			acknowledged = wc.Acknowledged
			break
		}
	}

	if !acknowledged {
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

	if elem, ok := u.ElementAtOK(0); !ok || !strings.HasPrefix(elem.Key(), "$") {
		return &DocumentResult{err: errors.New("update document must contain key beginning with '$")}
	}

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return &DocumentResult{err: err}
	}

	if coll.writeConcern != nil {
		wc, err := Opt.WriteConcern(coll.writeConcern)
		if err != nil {
			return &DocumentResult{err: err}
		}

		opts = append(opts, wc)
	}

	findOneAndUpdate := func() (Cursor, error) {
		return ops.FindOneAndUpdate(
			ctx,
			s,
			coll.namespace(),
			f,
			u,
			opts...,
		)
	}

	acknowledged := true

	for _, opt := range opts {
		if wc, ok := opt.(options.OptWriteConcern); ok {
			acknowledged = wc.Acknowledged
			break
		}
	}

	if !acknowledged {
		// TODO(skriptble): This is probably wrong. We should be returning
		// ErrUnacknowledged but we don't have that yet  ¯\_(ツ)_/¯
		go func() { _, _ = findOneAndUpdate() }()
		return nil
	}

	cur, err := findOneAndUpdate()
	return &DocumentResult{cur: cur, err: err}
}

// Watch returns a change stream cursor used to receive notifications of changes to the collection.
// This method is preferred to running a raw aggregation with a $changeStream stage because it
// supports resumability in the case of some errors.
func (coll *Collection) Watch(ctx context.Context, pipeline interface{},
	opts ...options.ChangeStreamOptioner) (Cursor, error) {
	return newChangeStream(ctx, coll, pipeline, opts...)
}

// Indexes returns the index view for this collection.
func (coll *Collection) Indexes() IndexView {
	return IndexView{coll: coll}
}
