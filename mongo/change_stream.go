// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/options"
)

// ErrMissingResumeToken indicates that a change stream notification from the server did not
// contain a resume token.
var ErrMissingResumeToken = errors.New("cannot provide resume functionality when the resume token is missing")

type changeStream struct {
	pipeline    *bson.Array
	options     []*bson.Element
	coll        *Collection
	cursor      Cursor
	session     *session.Client
	clock       *session.ClusterClock
	resumeToken *bson.Document
	err         error
}

const errorCodeNotMaster int32 = 10107
const errorCodeCursorNotFound int32 = 43

func newChangeStream(ctx context.Context, coll *Collection, pipeline interface{},
	opts ...*options.ChangeStreamOptions) (*changeStream, error) {

	pipelineArr, err := transformAggregatePipeline(coll.registry, pipeline)
	if err != nil {
		return nil, err
	}

	csOpts := options.ToChangeStreamOptions(opts...)
	sess := sessionFromContext(ctx)

	err = coll.client.ValidSession(sess)
	if err != nil {
		return nil, err
	}

	changeStreamOptions := make([]*bson.Element, 0)
	aggOpts := options.Aggregate()

	if csOpts.BatchSize != nil {
		changeStreamOptions = append(changeStreamOptions, bson.EC.Int32("batchSize", *csOpts.BatchSize))
	}
	if csOpts.Collation != nil {
		changeStreamOptions = append(changeStreamOptions, bson.EC.SubDocument("collation", csOpts.Collation.ToDocument()))
	}
	if csOpts.FullDocument != nil {
		changeStreamOptions = append(changeStreamOptions, bson.EC.String("fullDocument", string(*csOpts.FullDocument)))
	}
	if csOpts.MaxAwaitTimeMS != nil {
		aggOpts.MaxAwaitTimeMS = csOpts.MaxAwaitTimeMS
	}
	if csOpts.ResumeAfter != nil {
		changeStreamOptions = append(changeStreamOptions, bson.EC.SubDocument("resumeAfter", csOpts.ResumeAfter))
	}

	pipelineArr.Prepend(
		bson.VC.Document(
			bson.NewDocument(
				bson.EC.SubDocument("$changeStream", bson.NewDocument(changeStreamOptions...)))))

	cursor, err := coll.Aggregate(ctx, pipelineArr, aggOpts)
	if err != nil {
		return nil, err
	}

	cs := &changeStream{
		pipeline: pipelineArr,
		options:  changeStreamOptions,
		coll:     coll,
		cursor:   cursor,
		session:  sess,
		clock:    coll.client.clock,
	}

	return cs, nil
}

func (cs *changeStream) ID() int64 {
	return cs.cursor.ID()
}

func (cs *changeStream) Next(ctx context.Context) bool {
	if cs.cursor.Next(ctx) {
		return true
	}

	err := cs.cursor.Err()
	if err == nil {
		return false
	}

	switch t := err.(type) {
	case command.Error:
		if t.Code != errorCodeNotMaster && t.Code != errorCodeCursorNotFound {
			return false
		}
	}

	found := false

	for i, opt := range cs.options {
		if opt.Key() == "resumeAfter" {
			cs.options[i] = bson.EC.SubDocument("resumeAfter", cs.resumeToken)
			found = true
			break
		}
	}

	if !found {
		cs.options = append(cs.options, bson.EC.SubDocument("resumeAfter", cs.resumeToken))
	}

	oldns := cs.coll.namespace()
	killCursors := command.KillCursors{
		NS:  command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		IDs: []int64{cs.ID()},
	}

	ss, err := cs.coll.client.topology.SelectServer(ctx, cs.coll.readSelector)
	if err != nil {
		cs.err = err
		return false
	}

	conn, err := ss.Connection(ctx)
	if err != nil {
		cs.err = err
		return false
	}
	defer conn.Close()

	_, _ = killCursors.RoundTrip(ctx, ss.Description(), conn)

	cs.pipeline.Set(0, bson.VC.Document(
		bson.NewDocument(
			bson.EC.SubDocument("$changeStream", bson.NewDocument(cs.options...))),
	),
	)

	oldns = cs.coll.namespace()
	aggCmd := command.Aggregate{
		NS:       command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		Pipeline: cs.pipeline,
		Session:  cs.session,
		Clock:    cs.coll.client.clock,
	}

	cur, err := aggCmd.RoundTrip(ctx, ss.Description(), ss, conn)
	cs.cursor = cur
	cs.err = err

	if cs.err != nil {
		return false
	}

	return cs.cursor.Next(ctx)
}

func (cs *changeStream) Decode(out interface{}) error {
	br, err := cs.DecodeBytes()
	if err != nil {
		return err
	}

	return bsoncodec.UnmarshalWithRegistry(cs.coll.registry, br, out)
}

func (cs *changeStream) DecodeBytes() (bson.Reader, error) {
	br, err := cs.cursor.DecodeBytes()
	if err != nil {
		return nil, err
	}

	id, err := br.Lookup("_id")
	if err != nil {
		_ = cs.Close(context.Background())
		return nil, ErrMissingResumeToken
	}

	cs.resumeToken = id.Value().MutableDocument()

	return br, nil
}

func (cs *changeStream) Err() error {
	if cs.err != nil {
		return cs.err
	}

	return cs.cursor.Err()
}

func (cs *changeStream) Close(ctx context.Context) error {
	return cs.cursor.Close(ctx)
}
