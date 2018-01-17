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

	"github.com/10gen/mongo-go-driver/mongo/internal"
	"github.com/10gen/mongo-go-driver/mongo/private/conn"
	"github.com/10gen/mongo-go-driver/mongo/private/msg"
	"github.com/skriptble/wilson/bson"
)

// NewExhaustedCursor creates a new exhausted cursor.
func NewExhaustedCursor() (Cursor, error) {
	return &exhaustedCursorImpl{}, nil
}

type exhaustedCursorImpl struct{}

func (e *exhaustedCursorImpl) Next(_ context.Context, _ interface{}) bool {
	return false
}

func (e *exhaustedCursorImpl) Decode(interface{}) error { return nil }

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
			impl.currentBatch2 = elem.Value().MutableArray()
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
	// Get the next result from the cursor.
	// Returns true if there were no errors and there is a next result.
	Next(context.Context, interface{}) bool

	Decode(interface{}) error

	// Returns the error status of the cursor
	Err() error

	// Close the cursor.  Ordinarily this is a no-op as the server closes the cursor when it is exhausted.
	// Returns the error status of this cursor so that clients do not have to call Err() separately
	Close(context.Context) error
}

type cursorImpl struct {
	namespace     Namespace
	batchSize     int32
	current       int
	currentBatch2 *bson.Array
	cursorID      int64
	err           error
	server        Server
}

func (c *cursorImpl) Next(ctx context.Context, result interface{}) bool {
	// TODO(skriptble): This is really messy, needs to be redesigned.
	c.current++
	if c.current < c.currentBatch2.Len() {
		if result != nil {
			c.err = c.Decode(result)
			if c.err != nil {
				return false
			}
		}
		return true
	}

	c.getMore(ctx)
	if c.err != nil {
		return false
	}

	if c.currentBatch2.Len() == 0 {
		return false
	}

	if result != nil {
		c.err = c.Decode(result)
		if c.err != nil {
			return false
		}
	}

	return true
}

func (c *cursorImpl) Decode(v interface{}) error {
	br, err := c.currentBatch2.Lookup(uint(c.current))
	if err != nil {
		return err
	}
	if br.Type() != bson.TypeEmbeddedDocument {
		return errors.New("Non-Document in batch of documents for cursor")
	}
	dec := bson.NewDecoder(bytes.NewReader(br.ReaderDocument()))
	err = dec.Decode(v)
	return err
}

func (c *cursorImpl) Err() error {
	return c.err
}

func (c *cursorImpl) Close(ctx context.Context) error {
	if c.cursorID == 0 {
		return c.err
	}

	killCursorsCommand := bson.NewDocument(2).Append(
		bson.C.String("killCursors", c.namespace.Collection),
		bson.C.ArrayFromElements("cursors", bson.AC.Int64(c.cursorID)),
	)

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

	_, err = conn.ExecuteCommand(ctx, connection, killCursorsRequest, nil)
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
	c.currentBatch2.Reset()
	c.current = 0

	if c.cursorID == 0 {
		return
	}

	getMoreCommand := bson.NewDocument(3).Append(
		bson.C.Int64("getMore", c.cursorID),
		bson.C.String("collection", c.namespace.Collection),
	)
	if c.batchSize != 0 {
		getMoreCommand.Append(bson.C.Int32("batchSize", c.batchSize))
	}
	getMoreRequest := msg.NewCommand(
		msg.NextRequestID(),
		c.namespace.DB,
		false,
		getMoreCommand,
	)

	// var response struct {
	// 	OK     bool `bson:"ok"`
	// 	Cursor struct {
	// 		NextBatch []oldbson.Raw `bson:"nextBatch"`
	// 		NS        string        `bson:"ns"`
	// 		ID        int64         `bson:"id"`
	// 	} `bson:"cursor"`
	// }

	connection, err := c.server.Connection(ctx)
	if err != nil {
		c.err = internal.WrapErrorf(err, "unable to get a connection to get the next batch for cursor %d", c.cursorID)
		return
	}

	response, err := conn.ExecuteCommand(ctx, connection, getMoreRequest, nil)
	if err != nil {
		c.err = internal.WrapErrorf(err, "unable get the next batch for cursor %d", c.cursorID)
		return
	}

	err = connection.Close()
	if err != nil {
		c.err = internal.WrapErrorf(err, "unable to close connection for cursor %d", c.cursorID)
		return
	}

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
	c.currentBatch2 = nextBatchVal.Value().MutableArray()
}
