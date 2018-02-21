// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/internal"
	"github.com/mongodb/mongo-go-driver/mongo/private/conn"
	"github.com/mongodb/mongo-go-driver/mongo/private/msg"
)

// NewExhaustedCursor creates a new exhausted cursor.
func NewExhaustedCursor() (Cursor, error) {
	return &exhaustedCursorImpl{}, nil
}

type exhaustedCursorImpl struct{}

func (e *exhaustedCursorImpl) ID() int64 {
	return -1
}

func (e *exhaustedCursorImpl) Next(context.Context) bool {
	return false
}

func (e *exhaustedCursorImpl) Decode(interface{}) error { return nil }

func (e *exhaustedCursorImpl) DecodeBytes() (bson.Reader, error) { return nil, nil }

func (e *exhaustedCursorImpl) Err() error {
	return nil
}

func (e *exhaustedCursorImpl) Close(_ context.Context) error {
	return nil
}

// NewCursor creates a new cursor from the given cursor result.
func NewCursor(cursorResult bson.Reader, batchSize int32, server Server) (Cursor, error) {
	cur, err := cursorResult.Lookup("cursor")
	if err != nil {
		return nil, err
	}
	if cur.Value().Type() != bson.TypeEmbeddedDocument {
		return nil, fmt.Errorf("cursor should be an embedded document but is a BSON %s", cur.Value().Type())
	}

	itr, err := cur.Value().ReaderDocument().Iterator()
	if err != nil {
		return nil, err
	}
	var elem *bson.Element
	impl := &cursorImpl{
		current:   -1,
		batchSize: batchSize,
		server:    server,
	}
	for itr.Next() {
		elem = itr.Element()
		switch elem.Key() {
		case "firstBatch":
			if elem.Value().Type() != bson.TypeArray {
				return nil, fmt.Errorf("firstBatch should be an array but is a BSON %s", elem.Value().Type())
			}
			impl.currentBatch = elem.Value().MutableArray()
		case "ns":
			if elem.Value().Type() != bson.TypeString {
				return nil, fmt.Errorf("namespace should be a string but is a BSON %s", elem.Value().Type())
			}
			namespace := ParseNamespace(elem.Value().StringValue())
			err = namespace.validate()
			if err != nil {
				return nil, err
			}
			impl.namespace = namespace
		case "id":
			if elem.Value().Type() != bson.TypeInt64 {
				return nil, fmt.Errorf("id should be an int64 but is a BSON %s", elem.Value().Type())
			}
			impl.cursorID = elem.Value().Int64()
		}
	}

	return impl, nil
}

// Cursor instances iterate a stream of documents. Each document is
// decoded into the result according to the rules of the bson package.
//
// A typical usage of the Cursor interface would be:
//
//		cursor := ...    // get a cursor from some operation
//		ctx := ...       // create a context for the operation
//		var doc bson.D
//		for cursor.Next(ctx, &doc) {
//			...
//		}
//		err := cursor.Close(ctx)
type Cursor interface {
	// NOTE: Whenever this changes, mongo.Cursor must be changed to match it.

	// Get the ID of the cursor.
	ID() int64

	// Get the next result from the cursor.
	// Returns true if there were no errors and there is a next result.
	Next(context.Context) bool

	Decode(interface{}) error

	DecodeBytes() (bson.Reader, error)

	// Returns the error status of the cursor
	Err() error

	// Close the cursor.  Ordinarily this is a no-op as the server closes the cursor when it is exhausted.
	// Returns the error status of this cursor so that clients do not have to call Err() separately
	Close(context.Context) error
}

type cursorImpl struct {
	namespace    Namespace
	batchSize    int32
	current      int
	currentBatch *bson.Array
	cursorID     int64
	err          error
	server       Server
}

func (c *cursorImpl) ID() int64 {
	return c.cursorID
}

func (c *cursorImpl) Next(ctx context.Context) bool {
	// TODO(skriptble): This is really messy, needs to be redesigned.
	c.current++
	if c.current < c.currentBatch.Len() {
		return true
	}

	c.getMore(ctx)
	if c.err != nil {
		return false
	}

	if c.currentBatch.Len() == 0 {
		return false
	}

	return true
}

