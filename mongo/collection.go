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

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/dispatch"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// Collection performs operations on a given collection.
type Collection struct {
	client         *Client
	db             *Database
	name           string
	readConcern    *readconcern.ReadConcern
	writeConcern   *writeconcern.WriteConcern
	readPreference *readpref.ReadPref
	readSelector   description.ServerSelector
	writeSelector  description.ServerSelector
}

func newCollection(db *Database, name string) *Collection {
	coll := &Collection{
		client:         db.client,
		db:             db,
		name:           name,
		readPreference: db.readPreference,
		readConcern:    db.readConcern,
		writeConcern:   db.writeConcern,
		readSelector:   db.readSelector,
		writeSelector:  db.writeSelector,
	}

	return coll
}

// Name provides access to the name of the collection.
func (coll *Collection) Name() string {
	return coll.name
}

// namespace returns the namespace of the collection.
func (coll *Collection) namespace() command.Namespace {
	return command.NewNamespace(coll.db.name, coll.name)
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
	opts ...option.InsertOneOptioner) (*InsertOneResult, error) {

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

	newOptions := make([]option.InsertOptioner, 0, len(opts))
	for _, opt := range opts {
		newOptions = append(newOptions, opt)
	}

	oldns := coll.namespace()
	cmd := command.Insert{
		NS:   command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		Docs: []*bson.Document{doc},
		Opts: newOptions,
	}

	res, err := dispatch.Insert(ctx, cmd, coll.client.topology, coll.writeSelector, coll.writeConcern)
	rr, err := processWriteError(res.WriteConcernError, res.WriteErrors, err)
	if rr&rrOne == 0 {
		return nil, err
	}
	return &InsertOneResult{InsertedID: insertedID}, err
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
	opts ...option.InsertManyOptioner) (*InsertManyResult, error) {

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

	newOptions := make([]option.InsertOptioner, 0, len(opts))
	for _, opt := range opts {
		newOptions = append(newOptions, opt)
	}

	oldns := coll.namespace()
	cmd := command.Insert{
		NS:   command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		Docs: docs,
		Opts: newOptions,
	}

	res, err := dispatch.Insert(ctx, cmd, coll.client.topology, coll.writeSelector, coll.writeConcern)
	switch err {
	case nil:
	case dispatch.ErrUnacknowledgedWrite:
		return &InsertManyResult{InsertedIDs: result}, ErrUnacknowledgedWrite
	default:
		return nil, err
	}
	if len(res.WriteErrors) > 0 || res.WriteConcernError != nil {
		err = BulkWriteError{
			WriteErrors:       writeErrorsFromResult(res.WriteErrors),
			WriteConcernError: convertWriteConcernError(res.WriteConcernError),
		}
	}
	return &InsertManyResult{InsertedIDs: result}, err
}

// DeleteOne deletes a single document from the collection. A user can supply
// a custom context to this method, or nil to default to context.Background().
//
// This method uses TransformDocument to turn the filter parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// filter.
func (coll *Collection) DeleteOne(ctx context.Context, filter interface{},
	opts ...option.DeleteOptioner) (*DeleteResult, error) {

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

	oldns := coll.namespace()
	cmd := command.Delete{
		NS:      command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		Deletes: deleteDocs,
		Opts:    opts,
	}

	res, err := dispatch.Delete(ctx, cmd, coll.client.topology, coll.writeSelector, coll.writeConcern)
	rr, err := processWriteError(res.WriteConcernError, res.WriteErrors, err)
	if rr&rrOne == 0 {
		return nil, err
	}
	return &DeleteResult{DeletedCount: int64(res.N)}, err
}

// DeleteMany deletes multiple documents from the collection. A user can
// supply a custom context to this method, or nil to default to
// context.Background().
//
// This method uses TransformDocument to turn the filter parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// filter.
func (coll *Collection) DeleteMany(ctx context.Context, filter interface{},
	opts ...option.DeleteOptioner) (*DeleteResult, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := TransformDocument(filter)
	if err != nil {
		return nil, err
	}
	deleteDocs := []*bson.Document{bson.NewDocument(bson.EC.SubDocument("q", f), bson.EC.Int32("limit", 0))}

	oldns := coll.namespace()
	cmd := command.Delete{
		NS:      command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		Deletes: deleteDocs,
		Opts:    opts,
	}

	res, err := dispatch.Delete(ctx, cmd, coll.client.topology, coll.writeSelector, coll.writeConcern)
	rr, err := processWriteError(res.WriteConcernError, res.WriteErrors, err)
	if rr&rrMany == 0 {
		return nil, err
	}
	return &DeleteResult{DeletedCount: int64(res.N)}, err
}

