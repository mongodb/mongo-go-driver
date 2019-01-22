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
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/x/bsonx/bsoncore"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver"
)

// Cursor is used to iterate a stream of documents. Each document is decoded into the result
// according to the rules of the bson package.
//
// A typical usage of the Cursor type would be:
//
//		var cur *Cursor
//		ctx := context.Background()
//		defer cur.Close(ctx)
//
// 		for cur.Next(ctx) {
//			elem := &bson.D{}
//			if err := cur.Decode(elem); err != nil {
// 				log.Fatal(err)
// 			}
//
// 			// do something with elem....
//		}
//
// 		if err := cur.Err(); err != nil {
//			log.Fatal(err)
//		}
//
type Cursor struct {
	bc       *driver.BatchCursor
	pos      int
	batch    []byte
	current  bsoncore.Document
	registry *bsoncodec.Registry

	isListCollections bool

	err error
}

func newCursor(bc *driver.BatchCursor, registry *bsoncodec.Registry) (*Cursor, error) {
	if registry == nil {
		registry = bson.DefaultRegistry
	}
	if bc == nil {
		return nil, errors.New("batch cursor must not be nil")
	}
	return &Cursor{bc: bc, pos: 0, batch: make([]byte, 0, 256), registry: registry}, nil
}

func newListCollectionsCursor(bc *driver.BatchCursor, registry *bsoncodec.Registry) (*Cursor, error) {
	c, err := newCursor(bc, registry)
	if err != nil {
		return nil, err
	}
	c.isListCollections = true
	return c, nil
}

func newEmptyCursor() *Cursor {
	return &Cursor{bc: driver.NewEmptyBatchCursor()}
}

// ID returns the ID of this cursor.
func (c *Cursor) ID() int64 { return c.bc.ID() }

// Next gets the next result from this cursor. Returns true if there were no errors and the next
// result is available for decoding.
func (c *Cursor) Next(ctx context.Context) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.pos < len(c.batch) {
		var ok bool
		c.current, _, ok = bsoncore.ReadDocument(c.batch[c.pos:])
		if !ok {
			c.err = errors.New("could not read next document as it is corrupt")
			return false
		}
		c.pos += len(c.current)
		if c.isListCollections {
			c.current, c.err = c.projectNameElement(c.current)
			if c.err != nil {
				return false
			}
		}
		return true
	}

	// clear the batch
	c.batch = c.batch[:0]
	c.pos = 0
	c.current = c.current[:0]

	// call the Next method in a loop until at least one document is returned in the next batch or
	// the context times out.
	for len(c.batch) == 0 {
		// If we don't have a next batch
		if !c.bc.Next(ctx) {
			// Do we have an error? If so we return false.
			c.err = c.bc.Err()
			if c.err != nil {
				return false
			}
			// Is the cursor ID zero?
			if c.bc.ID() == 0 {
				return false
			}
			// empty batch, but cursor is still valid, so continue.
			continue
		}

		c.batch = c.bc.Batch(c.batch[:0])
	}

	var ok bool
	c.current, _, ok = bsoncore.ReadDocument(c.batch[c.pos:])
	if !ok {
		c.err = errors.New("could not read next document as it is corrupt")
		return false
	}
	c.pos += len(c.current)
	if c.isListCollections {
		c.current, c.err = c.projectNameElement(c.current)
		if c.err != nil {
			return false
		}
	}

	return true
}

// Decode will decode the current document into val.
func (c *Cursor) Decode(val interface{}) error {
	return bson.UnmarshalWithRegistry(c.registry, c.current, val)
}

// DecodeBytes will return the raw bytes BSON representation of the current document. The document
// will be appended to dst.
func (c *Cursor) DecodeBytes(dst []byte) (bson.Raw, error) { return bson.Raw(c.current), nil }

// Err returns the current error.
func (c *Cursor) Err() error { return c.err }

// Close closes this cursor.
func (c *Cursor) Close(ctx context.Context) error { return c.bc.Close(ctx) }

// project out the database name for a legacy server
func (*Cursor) projectNameElement(rawDoc bsoncore.Document) (bsoncore.Document, error) {
	elems, err := rawDoc.Elements()
	if err != nil {
		return nil, err
	}

	var filteredElems []byte
	for _, elem := range elems {
		key := elem.Key()
		if key != "name" {
			filteredElems = append(filteredElems, elem...)
			continue
		}

		name := elem.Value().StringValue()
		collName := name[strings.Index(name, ".")+1:]
		filteredElems = bsoncore.AppendStringElement(filteredElems, "name", collName)
	}

	var filteredDoc []byte
	filteredDoc = bsoncore.BuildDocument(filteredDoc, filteredElems)
	return filteredDoc, nil
}