func (c *cursorImpl) Decode(v interface{}) error {
	br, err := c.DecodeBytes()
	if err != nil {
		return err
	}
	return bson.NewDecoder(bytes.NewReader(br)).Decode(v)
}

func (c *cursorImpl) DecodeBytes() (bson.Reader, error) {
	br, err := c.currentBatch.Lookup(uint(c.current))
	if err != nil {
		return nil, err
	}
	if br.Type() != bson.TypeEmbeddedDocument {
		return nil, errors.New("Non-Document in batch of documents for cursor")
	}
	return br.ReaderDocument(), nil
}

func (c *cursorImpl) Err() error {
	return c.err
}

// TODO(GODRIVER-256): Only return the underlying Close error.
func (c *cursorImpl) Close(ctx context.Context) error {
	if c.cursorID == 0 {
		return c.err
	}

	killCursorsCommand := bson.NewDocument(
		bson.EC.String("killCursors", c.namespace.Collection),
		bson.EC.ArrayFromElements("cursors", bson.VC.Int64(c.cursorID)))

	killCursorsRequest := msg.NewCommand(
		msg.NextRequestID(),
		c.namespace.DB,
		false,
		killCursorsCommand,
	)

	connection, err := c.server.Connection(ctx)
	if err != nil {
		c.err = internal.MultiError(
			c.err,
			internal.WrapErrorf(err, "unable to get a connection to kill cursor %d", c.cursorID),
		)
		return c.err
	}

	_, err = conn.ExecuteCommand(ctx, connection, killCursorsRequest)
	if err != nil {
		c.err = internal.MultiError(
			c.err,
			internal.WrapErrorf(err, "unable to kill cursor %d", c.cursorID),
		)
		return c.err
	}

	c.cursorID = 0

	err = connection.Close()
	if err != nil {
		c.err = internal.MultiError(
			c.err,
			internal.WrapErrorf(err, "unable to close connection of cursor %d", c.cursorID),
		)
	}

	return c.err
}

func (c *cursorImpl) getMore(ctx context.Context) {
	c.currentBatch.Reset()
	c.current = 0

	if c.cursorID == 0 {
		return
	}

	getMoreCommand := bson.NewDocument(
		bson.EC.Int64("getMore", c.cursorID),
		bson.EC.String("collection", c.namespace.Collection))

	if c.batchSize != 0 {
		getMoreCommand.Append(bson.EC.Int32("batchSize", c.batchSize))
	}
	getMoreRequest := msg.NewCommand(
		msg.NextRequestID(),
		c.namespace.DB,
		false,
		getMoreCommand,
	)

	connection, err := c.server.Connection(ctx)
	if err != nil {
		c.err = internal.WrapErrorf(err, "unable to get a connection to get the next batch for cursor %d", c.cursorID)
		return
	}

	response, err := conn.ExecuteCommand(ctx, connection, getMoreRequest)
	if err != nil {
		c.err = internal.WrapErrorf(err, "unable get the next batch for cursor %d", c.cursorID)
		return
	}

	err = connection.Close()
	if err != nil {
		c.err = internal.WrapErrorf(err, "unable to close connection for cursor %d", c.cursorID)
		return
	}

	// cursor id
	idVal, err := response.Lookup("cursor", "id")
	if err != nil {
		c.err = internal.WrapError(err, "unable to find cursor ID in response")
		return
	}
	if idVal.Value().Type() != bson.TypeInt64 {
		err = fmt.Errorf("BSON Type %s is not %s", idVal.Value().Type(), bson.TypeInt64)
		c.err = internal.WrapErrorf(err, "cursorID of incorrect type in response")
		return
	}
	c.cursorID = idVal.Value().Int64()

	// cursor nextBatch
	nextBatchVal, err := response.Lookup("cursor", "nextBatch")
	if err != nil {
		c.err = internal.WrapError(err, "unable to find nextBatch in response")
		return
	}
	if nextBatchVal.Value().Type() != bson.TypeArray {
		err = fmt.Errorf("BSON Type %s is not %s", nextBatchVal.Value().Type(), bson.TypeArray)
		c.err = internal.WrapError(err, "nextBatch of incorrect type in response")
		return
	}
	c.currentBatch = nextBatchVal.Value().MutableArray()
}
