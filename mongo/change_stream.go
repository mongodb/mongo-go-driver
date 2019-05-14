// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/mongo/driverlegacy"
	"go.mongodb.org/mongo-driver/x/network/command"
)

const errorInterrupted int32 = 11601
const errorCappedPositionLost int32 = 136
const errorCursorKilled int32 = 237

// ErrMissingResumeToken indicates that a change stream notification from the server did not
// contain a resume token.
var ErrMissingResumeToken = errors.New("cannot provide resume functionality when the resume token is missing")

// ErrNilCursor indicates that the cursor for the change stream is nil.
var ErrNilCursor = errors.New("cursor is nil")

// ChangeStream instances iterate a stream of change documents. Each document can be decoded via the
// Decode method. Resume tokens should be retrieved via the ResumeToken method and can be stored to
// resume the change stream at a specific point in time.
//
// A typical usage of the ChangeStream type would be:
type ChangeStream struct {
	// Current is the BSON bytes of the current change document. This property is only valid until
	// the next call to Next or Close. If continued access is required to the bson.Raw, you must
	// make a copy of it.
	Current bson.Raw

	cmd         bsonx.Doc // aggregate command to run to create stream and rebuild cursor
	pipeline    bsonx.Arr
	options     *options.ChangeStreamOptions
	coll        *Collection
	db          *Database
	ns          command.Namespace
	cursor      pbrtBatchCursor
	batch       []bsoncore.Document
	cursorOpts  bsonx.Doc
	getMoreOpts bsonx.Doc

	resumeToken   bson.Raw
	err           error
	streamType    StreamType
	client        *Client
	sess          Session
	readPref      *readpref.ReadPref
	readConcern   *readconcern.ReadConcern
	registry      *bsoncodec.Registry
	operationTime *primitive.Timestamp
}

func (cs *ChangeStream) replaceOptions(desc description.SelectedServer) {
	// Cached resume token: use the resume token as the resumeAfter option and set no other resume options
	if cs.resumeToken != nil {
		cs.options.SetResumeAfter(cs.resumeToken)
		cs.options.SetStartAfter(nil)
		cs.options.SetStartAtOperationTime(nil)
		return
	}

	// No cached resume token but cached operation time: use the operation time as the startAtOperationTime option and
	// set no other resume options
	if (cs.operationTime != nil || cs.options.StartAtOperationTime != nil) && desc.WireVersion.Max >= 7 {
		opTime := cs.options.StartAtOperationTime
		if cs.operationTime != nil {
			opTime = cs.operationTime
		}

		cs.options.SetStartAtOperationTime(opTime)
		cs.options.SetResumeAfter(nil)
		cs.options.SetStartAfter(nil)
		return
	}

	// No cached resume token or operation time: set none of the resume options
	cs.options.SetResumeAfter(nil)
	cs.options.SetStartAfter(nil)
	cs.options.SetStartAtOperationTime(nil)
}

// Create options docs for the pipeline and cursor
func createCmdDocs(csType StreamType, opts *options.ChangeStreamOptions, registry *bsoncodec.Registry) (bsonx.Doc,
	bsonx.Doc, bsonx.Doc, bsonx.Doc, error) {

	pipelineDoc := bsonx.Doc{}
	cursorDoc := bsonx.Doc{}
	optsDoc := bsonx.Doc{}
	getMoreOptsDoc := bsonx.Doc{}

	if csType == ClientStream {
		pipelineDoc = pipelineDoc.Append("allChangesForCluster", bsonx.Boolean(true))
	}

	if opts.BatchSize != nil {
		cursorDoc = cursorDoc.Append("batchSize", bsonx.Int32(*opts.BatchSize))
	}
	if opts.Collation != nil {
		collDoc, err := bsonx.ReadDoc(opts.Collation.ToDocument())
		if err != nil {
			return nil, nil, nil, nil, err
		}
		optsDoc = optsDoc.Append("collation", bsonx.Document(collDoc))
	}
	if opts.FullDocument != nil {
		pipelineDoc = pipelineDoc.Append("fullDocument", bsonx.String(string(*opts.FullDocument)))
	}
	if opts.MaxAwaitTime != nil {
		ms := int64(time.Duration(*opts.MaxAwaitTime) / time.Millisecond)
		getMoreOptsDoc = getMoreOptsDoc.Append("maxTimeMS", bsonx.Int64(ms))
	}
	if opts.ResumeAfter != nil {
		rt, err := transformDocument(registry, opts.ResumeAfter)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		pipelineDoc = pipelineDoc.Append("resumeAfter", bsonx.Document(rt))
	}
	if opts.StartAtOperationTime != nil {
		pipelineDoc = pipelineDoc.Append("startAtOperationTime",
			bsonx.Timestamp(opts.StartAtOperationTime.T, opts.StartAtOperationTime.I))
	}
	if opts.StartAfter != nil {
		sa, err := transformDocument(registry, opts.StartAfter)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		pipelineDoc = pipelineDoc.Append("startAfter", bsonx.Document(sa))
	}

	return pipelineDoc, cursorDoc, optsDoc, getMoreOptsDoc, nil
}

