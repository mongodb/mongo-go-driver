// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"errors"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver/session"
	"github.com/mongodb/mongo-go-driver/x/network/command"
	"github.com/mongodb/mongo-go-driver/x/network/wiremessage"
)

type cursor struct {
	clientSession *session.Client
	clock         *session.ClusterClock
	namespace     command.Namespace
	current       int
	batch         []bson.RawValue
	id            int64
	err           error
	server        *Server
	opts          []bsonx.Elem
	registry      *bsoncodec.Registry

	// legacy server (< 3.2) fields
	batchSize   int32
	limit       int32
	numReturned int32 // number of docs returned by server
}

func newCursor(result bson.Raw, clientSession *session.Client, clock *session.ClusterClock, server *Server, opts ...bsonx.Elem) (command.Cursor, error) {
	cur, err := result.LookupErr("cursor")
	if err != nil {
		return nil, err
	}
	if cur.Type != bson.TypeEmbeddedDocument {
		return nil, fmt.Errorf("cursor should be an embedded document but it is a BSON %s", cur.Type)
	}

	elems, err := cur.Document().Elements()
	if err != nil {
		return nil, err
	}
	c := &cursor{
		clientSession: clientSession,
		clock:         clock,
		current:       -1,
		server:        server,
		registry:      server.cfg.registry,
		opts:          opts,
	}

	var ok bool
	for _, elem := range elems {
		switch elem.Key() {
		case "firstBatch":
			var arr bson.Raw
			arr, ok = elem.Value().ArrayOK()
			if !ok {
				return nil, fmt.Errorf("firstBatch should be an array but it is a BSON %s", elem.Value().Type)
			}
			c.batch, err = arr.Values()
			if err != nil {
				return nil, err
			}
		case "ns":
			if elem.Value().Type != bson.TypeString {
				return nil, fmt.Errorf("namespace should be a string but it is a BSON %s", elem.Value().Type)
			}
			namespace := command.ParseNamespace(elem.Value().StringValue())
			err = namespace.Validate()
			if err != nil {
				return nil, err
			}
			c.namespace = namespace
		case "id":
			c.id, ok = elem.Value().Int64OK()
			if !ok {
				return nil, fmt.Errorf("id should be an int64 but it is a BSON %s", elem.Value().Type)
			}
		}
	}

	// close session if everything fits in first batch
	if c.id == 0 {
		c.closeImplicitSession()
	}
	return c, nil
}

func newLegacyCursor(ns command.Namespace, cursorID int64, batch []bson.Raw, limit int32, batchSize int32, server *Server) (command.Cursor, error) {
	c := &cursor{
		id:          cursorID,
		current:     -1,
		server:      server,
		registry:    server.cfg.registry,
		namespace:   ns,
		limit:       limit,
		batchSize:   batchSize,
		numReturned: int32(len(batch)),
	}

	// take as many documents from the batch as needed
	firstBatchSize := int32(len(batch))
	if limit != 0 && limit < firstBatchSize {
		firstBatchSize = limit
	}
	batch = batch[:firstBatchSize]
	for _, doc := range batch {
		c.batch = append(c.batch, bson.RawValue{
			Type:  bsontype.EmbeddedDocument,
			Value: doc,
		})
	}

	return c, nil
}

// close the associated session if it's implicit
func (c *cursor) closeImplicitSession() {
	if c.clientSession != nil && c.clientSession.SessionType == session.Implicit {
		c.clientSession.EndSession()
	}
}

func (c *cursor) ID() int64 {
	return c.id
}

// returns true if the cursor is for a server with version < 3.2
func (c *cursor) legacy() bool {
	return c.server.Description().WireVersion.Max < 4
}

func (c *cursor) Next(ctx context.Context) bool {
	if ctx == nil {
		ctx = context.Background()
	}

	c.current++
	if c.current < len(c.batch) {
		return true
	}

	if c.id == 0 {
		return false
	}

	if c.legacy() {
		c.legacyGetMore(ctx)
	} else {
		c.getMore(ctx)
	}

	// call the getMore command in a loop until at least one document is returned in the next batch
	for len(c.batch) == 0 {
		if c.err != nil || (c.id == 0 && len(c.batch) == 0) {
			return false
		}

		if c.legacy() {
			c.legacyGetMore(ctx)
		} else {
			c.getMore(ctx)
		}
	}

	return true
}

func (c *cursor) Decode(v interface{}) error {
	br, err := c.DecodeBytes()
	if err != nil {
		return err
	}

	return bson.UnmarshalWithRegistry(c.registry, br, v)
}

func (c *cursor) DecodeBytes() (bson.Raw, error) {
	br := c.batch[c.current]
	if br.Type != bson.TypeEmbeddedDocument {
		return nil, errors.New("Non-Document in batch of documents for cursor")
	}
	return br.Document(), nil
}

func (c *cursor) Err() error {
	return c.err
}

