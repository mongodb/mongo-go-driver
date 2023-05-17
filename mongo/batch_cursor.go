// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
)

// batchCursor is the interface implemented by types that can provide batches of document results.
// The Cursor type is built on top of this type.
type batchCursor interface {
	// ID returns the ID of the cursor.
	ID() int64

	// Next returns true if there is a batch available.
	Next(context.Context) bool

	// Batch will return a DocumentSequence for the current batch of documents. The returned
	// DocumentSequence is only valid until the next call to Next or Close.
	Batch() *bsoncore.DocumentSequence

	// Server returns a pointer to the cursor's server.
	Server() driver.Server

	// Err returns the last error encountered.
	Err() error

	// Close closes the cursor.
	Close(context.Context) error

	// SetBatchSize is a modifier function used to adjust the batch size of
	// the cursor that implements it.
	SetBatchSize(int32)

	// SetMaxtimeMS will set the maximum number of milliseconds the server
	// will allow the operations to execute. This field cannot be sent if
	// the cursor was not configured with awaitData=true.
	SetMaxTimeMS(time.Duration)

	// SetComment will set a user-configurable comment that can be used to
	// identify the operation in logs.
	SetComment(interface{})

	// Tailable will set the cursor to "tail" or follow the target namespace
	// for new data. Querying a capped collection is one use case for a
	// tailable cursor
	Tailable(bool)

	// AwaitData should only be set alongside a tailable cursor. This is the
	// amount of time the server will wait for new data based on the
	// MaxTimeMS. If the time does expire an empty batch will be returned.
	AwaitData(bool)
}

// changeStreamCursor is the interface implemented by batch cursors that also provide the functionality for retrieving
// a postBatchResumeToken from commands and allows for the cursor to be killed rather than closed
type changeStreamCursor interface {
	batchCursor
	// PostBatchResumeToken returns the latest seen post batch resume token.
	PostBatchResumeToken() bsoncore.Document

	// KillCursor kills cursor on server without closing batch cursor
	KillCursor(context.Context) error
}