func getSession(ctx context.Context, client *Client) (Session, error) {
	sess := sessionFromContext(ctx)
	if err := client.validSession(sess); err != nil {
		return nil, err
	}

	var mongoSess Session
	if sess != nil {
		mongoSess = &sessionImpl{
			clientSession: sess,
			client:        client,
		}
	} else {
		// create implicit session because it will be needed
		newSess, err := session.NewClientSession(client.topology.SessionPool, client.id, session.Implicit)
		if err != nil {
			return nil, err
		}

		mongoSess = &sessionImpl{
			clientSession: newSess,
			client:        client,
		}
	}

	return mongoSess, nil
}

func parseOptions(csType StreamType, opts *options.ChangeStreamOptions, registry *bsoncodec.Registry) (bsonx.Doc,
	bsonx.Doc, bsonx.Doc, bsonx.Doc, error) {

	if opts.FullDocument == nil {
		opts = opts.SetFullDocument(options.Default)
	}

	pipelineDoc, cursorDoc, optsDoc, getMoreOptsDoc, err := createCmdDocs(csType, opts, registry)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return pipelineDoc, cursorDoc, optsDoc, getMoreOptsDoc, nil
}

func (cs *ChangeStream) runCommand(ctx context.Context, replaceOptions bool) error {
	ss, err := cs.client.topology.SelectServerLegacy(ctx, cs.db.writeSelector)
	if err != nil {
		return replaceErrors(err)
	}

	desc := ss.Description()
	conn, err := ss.ConnectionLegacy(ctx)
	if err != nil {
		return replaceErrors(err)
	}
	defer conn.Close()

	if replaceOptions {
		cs.replaceOptions(desc)
		optionsDoc, _, _, _, err := createCmdDocs(cs.streamType, cs.options, cs.registry)
		if err != nil {
			return err
		}

		changeStreamDoc := bsonx.Doc{
			{"$changeStream", bsonx.Document(optionsDoc)},
		}
		cs.pipeline[0] = bsonx.Document(changeStreamDoc)
		cs.cmd.Set("pipeline", bsonx.Array(cs.pipeline))
	}

	readCmd := command.Read{
		DB:          cs.db.name,
		Command:     cs.cmd,
		Session:     cs.sess.(*sessionImpl).clientSession,
		Clock:       cs.client.clock,
		ReadPref:    cs.readPref,
		ReadConcern: cs.readConcern,
	}

	rdr, err := readCmd.RoundTrip(ctx, desc, conn)
	if err != nil {
		cs.sess.EndSession(ctx)
		return replaceErrors(err)
	}

	batchCursor, err := driverlegacy.NewBatchCursor(bsoncore.Document(rdr), readCmd.Session, readCmd.Clock, ss.Server, cs.getMoreOpts...)
	if err != nil {
		cs.sess.EndSession(ctx)
		return replaceErrors(err)
	}
	cs.cursor = batchCursor

	cursorValue, err := rdr.LookupErr("cursor")
	if err != nil {
		return err
	}
	cursorDoc := cursorValue.Document()
	cs.ns = command.ParseNamespace(cursorDoc.Lookup("ns").StringValue())

	// Cache the operation time from the aggregate response if necessary
	if cs.options.StartAtOperationTime == nil && cs.options.ResumeAfter == nil && cs.options.StartAfter == nil && desc.WireVersion.Max >= 7 &&
		cs.emptyBatch() && cs.cursor.PostBatchResumeToken() == nil {

		opTime, err := rdr.LookupErr("operationTime")
		if err != nil {
			return err
		}

		t, i, ok := opTime.TimestampOK()
		if !ok {
			return fmt.Errorf("operationTime was of type %s not %s", opTime.Type, bson.TypeTimestamp)
		}
		cs.operationTime = &primitive.Timestamp{T: t, I: i}
	}

	// Cache the post batch resume token from the aggreate response if necessary
	cs.updatePbrtFromCommand()
	return nil
}

// Returns true if the underlying cursor's batch is empty
func (cs *ChangeStream) emptyBatch() bool {
	return len(cs.cursor.Batch().Data) == 5 // empty BSON array
}

