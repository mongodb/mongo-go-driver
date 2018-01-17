// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

// // CursorResult describes the initial results for any operation that can establish a cursor.
// type CursorResult interface {
// 	// The namespace the cursor is in
// 	Namespace() Namespace
// 	// The initial batch of results, which may be empty
// 	InitialBatch() []bson.Raw
// 	// The cursor id, which may be zero if no cursor was established
// 	CursorID() int64
// }
//
// type cursorRequest struct {
// 	BatchSize int32 `bson:"batchSize,omitempty"`
// }
//
// // The result of a command that returns a cursor
// type cursorReturningResult struct {
// 	// The cursor
// 	Cursor firstBatchCursorResult `bson:"cursor"`
// }
//
// // The first batch of a cursor
// type firstBatchCursorResult struct {
// 	// The first batch of the cursor
// 	FirstBatch []bson.Raw `bson:"firstBatch"`
// 	// The namespace to use for iterating the cursor
// 	NS string `bson:"ns"`
// 	// The cursor id
// 	ID int64 `bson:"id"`
// }
//
// func (cursorResult *firstBatchCursorResult) Namespace() Namespace {
// 	// Assume server returns a valid namespace string
// 	namespace := ParseNamespace(cursorResult.NS)
// 	return namespace
// }
//
// func (cursorResult *firstBatchCursorResult) InitialBatch() []bson.Raw {
// 	return cursorResult.FirstBatch
// }
//
// func (cursorResult *firstBatchCursorResult) CursorID() int64 {
// 	return cursorResult.ID
// }
