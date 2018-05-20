// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/option"
)

// Cursor instances iterate a stream of documents. Each document is
// decoded into the result according to the rules of the bson package.
//
// A typical usage of the Cursor interface would be:
//
//		var cur Cursor
//		ctx := context.Background()
//		defer cur.Close(ctx)
//
// 		for cur.Next(ctx) {
//			elem := bson.NewDocument()
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
type Cursor interface {
	// Get the ID of the cursor.
	ID() int64

	// Get the next result from the cursor.
	// Returns true if there were no errors and there is a next result.
	Next(context.Context) bool

	// Decode the next document into the provided object according to the
	// rules of the bson package.
	Decode(interface{}) error

	// Returns the next document as a bson.Reader. The user must copy the
	// bytes to retain them.
	DecodeBytes() (bson.Reader, error)

	// Returns the error status of the cursor
	Err() error

	// Close the cursor.
	Close(context.Context) error
}

// CursorBuilder is a type that can build a Cursor.
type CursorBuilder interface {
	BuildCursor(bson.Reader, ...option.CursorOptioner) (Cursor, error)
}

type emptyCursor struct{}

func (ec emptyCursor) ID() int64                         { return -1 }
func (ec emptyCursor) Next(context.Context) bool         { return false }
func (ec emptyCursor) Decode(interface{}) error          { return nil }
func (ec emptyCursor) DecodeBytes() (bson.Reader, error) { return nil, nil }
func (ec emptyCursor) Err() error                        { return nil }
func (ec emptyCursor) Close(context.Context) error       { return nil }