func (c *cursor) Close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if c.legacy() {
		return c.legacyKillCursor(ctx)
	}

	defer c.closeImplicitSession()
	conn, err := c.server.Connection(ctx)
	if err != nil {
		return err
	}

	_, err = (&command.KillCursors{
		Clock: c.clock,
		NS:    c.namespace,
		IDs:   []int64{c.id},
	}).RoundTrip(ctx, c.server.SelectedDescription(), conn)
	if err != nil {
		_ = conn.Close() // The command response error is more important here
		return err
	}

	c.id = 0
	return conn.Close()
}

// clear out the cursor's batch slice
func (c *cursor) clearBatch() {
	for idx := range c.batch {
		c.batch[idx].Type = bsontype.Type(0)
		c.batch[idx].Value = nil
	}

	c.batch = c.batch[:0]
	c.current = 0
}

func (c *cursor) legacyKillCursor(ctx context.Context) error {
	conn, err := c.server.Connection(ctx)
	if err != nil {
		return err
	}

	kc := wiremessage.KillCursors{
		NumberOfCursorIDs: 1,
		CursorIDs:         []int64{c.id},
		CollectionName:    c.namespace.Collection,
		DatabaseName:      c.namespace.DB,
	}

	err = conn.WriteWireMessage(ctx, kc)
	if err != nil {
		_ = conn.Close()
		return err
	}

	err = conn.Close() // no reply from OP_KILL_CURSORS
	if err != nil {
		return err
	}

	c.id = 0
	c.clearBatch()
	return nil
}

func (c *cursor) legacyGetMore(ctx context.Context) {
	c.clearBatch()
	if c.id == 0 {
		return
	}

	conn, err := c.server.Connection(ctx)
	if err != nil {
		c.err = err
		return
	}

	numToReturn := c.batchSize
	if c.limit != 0 && c.numReturned+c.batchSize > c.limit {
		numToReturn = c.limit - c.numReturned
	}
	gm := wiremessage.GetMore{
		FullCollectionName: c.namespace.DB + "." + c.namespace.Collection,
		CursorID:           c.id,
		NumberToReturn:     numToReturn,
	}

	err = conn.WriteWireMessage(ctx, gm)
	if err != nil {
		_ = conn.Close()
		c.err = err
		return
	}

	response, err := conn.ReadWireMessage(ctx)
	if err != nil {
		_ = conn.Close()
		c.err = err
		return
	}

	err = conn.Close()
	if err != nil {
		c.err = err
		return
	}

	reply, ok := response.(wiremessage.Reply)
	if !ok {
		c.err = errors.New("did not receive OP_REPLY response")
		return
	}

	err = validateGetMoreReply(reply)
	if err != nil {
		c.err = err
		return
	}

	c.id = reply.CursorID
	c.numReturned += reply.NumberReturned
	numDocs := reply.NumberReturned // number of docs to put into the batch
	if c.limit != 0 && c.numReturned >= c.limit {
		numDocs = reply.NumberReturned - (c.numReturned - c.limit)
		err = c.Close(ctx)
		if err != nil {
			c.err = err
			return
		}
	}

	var i int32
	for i = 0; i < numDocs; i++ {
		c.batch = append(c.batch, bson.RawValue{
			Type:  bsontype.EmbeddedDocument,
			Value: reply.Documents[i],
		})
	}
}

func (c *cursor) getMore(ctx context.Context) {
	c.clearBatch()
	if c.id == 0 {
		return
	}

	conn, err := c.server.Connection(ctx)
	if err != nil {
		c.err = err
		return
	}

	response, err := (&command.GetMore{
		Clock:   c.clock,
		ID:      c.id,
		NS:      c.namespace,
		Opts:    c.opts,
		Session: c.clientSession,
	}).RoundTrip(ctx, c.server.SelectedDescription(), conn)
	if err != nil {
		_ = conn.Close() // The command response error is more important here
		c.err = err
		return
	}

	err = conn.Close()
	if err != nil {
		c.err = err
		return
	}

	id, err := response.LookupErr("cursor", "id")
	if err != nil {
		c.err = err
		return
	}
	var ok bool
	c.id, ok = id.Int64OK()
	if !ok {
		c.err = fmt.Errorf("BSON Type %s is not %s", id.Type, bson.TypeInt64)
		return
	}

	// if this is the last getMore, close the session
	if c.id == 0 {
		c.closeImplicitSession()
	}

	batch, err := response.LookupErr("cursor", "nextBatch")
	if err != nil {
		c.err = err
		return
	}
	var arr bson.Raw
	arr, ok = batch.ArrayOK()
	if !ok {
		c.err = fmt.Errorf("BSON Type %s is not %s", batch.Type, bson.TypeArray)
		return
	}
	c.batch, c.err = arr.Values()

	return
}

func validateGetMoreReply(reply wiremessage.Reply) error {
	if int(reply.NumberReturned) != len(reply.Documents) {
		return command.NewCommandResponseError("malformed OP_REPLY: NumberReturned does not match number of returned documents", nil)
	}

	if reply.ResponseFlags&wiremessage.CursorNotFound == wiremessage.CursorNotFound {
		return command.QueryFailureError{
			Message: "query failure - cursor not found",
		}
	}
	if reply.ResponseFlags&wiremessage.QueryFailure == wiremessage.QueryFailure {
		return command.QueryFailureError{
			Message:  "query failure",
			Response: reply.Documents[0],
		}
	}

	return nil
}
