// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"
	"strings"

	"github.com/10gen/mongo-go-driver/mongo/options"
	"github.com/10gen/mongo-go-driver/mongo/private/cluster"
	"github.com/10gen/mongo-go-driver/mongo/private/ops"
	"github.com/10gen/mongo-go-driver/mongo/readconcern"
	"github.com/10gen/mongo-go-driver/mongo/readpref"
	"github.com/10gen/mongo-go-driver/mongo/writeconcern"
	"github.com/skriptble/wilson/bson"
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
// TODO GODRIVER-76: Document which types for interface{} are valid.
//
// TODO(skriptble): Determine if we should unwrap the value for the
// InsertOneResult or just return the bson.Element or a bson.Value.
func (coll *Collection) InsertOne(ctx context.Context, document interface{},
	options ...options.InsertOption) (*InsertOneResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	doc, err := transformDocument(document)
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

	insert := func() error {
		return ops.Insert(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			[]*bson.Document{doc},
			nil,
			options...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		go func() { _ = insert() }()
		return nil, nil
	}

	err = insert()

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
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) InsertMany(ctx context.Context, documents []interface{},
	options ...options.InsertOption) (*InsertManyResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	result := make([]interface{}, len(documents))
	docs := make([]*bson.Document, len(documents))

	for i, doc := range documents {
		bdoc, err := transformDocument(doc)
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

	insert := func() error {
		return ops.Insert(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			docs,
			nil,
			options...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		go func() { _ = insert() }()
		return nil, nil
	}

	err = insert()

	if err != nil {
		return nil, err
	}

	return &InsertManyResult{InsertedIDs: result}, nil
}

// DeleteOne deletes a single document from the collection. A user can supply
// a custom context to this method, or nil to default to context.Background().
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) DeleteOne(ctx context.Context, filter interface{},
	options ...options.DeleteOption) (*DeleteResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := transformDocument(filter)
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
	doDelete := func() error {
		return ops.Delete(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			deleteDocs,
			&result,
			options...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		go func() { _ = doDelete() }()
		return nil, nil
	}

	err = doDelete()

	if err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteMany deletes multiple documents from the collection. A user can
// supply a custom context to this method, or nil to default to
// context.Background().
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) DeleteMany(ctx context.Context, filter interface{},
	options ...options.DeleteOption) (*DeleteResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := transformDocument(filter)
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
	doDelete := func() error {
		return ops.Delete(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			deleteDocs,
			&result,
			options...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		go func() { _ = doDelete() }()
		return nil, nil
	}

	err = doDelete()

	if err != nil {
		return nil, err
	}

	return &result, nil
}

// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) updateOrReplaceOne(ctx context.Context, filter interface{},
	update interface{}, options ...options.UpdateOption) (*UpdateResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := transformDocument(filter)
	if err != nil {
		return nil, err
	}

	u, err := transformDocument(update)
	if err != nil {
		return nil, err
	}

	updateDocs := []*bson.Document{
		bson.NewDocument(
			bson.C.SubDocument("q", f), bson.C.SubDocument("u", u), bson.C.Boolean("multi", false)),
	}
	// updateDocs := []oldbson.D{{
	// 	{Name: "q", Value: filter},
	// 	{Name: "u", Value: update},
	// 	{Name: "multi", Value: false},
	// }}

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	var result UpdateResult = UpdateResult{}
	doUpdate := func() error {
		return ops.Update(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			updateDocs,
			&result,
			options...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		go func() { _ = doUpdate() }()
		return nil, nil
	}

	err = doUpdate()

	if err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateOne updates a single document in the collection. A user can supply a
// custom context to this method, or nil to default to context.Background().
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) UpdateOne(ctx context.Context, filter interface{}, update interface{},
	options ...options.UpdateOption) (*UpdateResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := transformDocument(filter)
	if err != nil {
		return nil, err
	}

	u, err := transformDocument(update)
	if err != nil {
		return nil, err
	}

	if err := ensureDollarKey2(u); err != nil {
		return nil, err
	}

	return coll.updateOrReplaceOne(ctx, f, u, options...)
}

// UpdateMany updates multiple documents in the collection. A user can supply
// a custom context to this method, or nil to default to context.Background().
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) UpdateMany(ctx context.Context, filter interface{}, update interface{},
	options ...options.UpdateOption) (*UpdateResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := transformDocument(filter)
	if err != nil {
		return nil, err
	}

	u, err := transformDocument(update)
	if err != nil {
		return nil, err
	}

	if err := ensureDollarKey2(u); err != nil {
		return nil, err
	}

	updateDocs := []*bson.Document{
		bson.NewDocument(
			bson.C.SubDocument("q", f), bson.C.SubDocument("u", u), bson.C.Boolean("multi", true)),
	}
	// updateDocs := []oldbson.D{{
	// 	{Name: "q", Value: filter},
	// 	{Name: "u", Value: update},
	// 	{Name: "multi", Value: true},
	// }}

	// TODO GODRIVER-27: write concern

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	var result UpdateResult
	doUpdate := func() error {
		return ops.Update(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			updateDocs,
			&result,
			options...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		go func() { _ = doUpdate() }()
		return nil, nil
	}

	err = doUpdate()

	if err != nil {
		return nil, err
	}

	return &result, nil
}

// ReplaceOne replaces a single document in the collection. A user can supply
// a custom context to this method, or nil to default to context.Background().
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) ReplaceOne(ctx context.Context, filter interface{},
	replacement interface{}, options ...options.UpdateOption) (*UpdateResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := transformDocument(filter)
	if err != nil {
		return nil, err
	}

	r, err := transformDocument(replacement)
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
	// bytes, err := oldbson.Marshal(replacement)
	// if err != nil {
	// 	return nil, err
	// }
	//
	// // TODO GODRIVER-111: Roundtrip is inefficient.
	// var doc oldbson.D
	// err = oldbson.Unmarshal(bytes, &doc)
	// if err != nil {
	// 	return nil, err
	// }
	//
	// if len(doc) > 0 && doc[0].Name[0] == '$' {
	// 	return nil, errors.New("replacement document cannot contains keys beginning with '$")
	// }
	//
	return coll.updateOrReplaceOne(ctx, f, r, options...)
}

// Aggregate runs an aggregation framework pipeline. A user can supply a custom context to
// this method.
//
// See https://docs.mongodb.com/manual/aggregation/.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) Aggregate(ctx context.Context, pipeline interface{},
	options ...options.AggregateOption) (Cursor, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	var pipelineArr *bson.Array
	switch t := pipeline.(type) {
	case *bson.Array:
		pipelineArr = t
	default:
		p, err := transformDocument(pipeline)
		if err != nil {
			return nil, err
		}

		pipelineArr = bson.ArrayFromDocument(p)
	}

	// TODO GODRIVER-95: Check for $out and use readable server/read preference if not found
	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	cursor, err := ops.Aggregate(ctx, s, coll.namespace(), coll.readConcern, pipelineArr, options...)
	if err != nil {
		return nil, err
	}

	return cursor, nil
}

// Count gets the number of documents matching the filter. A user can supply a
// custom context to this method, or nil to default to context.Background().
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) Count(ctx context.Context, filter interface{},
	options ...options.CountOption) (int64, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := transformDocument(filter)
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
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) Distinct(ctx context.Context, fieldName string, filter interface{},
	options ...options.DistinctOption) ([]interface{}, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	var f *bson.Document
	var err error
	if filter != nil {
		f, err = transformDocument(filter)
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
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) Find(ctx context.Context, filter interface{},
	options ...options.FindOption) (Cursor, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	var f *bson.Document
	var err error
	if filter != nil {
		f, err = transformDocument(filter)
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
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) FindOne(ctx context.Context, filter interface{}, result interface{},
	options ...options.FindOption) (bool, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	options = append(options, Limit(1))

	var f *bson.Document
	var err error
	if filter != nil {
		f, err = transformDocument(filter)
		if err != nil {
			return false, err
		}
	}

	cursor, err := coll.Find(ctx, f, options...)
	if err != nil {
		return false, err
	}

	found := cursor.Next(ctx, result)
	if err = cursor.Err(); err != nil {
		return false, err
	}

	return found, nil
}

// FindOneAndDelete find a single document and deletes it, returning the
// original in result.  The document to return may be nil.
//
// A user can supply a custom context to this method, or nil to default to
// context.Background().
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) FindOneAndDelete(ctx context.Context, filter interface{},
	result interface{}, opts ...options.FindOneAndDeleteOption) (bool, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	var f *bson.Document
	var err error
	if filter != nil {
		f, err = transformDocument(filter)
		if err != nil {
			return false, err
		}
	}

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return false, err
	}

	findOneAndDelete := func() (bool, error) {
		return ops.FindOneAndDelete(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			f,
			result,
			opts...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		go func() { _, _ = findOneAndDelete() }()
		return false, nil
	}

	return findOneAndDelete()
}

// FindOneAndReplace finds a single document and replaces it, returning either
// the original or the replaced document. The document to return may be nil.
//
// A user can supply a custom context to this method, or nil to default to
// context.Background().
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) FindOneAndReplace(ctx context.Context, filter interface{},
	replacement interface{}, result interface{}, opts ...options.FindOneAndReplaceOption) (bool, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := transformDocument(filter)
	if err != nil {
		return false, err
	}

	r, err := transformDocument(replacement)
	if err != nil {
		return false, err
	}

	elem, err := r.ElementAt(0)
	if err != nil && err != bson.ErrOutOfBounds {
		return false, err
	}

	if strings.HasPrefix(elem.Key(), "$") {
		return false, errors.New("replacement document cannot contains keys beginning with '$")
	}

	// bytes, err := oldbson.Marshal(replacement)
	// if err != nil {
	// 	return false, err
	// }
	//
	// // TODO GODRIVER-111: Roundtrip is inefficient.
	// var doc oldbson.D
	// err = oldbson.Unmarshal(bytes, &doc)
	// if err != nil {
	// 	return false, err
	// }
	//
	// if len(doc) > 0 && doc[0].Name[0] == '$' {
	// 	return false, errors.New("replacement document cannot contains keys beginning with '$")
	// }

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return false, err
	}

	findOneAndReplace := func() (bool, error) {
		return ops.FindOneAndReplace(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			f,
			r,
			result,
			opts...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		go func() { _, _ = findOneAndReplace() }()
		return false, nil
	}

	return findOneAndReplace()
}

// FindOneAndUpdate finds a single document and updates it, returning either
// the original or the updated. The document to return may be nil.
//
// A user can supply a custom context to this method, or nil to default to
// context.Background().
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) FindOneAndUpdate(ctx context.Context, filter interface{},
	update interface{}, result interface{}, opts ...options.FindOneAndUpdateOption) (bool, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := transformDocument(filter)
	if err != nil {
		return false, err
	}

	u, err := transformDocument(update)
	if err != nil {
		return false, err
	}

	elem, err := u.ElementAt(0)
	if err != nil && err != bson.ErrOutOfBounds {
		return false, err
	}

	if !strings.HasPrefix(elem.Key(), "$") {
		return false, errors.New("update document must contain key beginning with '$")
	}

	// bytes, err := oldbson.Marshal(update)
	// if err != nil {
	// 	return false, err
	// }
	//
	// // TODO GODRIVER-111: Roundtrip is inefficient.
	// var doc oldbson.D
	// err = oldbson.Unmarshal(bytes, &doc)
	// if err != nil {
	// 	return false, err
	// }
	//
	// if len(doc) > 0 && doc[0].Name[0] != '$' {
	// 	return false, errors.New("update document must contain key beginning with '$")
	// }

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return false, err
	}

	findOneAndUpdate := func() (bool, error) {
		return ops.FindOneAndUpdate(
			ctx,
			s,
			coll.namespace(),
			coll.writeConcern,
			f,
			u,
			result,
			opts...,
		)
	}

	if !coll.writeConcern.Acknowledged() {
		go func() { _, _ = findOneAndUpdate() }()
		return false, nil
	}

	return findOneAndUpdate()
}