func (coll *Collection) updateOrReplaceOne(ctx context.Context, filter,
	update *bson.Document, opts ...option.UpdateOptioner) (*UpdateResult, error) {

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

	oldns := coll.namespace()
	cmd := command.Update{
		NS:   command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		Docs: updateDocs,
		Opts: opts,
	}

	r, err := dispatch.Update(ctx, cmd, coll.client.topology, coll.writeSelector, coll.writeConcern)
	if err != nil && err != dispatch.ErrUnacknowledgedWrite {
		return nil, err
	}
	res := &UpdateResult{
		MatchedCount:  r.MatchedCount,
		ModifiedCount: r.ModifiedCount,
	}
	if len(r.Upserted) > 0 {
		res.UpsertedID = r.Upserted[0].ID
		res.MatchedCount--
	}

	rr, err := processWriteError(r.WriteConcernError, r.WriteErrors, err)
	if rr&rrOne == 0 {
		return nil, err
	}
	return res, err
}

// UpdateOne updates a single document in the collection. A user can supply a
// custom context to this method, or nil to default to context.Background().
//
// This method uses TransformDocument to turn the filter and update parameter
// into a *bson.Document. See TransformDocument for the list of valid types for
// filter and update.
func (coll *Collection) UpdateOne(ctx context.Context, filter interface{}, update interface{},
	options ...option.UpdateOptioner) (*UpdateResult, error) {

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
	opts ...option.UpdateOptioner) (*UpdateResult, error) {

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

	oldns := coll.namespace()
	cmd := command.Update{
		NS:   command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		Docs: updateDocs,
		Opts: opts,
	}

	r, err := dispatch.Update(ctx, cmd, coll.client.topology, coll.writeSelector, coll.writeConcern)
	if err != nil && err != dispatch.ErrUnacknowledgedWrite {
		return nil, err
	}
	res := &UpdateResult{
		MatchedCount:  r.MatchedCount,
		ModifiedCount: r.ModifiedCount,
	}
	// TODO(skriptble): Is this correct? Do we only return the first upserted ID for an UpdateMany?
	if len(r.Upserted) > 0 {
		res.UpsertedID = r.Upserted[0].ID
		res.MatchedCount--
	}

	rr, err := processWriteError(r.WriteConcernError, r.WriteErrors, err)
	if rr&rrMany == 0 {
		return nil, err
	}
	return res, err
}

// ReplaceOne replaces a single document in the collection. A user can supply
// a custom context to this method, or nil to default to context.Background().
//
// This method uses TransformDocument to turn the filter and replacement
// parameter into a *bson.Document. See TransformDocument for the list of
// valid types for filter and replacement.
func (coll *Collection) ReplaceOne(ctx context.Context, filter interface{},
	replacement interface{}, opts ...option.ReplaceOptioner) (*UpdateResult, error) {

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

	updateOptions := make([]option.UpdateOptioner, 0, len(opts))
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
	opts ...option.AggregateOptioner) (Cursor, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	pipelineArr, err := transformAggregatePipeline(pipeline)
	if err != nil {
		return nil, err
	}

	oldns := coll.namespace()
	cmd := command.Aggregate{
		NS:       command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		Pipeline: pipelineArr,
		Opts:     opts,
		ReadPref: coll.readPreference,
	}
	return dispatch.Aggregate(ctx, cmd, coll.client.topology, coll.readSelector, coll.writeSelector, coll.writeConcern)
}

// Count gets the number of documents matching the filter. A user can supply a
// custom context to this method, or nil to default to context.Background().
//
// This method uses TransformDocument to turn the filter parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// filter.
func (coll *Collection) Count(ctx context.Context, filter interface{},
	opts ...option.CountOptioner) (int64, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	f, err := TransformDocument(filter)
	if err != nil {
		return 0, err
	}

	oldns := coll.namespace()
	cmd := command.Count{
		NS:       command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		Query:    f,
		Opts:     opts,
		ReadPref: coll.readPreference,
	}
	return dispatch.Count(ctx, cmd, coll.client.topology, coll.readSelector, coll.readConcern)
}