// Updates the post batch resume token after a successful aggregate or getMore operation.
func (cs *ChangeStream) updatePbrtFromCommand() {
	// Only cache the pbrt if an empty batch was returned and a pbrt was included
	pbrt := cs.cursor.PostBatchResumeToken()
	if cs.emptyBatch() && pbrt != nil {
		cs.resumeToken = bson.Raw(pbrt)
	}
}

func newChangeStream(ctx context.Context, coll *Collection, pipeline interface{},
	opts ...*options.ChangeStreamOptions) (*ChangeStream, error) {

	pipelineArr, err := transformAggregatePipeline(coll.registry, pipeline)
	if err != nil {
		return nil, err
	}

	csOpts := options.MergeChangeStreamOptions(opts...)
	pipelineDoc, cursorDoc, optsDoc, getMoreDoc, err := parseOptions(CollectionStream, csOpts, coll.registry)
	if err != nil {
		return nil, err
	}
	sess, err := getSession(ctx, coll.client)
	if err != nil {
		return nil, err
	}

	csDoc := bsonx.Document(bsonx.Doc{
		{"$changeStream", bsonx.Document(pipelineDoc)},
	})
	pipelineArr = append(bsonx.Arr{csDoc}, pipelineArr...)

	cmd := bsonx.Doc{
		{"aggregate", bsonx.String(coll.name)},
		{"pipeline", bsonx.Array(pipelineArr)},
		{"cursor", bsonx.Document(cursorDoc)},
	}
	cmd = append(cmd, optsDoc...)

	// When starting a change stream, cache startAfter as the first resume token if it is set. If not, cache
	// resumeAfter. If neither is set, do not cache a resume token.
	resumeToken := csOpts.StartAfter
	if resumeToken == nil {
		resumeToken = csOpts.ResumeAfter
	}
	var marshaledToken bson.Raw
	if resumeToken != nil {
		marshaledToken, err = bson.Marshal(resumeToken)
		if err != nil {
			return nil, err
		}
	}

	cs := &ChangeStream{
		client:      coll.client,
		sess:        sess,
		cmd:         cmd,
		pipeline:    pipelineArr,
		coll:        coll,
		db:          coll.db,
		streamType:  CollectionStream,
		readPref:    coll.readPreference,
		readConcern: coll.readConcern,
		options:     csOpts,
		registry:    coll.registry,
		cursorOpts:  cursorDoc,
		getMoreOpts: getMoreDoc,
		resumeToken: marshaledToken,
	}

	err = cs.runCommand(ctx, false)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func newDbChangeStream(ctx context.Context, db *Database, pipeline interface{},
	opts ...*options.ChangeStreamOptions) (*ChangeStream, error) {

	pipelineArr, err := transformAggregatePipeline(db.registry, pipeline)
	if err != nil {
		return nil, err
	}

	csOpts := options.MergeChangeStreamOptions(opts...)
	pipelineDoc, cursorDoc, optsDoc, getMoreDoc, err := parseOptions(DatabaseStream, csOpts, db.registry)
	if err != nil {
		return nil, err
	}
	sess, err := getSession(ctx, db.client)
	if err != nil {
		return nil, err
	}

	csDoc := bsonx.Document(bsonx.Doc{
		{"$changeStream", bsonx.Document(pipelineDoc)},
	})
	pipelineArr = append(bsonx.Arr{csDoc}, pipelineArr...)

	cmd := bsonx.Doc{
		{"aggregate", bsonx.Int32(1)},
		{"pipeline", bsonx.Array(pipelineArr)},
		{"cursor", bsonx.Document(cursorDoc)},
	}
	cmd = append(cmd, optsDoc...)

	cs := &ChangeStream{
		client:      db.client,
		db:          db,
		sess:        sess,
		cmd:         cmd,
		pipeline:    pipelineArr,
		streamType:  DatabaseStream,
		readPref:    db.readPreference,
		readConcern: db.readConcern,
		options:     csOpts,
		registry:    db.registry,
		cursorOpts:  cursorDoc,
		getMoreOpts: getMoreDoc,
	}

	err = cs.runCommand(ctx, false)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func newClientChangeStream(ctx context.Context, client *Client, pipeline interface{},
	opts ...*options.ChangeStreamOptions) (*ChangeStream, error) {

	pipelineArr, err := transformAggregatePipeline(client.registry, pipeline)
	if err != nil {
		return nil, err
	}

	csOpts := options.MergeChangeStreamOptions(opts...)
	pipelineDoc, cursorDoc, optsDoc, getMoreDoc, err := parseOptions(ClientStream, csOpts, client.registry)
	if err != nil {
		return nil, err
	}
	sess, err := getSession(ctx, client)
	if err != nil {
		return nil, err
	}

	csDoc := bsonx.Document(bsonx.Doc{
		{"$changeStream", bsonx.Document(pipelineDoc)},
	})
	pipelineArr = append(bsonx.Arr{csDoc}, pipelineArr...)

	cmd := bsonx.Doc{
		{"aggregate", bsonx.Int32(1)},
		{"pipeline", bsonx.Array(pipelineArr)},
		{"cursor", bsonx.Document(cursorDoc)},
	}
	cmd = append(cmd, optsDoc...)

	cs := &ChangeStream{
		client:      client,
		db:          client.Database("admin"),
		sess:        sess,
		cmd:         cmd,
		pipeline:    pipelineArr,
		streamType:  ClientStream,
		readPref:    client.readPreference,
		readConcern: client.readConcern,
		options:     csOpts,
		registry:    client.registry,
		cursorOpts:  cursorDoc,
		getMoreOpts: getMoreDoc,
	}

	err = cs.runCommand(ctx, false)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func (cs *ChangeStream) storeResumeToken() error {
	// If cs.Current is the last document in the batch and a pbrt is included, cache the pbrt
	// Otherwise, cache the _id of the document

	var tokenDoc bson.Raw
	if len(cs.batch) == 0 {
		pbrt := cs.cursor.PostBatchResumeToken()
		if pbrt != nil {
			tokenDoc = bson.Raw(pbrt)
		}
	}

	if tokenDoc == nil {
		var ok bool
		tokenDoc, ok = cs.Current.Lookup("_id").DocumentOK()
		if !ok {
			_ = cs.Close(context.Background())
			return ErrMissingResumeToken
		}
	}

	cs.resumeToken = tokenDoc
	return nil
}

// ID returns the cursor ID for this change stream.
func (cs *ChangeStream) ID() int64 {
	if cs.cursor == nil {
		return 0
	}

	return cs.cursor.ID()
}

// Next gets the next result from this change stream. Returns true if there were no errors and the next
// result is available for decoding.
func (cs *ChangeStream) Next(ctx context.Context) bool {
	if ctx == nil {
		ctx = context.Background()
	}

	if len(cs.batch) == 0 {
		cs.loopNext(ctx)
		if cs.err != nil || len(cs.batch) == 0 {
			return false
		}
	}

	cs.Current = bson.Raw(cs.batch[0])
	cs.batch = cs.batch[1:]
	err := cs.storeResumeToken()
	if err != nil {
		cs.err = err
		return false
	}
	return true
}

func (cs *ChangeStream) loopNext(ctx context.Context) {
	for {
		if cs.cursor == nil {
			return
		}

		if cs.cursor.Next(ctx) {
			// If this is the first batch, the batch cursor will return true, but the batch could be empty.
			cs.batch, cs.err = cs.cursor.Batch().Documents()
			if cs.err != nil || len(cs.batch) > 0 {
				return
			}

			// no error but empty batch
			cs.updatePbrtFromCommand()
			continue
		}

		cs.err = cs.cursor.Err()
		if cs.err == nil {
			// If a getMore was done but the batch was empty, the batch cursor will return false with no error
			if len(cs.batch) == 0 {
				continue
			}

			return
		}

		switch t := cs.err.(type) {
		case command.Error:
			if t.Code == errorInterrupted || t.Code == errorCappedPositionLost || t.Code == errorCursorKilled {
				return
			}
		}

		_, _ = driverlegacy.KillCursors(ctx, cs.ns, cs.cursor.Server(), cs.ID())
		cs.err = cs.runCommand(ctx, true)
		if cs.err != nil {
			return
		}
	}
}

// Decode will decode the current document into val.
func (cs *ChangeStream) Decode(out interface{}) error {
	if cs.cursor == nil {
		return ErrNilCursor
	}

	return bson.UnmarshalWithRegistry(cs.registry, cs.Current, out)
}

// Err returns the current error.
func (cs *ChangeStream) Err() error {
	if cs.err != nil {
		return replaceErrors(cs.err)
	}
	if cs.cursor == nil {
		return nil
	}

	return cs.cursor.Err()
}

// Close closes this cursor.
func (cs *ChangeStream) Close(ctx context.Context) error {
	if cs.cursor == nil {
		return nil // cursor is already closed
	}

	return replaceErrors(cs.cursor.Close(ctx))
}

// ResumeToken returns the last cached resume token for this change stream.
func (cs *ChangeStream) ResumeToken() bson.Raw {
	return cs.resumeToken
}

// StreamType represents the type of a change stream.
type StreamType uint8

// These constants represent valid change stream types. A change stream can be initialized over a collection, all
// collections in a database, or over a whole client.
const (
	CollectionStream StreamType = iota
	DatabaseStream
	ClientStream
)