// Distinct finds the distinct values for a specified field across a single
// collection. A user can supply a custom context to this method, or nil to
// default to context.Background().
//
// This method uses TransformDocument to turn the filter parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// filter.
func (coll *Collection) Distinct(ctx context.Context, fieldName string, filter interface{},
	opts ...option.DistinctOptioner) ([]interface{}, error) {

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

	oldns := coll.namespace()
	cmd := command.Distinct{
		NS:       command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		Field:    fieldName,
		Query:    f,
		Opts:     opts,
		ReadPref: coll.readPreference,
	}
	res, err := dispatch.Distinct(ctx, cmd, coll.client.topology, coll.readSelector, coll.readConcern)
	if err != nil {
		return nil, err
	}

	return res.Values, nil
}

// Find finds the documents matching a model. A user can supply a custom context to this
// method.
//
// This method uses TransformDocument to turn the filter parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// filter.
func (coll *Collection) Find(ctx context.Context, filter interface{},
	opts ...option.FindOptioner) (Cursor, error) {

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

	oldns := coll.namespace()
	cmd := command.Find{
		NS:       command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		Filter:   f,
		Opts:     opts,
		ReadPref: coll.readPreference,
	}
	return dispatch.Find(ctx, cmd, coll.client.topology, coll.readSelector, coll.readConcern)
}

// FindOne returns up to one document that matches the model. A user can
// supply a custom context to this method, or nil to default to
// context.Background().
//
// This method uses TransformDocument to turn the filter parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// filter.
func (coll *Collection) FindOne(ctx context.Context, filter interface{},
	opts ...option.FindOneOptioner) *DocumentResult {

	findOpts := make([]option.FindOptioner, 0, len(opts))
	for _, opt := range opts {
		findOpts = append(findOpts, opt.(option.FindOptioner))
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

	oldns := coll.namespace()
	cmd := command.Find{
		NS:       command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		Filter:   f,
		Opts:     findOpts,
		ReadPref: coll.readPreference,
	}
	cursor, err := dispatch.Find(ctx, cmd, coll.client.topology, coll.readSelector, coll.readConcern)
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
	opts ...option.FindOneAndDeleteOptioner) *DocumentResult {

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

	oldns := coll.namespace()
	cmd := command.FindOneAndDelete{
		NS:    command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		Query: f,
		Opts:  opts,
	}
	res, err := dispatch.FindOneAndDelete(ctx, cmd, coll.client.topology, coll.writeSelector, coll.writeConcern)
	if err != nil {
		return &DocumentResult{err: err}
	}

	return &DocumentResult{rdr: res.Value}
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
	replacement interface{}, opts ...option.FindOneAndReplaceOptioner) *DocumentResult {

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

	oldns := coll.namespace()
	cmd := command.FindOneAndReplace{
		NS:          command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		Query:       f,
		Replacement: r,
		Opts:        opts,
	}
	res, err := dispatch.FindOneAndReplace(ctx, cmd, coll.client.topology, coll.writeSelector, coll.writeConcern)
	if err != nil {
		return &DocumentResult{err: err}
	}

	return &DocumentResult{rdr: res.Value}
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
	update interface{}, opts ...option.FindOneAndUpdateOptioner) *DocumentResult {

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

	oldns := coll.namespace()
	cmd := command.FindOneAndUpdate{
		NS:     command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		Query:  f,
		Update: u,
		Opts:   opts,
	}
	res, err := dispatch.FindOneAndUpdate(ctx, cmd, coll.client.topology, coll.writeSelector, coll.writeConcern)
	if err != nil {
		return &DocumentResult{err: err}
	}

	return &DocumentResult{rdr: res.Value}
}

// Watch returns a change stream cursor used to receive notifications of changes to the collection.
// This method is preferred to running a raw aggregation with a $changeStream stage because it
// supports resumability in the case of some errors.
func (coll *Collection) Watch(ctx context.Context, pipeline interface{},
	opts ...option.ChangeStreamOptioner) (Cursor, error) {
	return newChangeStream(ctx, coll, pipeline, opts...)
}

// Indexes returns the index view for this collection.
func (coll *Collection) Indexes() IndexView {
	return IndexView{coll: coll}
}

// Drop drops this collection from database.
func (coll *Collection) Drop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	cmd := command.DropCollection{
		DB:         coll.db.name,
		Collection: coll.name,
	}
	_, err := dispatch.DropCollection(ctx, cmd, coll.client.topology, coll.writeSelector)
	if err != nil && !command.IsNotFound(err) {
		return err
	}
	return nil
}
