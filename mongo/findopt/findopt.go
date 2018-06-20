// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package findopt

import (
	"time"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var (
	_ DeleteOne  = (*DeleteOneBundle)(nil)
	_ DeleteOne  = (*OptCollation)(nil)
	_ DeleteOne  = (*OptFields)(nil)
	_ DeleteOne  = (*OptMaxTime)(nil)
	_ DeleteOne  = (*OptProjection)(nil)
	_ DeleteOne  = (*OptSort)(nil)
	_ Find       = (*FindBundle)(nil)
	_ Find       = (*OptAllowPartialResults)(nil)
	_ Find       = (*OptBatchSize)(nil)
	_ Find       = (*OptCollation)(nil)
	_ Find       = (*OptComment)(nil)
	_ Find       = (*OptCursorType)(nil)
	_ Find       = (*OptHint)(nil)
	_ Find       = (*OptLimit)(nil)
	_ Find       = (*OptMax)(nil)
	_ Find       = (*OptMaxAwaitTime)(nil)
	_ Find       = (*OptMaxScan)(nil)
	_ Find       = (*OptMaxTime)(nil)
	_ Find       = (*OptMin)(nil)
	_ Find       = (*OptNoCursorTimeout)(nil)
	_ Find       = (*OptOplogReplay)(nil)
	_ Find       = (*OptProjection)(nil)
	_ Find       = (*OptReturnKey)(nil)
	_ Find       = (*OptShowRecordID)(nil)
	_ Find       = (*OptSkip)(nil)
	_ Find       = (*OptSnapshot)(nil)
	_ Find       = (*OptSort)(nil)
	_ One        = (*OneBundle)(nil)
	_ One        = (*OptAllowPartialResults)(nil)
	_ One        = (*OptBatchSize)(nil)
	_ One        = (*OptCollation)(nil)
	_ One        = (*OptComment)(nil)
	_ One        = (*OptCursorType)(nil)
	_ One        = (*OptHint)(nil)
	_ One        = (*OptMax)(nil)
	_ One        = (*OptMaxAwaitTime)(nil)
	_ One        = (*OptMaxScan)(nil)
	_ One        = (*OptMaxTime)(nil)
	_ One        = (*OptMin)(nil)
	_ One        = (*OptNoCursorTimeout)(nil)
	_ One        = (*OptOplogReplay)(nil)
	_ One        = (*OptProjection)(nil)
	_ One        = (*OptReturnKey)(nil)
	_ One        = (*OptShowRecordID)(nil)
	_ One        = (*OptSkip)(nil)
	_ One        = (*OptSnapshot)(nil)
	_ One        = (*OptSort)(nil)
	_ ReplaceOne = (*ReplaceOneBundle)(nil)
	_ ReplaceOne = (*OptBypassDocumentValidation)(nil)
	_ ReplaceOne = (*OptCollation)(nil)
	_ ReplaceOne = (*OptFields)(nil)
	_ ReplaceOne = (*OptMaxTime)(nil)
	_ ReplaceOne = (*OptProjection)(nil)
	_ ReplaceOne = (*OptReturnDocument)(nil)
	_ ReplaceOne = (*OptSort)(nil)
	_ ReplaceOne = (*OptUpsert)(nil)
	_ UpdateOne  = (*UpdateOneBundle)(nil)
	_ UpdateOne  = (*OptArrayFilters)(nil)
	_ UpdateOne  = (*OptBypassDocumentValidation)(nil)
	_ UpdateOne  = (*OptCollation)(nil)
	_ UpdateOne  = (*OptFields)(nil)
	_ UpdateOne  = (*OptMaxTime)(nil)
	_ UpdateOne  = (*OptProjection)(nil)
	_ UpdateOne  = (*OptReturnDocument)(nil)
	_ UpdateOne  = (*OptSort)(nil)
	_ UpdateOne  = (*OptUpsert)(nil)
)

// AllowPartialResults gets partial results if some shards are down.
// Find, One
func AllowPartialResults(b bool) OptAllowPartialResults {
	return OptAllowPartialResults(b)
}

// ArrayFilters specifies which array elements an update should apply.
// UpdateOne
func ArrayFilters(filters ...interface{}) OptArrayFilters {
	return OptArrayFilters(filters)
}

// BatchSize specifies the number of documents to return in each batch.
// Find, One
func BatchSize(i int32) OptBatchSize {
	return OptBatchSize(i)
}

// BypassDocumentValidation allows the write to opt-out of document-level validation.
// ReplaceOne, UpdateOne
func BypassDocumentValidation(b bool) OptBypassDocumentValidation {
	return OptBypassDocumentValidation(b)
}

// Collation specifies a Collation.
// Find, One, DeleteOne, ReplaceOne, UpdateOne
func Collation(collation *mongoopt.Collation) OptCollation {
	return OptCollation{
		Collation: collation.Convert(),
	}
}

// CursorType specifies the type of cursor to use.
// Find, One
func CursorType(ct mongoopt.CursorType) OptCursorType {
	return OptCursorType(ct)
}

// Comment specifies a string to help trace the operation through the database profiler, currentOp, and logs.
// Find, One
func Comment(s string) OptComment {
	return OptComment(s)
}

// Hint specifies which index to use.
// Find, One
func Hint(hint interface{}) OptHint {
	return OptHint{hint}
}

// Limit sets a limit on the number of results.
// Find
func Limit(i int64) OptLimit {
	return OptLimit(i)
}

// Max sets an exclusive upper bound for a specific index.
// Find, One
func Max(max interface{}) OptMax {
	return OptMax{max}
}

// MaxAwaitTime specifies the max amount of time for the server to wait on new documents.
// Find, One
func MaxAwaitTime(d time.Duration) OptMaxAwaitTime {
	return OptMaxAwaitTime(d)
}

// MaxScan specifies the maximum number of documents or index keys to scan.
// Find, One
func MaxScan(i int64) OptMaxScan {
	return OptMaxScan(i)
}

// MaxTime specifies the max time to allow the query to run.
// Find, One, DeleteOne, ReplaceOne, UpdateOne
func MaxTime(d time.Duration) OptMaxTime {
	return OptMaxTime(d)
}

// Min specifies the inclusive lower bound for a specific index.
// Find, One
func Min(min interface{}) OptMin {
	return OptMin{min}
}

// NoCursorTimeout prevents cursors from timing out after an inactivity period.
// Find, One
func NoCursorTimeout(b bool) OptNoCursorTimeout {
	return OptNoCursorTimeout(b)
}

// OplogReplay is for internal use and should not be set.
// Find, One
func OplogReplay(b bool) OptOplogReplay {
	return OptOplogReplay(b)
}

// Projection limits the fields returned for all documents.
// Find, One, DeleteOne, ReplaceOne, UpdateOne
func Projection(projection interface{}) OptProjection {
	return OptProjection{
		Projection: projection,
	}
}

// ReturnDocument specifies whether to return the updated or original document.
// ReplaceOne, UpdateOne
func ReturnDocument(rd mongoopt.ReturnDocument) OptReturnDocument {
	return OptReturnDocument(rd)
}

// ReturnKey specifies whether only index keys should be returned for results.
// Find, One
func ReturnKey(b bool) OptReturnKey {
	return OptReturnKey(b)
}

// ShowRecordID specifies whether to return the record identifier for each document.
// Find, One
func ShowRecordID(b bool) OptShowRecordID {
	return OptShowRecordID(b)
}

// Skip specifies the number of documents to skip before returning.
// Find, One
func Skip(i int64) OptSkip {
	return OptSkip(i)
}

// Snapshot prevents the cursor from returning a document more than once because of an
// intervening write operation.
// Find, One
func Snapshot(b bool) OptSnapshot {
	return OptSnapshot(b)
}

// Sort specifies the order in which to return results.
// Find, One, DeleteOne, ReplaceOne, UpdateOne
func Sort(sort interface{}) OptSort {
	return OptSort{sort}
}

// Upsert specifies whether a document should be inserted if no match is found.
// ReplaceOne, UpdateOne
func Upsert(b bool) OptUpsert {
	return OptUpsert(b)
}

// OptAllowPartialResults gets partial results if some shards are down.
type OptAllowPartialResults option.OptAllowPartialResults

func (OptAllowPartialResults) find() {}
func (OptAllowPartialResults) one()  {}

// ConvertFindOption implements the Find interface.
func (opt OptAllowPartialResults) ConvertFindOption() option.FindOptioner {
	return option.OptAllowPartialResults(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptAllowPartialResults) ConvertFindOneOption() option.FindOptioner {
	return option.OptAllowPartialResults(opt)
}

// OptArrayFilters specifies which array elements an update should apply.
type OptArrayFilters option.OptArrayFilters

func (OptArrayFilters) updateOne() {}

// ConvertUpdateOneOption implements the UpdateOne interface.
func (opt OptArrayFilters) ConvertUpdateOneOption() option.FindOneAndUpdateOptioner {
	return option.OptArrayFilters(opt)
}

// OptBatchSize specifies the number of documents to return in each batch.
type OptBatchSize option.OptBatchSize

func (OptBatchSize) find() {}
func (OptBatchSize) one()  {}

// ConvertFindOption implements the Find interface.
func (opt OptBatchSize) ConvertFindOption() option.FindOptioner {
	return option.OptBatchSize(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptBatchSize) ConvertFindOneOption() option.FindOptioner {
	return option.OptBatchSize(opt)
}

// OptBypassDocumentValidation allows the write to opt-out of document-level validation.
type OptBypassDocumentValidation option.OptBypassDocumentValidation

func (OptBypassDocumentValidation) replaceOne() {}
func (OptBypassDocumentValidation) updateOne()  {}

// ConvertReplaceOneOption implements the ReplaceOne interface.
func (opt OptBypassDocumentValidation) ConvertReplaceOneOption() option.FindOneAndReplaceOptioner {
	return option.OptBypassDocumentValidation(opt)
}

// ConvertUpdateOneOption implements the UpdateOne interface.
func (opt OptBypassDocumentValidation) ConvertUpdateOneOption() option.FindOneAndUpdateOptioner {
	return option.OptBypassDocumentValidation(opt)
}

// OptCollation specifies a Collation.
type OptCollation option.OptCollation

func (OptCollation) find()       {}
func (OptCollation) one()        {}
func (OptCollation) deleteOne()  {}
func (OptCollation) replaceOne() {}
func (OptCollation) updateOne()  {}

// ConvertFindOption implements the Find interface.
func (opt OptCollation) ConvertFindOption() option.FindOptioner {
	return option.OptCollation(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptCollation) ConvertFindOneOption() option.FindOptioner {
	return option.OptCollation(opt)
}

// ConvertDeleteOneOption implements the DeleteOne interface.
func (opt OptCollation) ConvertDeleteOneOption() option.FindOneAndDeleteOptioner {
	return option.OptCollation(opt)
}

// ConvertReplaceOneOption implements the ReplaceOne interface.
func (opt OptCollation) ConvertReplaceOneOption() option.FindOneAndReplaceOptioner {
	return option.OptCollation(opt)
}

// ConvertUpdateOneOption implements the UpdateOne interface.
func (opt OptCollation) ConvertUpdateOneOption() option.FindOneAndUpdateOptioner {
	return option.OptCollation(opt)
}

// OptCursorType specifies the type of cursor to use.
type OptCursorType option.OptCursorType

func (OptCursorType) find() {}
func (OptCursorType) one()  {}

// ConvertFindOption implements the Find interface.
func (opt OptCursorType) ConvertFindOption() option.FindOptioner {
	return option.OptCursorType(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptCursorType) ConvertFindOneOption() option.FindOptioner {
	return option.OptCursorType(opt)
}

// OptComment specifies a string to help trace the operation through the database profiler, currentOp, and logs.
type OptComment option.OptComment

func (OptComment) find() {}
func (OptComment) one()  {}

// ConvertFindOption implements the Find interface.
func (opt OptComment) ConvertFindOption() option.FindOptioner {
	return option.OptComment(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptComment) ConvertFindOneOption() option.FindOptioner {
	return option.OptComment(opt)
}

// OptFields limits the fields returned for find/modify commands.
type OptFields option.OptFields

func (OptFields) deleteOne()  {}
func (OptFields) updateOne()  {}
func (OptFields) replaceOne() {}

// ConvertDeleteOneOption implements the DeleteOne interface.
func (opt OptFields) ConvertDeleteOneOption() option.FindOneAndDeleteOptioner {
	return option.OptFields(opt)
}

// ConvertReplaceOneOption implements the ReplaceOne interface.
func (opt OptFields) ConvertReplaceOneOption() option.FindOneAndReplaceOptioner {
	return option.OptFields(opt)
}

// ConvertUpdateOneOption implements the UpdateOne interface.
func (opt OptFields) ConvertUpdateOneOption() option.FindOneAndUpdateOptioner {
	return option.OptFields(opt)
}

// OptHint specifies which index to use.
type OptHint option.OptHint

func (OptHint) find() {}
func (OptHint) one()  {}

// ConvertFindOption implements the Find interface.
func (opt OptHint) ConvertFindOption() option.FindOptioner {
	return option.OptHint(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptHint) ConvertFindOneOption() option.FindOptioner {
	return option.OptHint(opt)
}

// OptLimit sets a limit on the number of results.
type OptLimit option.OptLimit

func (OptLimit) find() {}

// ConvertFindOption implements the Find interface.
func (opt OptLimit) ConvertFindOption() option.FindOptioner {
	return option.OptLimit(opt)
}

// OptMax sets an exclusive upper bound for a specific index.
type OptMax option.OptMax

func (OptMax) find() {}
func (OptMax) one()  {}

// ConvertFindOption implements the Find interface.
func (opt OptMax) ConvertFindOption() option.FindOptioner {
	return option.OptMax(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptMax) ConvertFindOneOption() option.FindOptioner {
	return option.OptMax(opt)
}

// OptMaxAwaitTime specifies the max amount of time for the server to wait on new documents.
type OptMaxAwaitTime option.OptMaxAwaitTime

func (OptMaxAwaitTime) find() {}
func (OptMaxAwaitTime) one()  {}

// ConvertFindOption implements the Find interface.
func (opt OptMaxAwaitTime) ConvertFindOption() option.FindOptioner {
	return option.OptMaxAwaitTime(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptMaxAwaitTime) ConvertFindOneOption() option.FindOptioner {
	return option.OptMaxAwaitTime(opt)
}

// OptMaxScan specifies the maximum number of documents or index keys to scan.
type OptMaxScan option.OptMaxScan

func (OptMaxScan) find() {}
func (OptMaxScan) one()  {}

// ConvertFindOption implements the Find interface.
func (opt OptMaxScan) ConvertFindOption() option.FindOptioner {
	return option.OptMaxScan(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptMaxScan) ConvertFindOneOption() option.FindOptioner {
	return option.OptMaxScan(opt)
}

// OptMaxTime specifies the max time to allow the query to run.
type OptMaxTime option.OptMaxTime

func (OptMaxTime) find()       {}
func (OptMaxTime) one()        {}
func (OptMaxTime) deleteOne()  {}
func (OptMaxTime) replaceOne() {}
func (OptMaxTime) updateOne()  {}

// ConvertFindOption implements the Find interface.
func (opt OptMaxTime) ConvertFindOption() option.FindOptioner {
	return option.OptMaxTime(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptMaxTime) ConvertFindOneOption() option.FindOptioner {
	return option.OptMaxTime(opt)
}

// ConvertDeleteOneOption implements the DeleteOne interface.
func (opt OptMaxTime) ConvertDeleteOneOption() option.FindOneAndDeleteOptioner {
	return option.OptMaxTime(opt)
}

// ConvertReplaceOneOption implements the ReplaceOne interface.
func (opt OptMaxTime) ConvertReplaceOneOption() option.FindOneAndReplaceOptioner {
	return option.OptMaxTime(opt)
}

// ConvertUpdateOneOption implements the UpdateOne interface.
func (opt OptMaxTime) ConvertUpdateOneOption() option.FindOneAndUpdateOptioner {
	return option.OptMaxTime(opt)
}

// OptMin specifies the inclusive lower bound for a specific index.
type OptMin option.OptMin

func (OptMin) find() {}
func (OptMin) one()  {}

// ConvertFindOption implements the Find interface.
func (opt OptMin) ConvertFindOption() option.FindOptioner {
	return option.OptMin(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptMin) ConvertFindOneOption() option.FindOptioner {
	return option.OptMin(opt)
}

// OptNoCursorTimeout prevents cursors from timing out after an inactivity period.
type OptNoCursorTimeout option.OptNoCursorTimeout

func (OptNoCursorTimeout) find() {}
func (OptNoCursorTimeout) one()  {}

// ConvertFindOption implements the Find interface.
func (opt OptNoCursorTimeout) ConvertFindOption() option.FindOptioner {
	return option.OptNoCursorTimeout(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptNoCursorTimeout) ConvertFindOneOption() option.FindOptioner {
	return option.OptNoCursorTimeout(opt)
}

// OptOplogReplay is for internal use and should not be set.
type OptOplogReplay option.OptOplogReplay

func (OptOplogReplay) find() {}
func (OptOplogReplay) one()  {}

// ConvertFindOption implements the Find interface.
func (opt OptOplogReplay) ConvertFindOption() option.FindOptioner {
	return option.OptOplogReplay(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptOplogReplay) ConvertFindOneOption() option.FindOptioner {
	return option.OptOplogReplay(opt)
}

// OptProjection limits the fields returned for all documents.
type OptProjection option.OptProjection

func (OptProjection) find()       {}
func (OptProjection) one()        {}
func (OptProjection) deleteOne()  {}
func (OptProjection) replaceOne() {}
func (OptProjection) updateOne()  {}

// ConvertFindOption implements the Find interface.
func (opt OptProjection) ConvertFindOption() option.FindOptioner {
	return option.OptProjection(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptProjection) ConvertFindOneOption() option.FindOptioner {
	return option.OptProjection(opt)
}

// ConvertDeleteOneOption implements the DeleteOne interface.
func (opt OptProjection) ConvertDeleteOneOption() option.FindOneAndDeleteOptioner {
	return option.OptFields{
		Fields: opt.Projection,
	}
}

// ConvertReplaceOneOption option implements the ReplaceOne interface.
func (opt OptProjection) ConvertReplaceOneOption() option.FindOneAndReplaceOptioner {
	return option.OptFields{
		Fields: opt.Projection,
	}
}

// ConvertUpdateOneOption option implements the UpdateOne interface.
func (opt OptProjection) ConvertUpdateOneOption() option.FindOneAndUpdateOptioner {
	return option.OptFields{
		Fields: opt.Projection,
	}
}

// OptReturnDocument specifies whether to return the updated or original document.
type OptReturnDocument option.OptReturnDocument

func (OptReturnDocument) replaceOne() {}
func (OptReturnDocument) updateOne()  {}

// ConvertReplaceOneOption implements the ReplaceOne interface.
func (opt OptReturnDocument) ConvertReplaceOneOption() option.FindOneAndReplaceOptioner {
	return option.OptReturnDocument(opt)
}

// ConvertUpdateOneOption implements the UpdateOne interface.
func (opt OptReturnDocument) ConvertUpdateOneOption() option.FindOneAndUpdateOptioner {
	return option.OptReturnDocument(opt)
}

// OptReturnKey specifies whether only index keys should be returned for results.
type OptReturnKey option.OptReturnKey

func (OptReturnKey) find() {}
func (OptReturnKey) one()  {}

// ConvertFindOption implements the Find interface.
func (opt OptReturnKey) ConvertFindOption() option.FindOptioner {
	return option.OptReturnKey(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptReturnKey) ConvertFindOneOption() option.FindOptioner {
	return option.OptReturnKey(opt)
}

// OptShowRecordID specifies whether to return the record identifier for each document.
type OptShowRecordID option.OptShowRecordID

func (OptShowRecordID) find() {}
func (OptShowRecordID) one()  {}

// ConvertFindOption implements the Find interface.
func (opt OptShowRecordID) ConvertFindOption() option.FindOptioner {
	return option.OptShowRecordID(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptShowRecordID) ConvertFindOneOption() option.FindOptioner {
	return option.OptShowRecordID(opt)
}

// OptSkip specifies the number of documents to skip before returning.
type OptSkip option.OptSkip

func (OptSkip) find() {}
func (OptSkip) one()  {}

// ConvertFindOption implements the Find interface.
func (opt OptSkip) ConvertFindOption() option.FindOptioner {
	return option.OptSkip(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptSkip) ConvertFindOneOption() option.FindOptioner {
	return option.OptSkip(opt)
}

// OptSnapshot prevents the cursor from returning a document more than once because of an
// intervening write operation.
type OptSnapshot option.OptSnapshot

func (OptSnapshot) find() {}
func (OptSnapshot) one()  {}

// ConvertFindOption implements the Find interface.
func (opt OptSnapshot) ConvertFindOption() option.FindOptioner {
	return option.OptSnapshot(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptSnapshot) ConvertFindOneOption() option.FindOptioner {
	return option.OptSnapshot(opt)
}

// OptSort specifies the order in which to return results.
type OptSort option.OptSort

func (OptSort) find()       {}
func (OptSort) one()        {}
func (OptSort) deleteOne()  {}
func (OptSort) replaceOne() {}
func (OptSort) updateOne()  {}

// ConvertFindOption implements the Find interface.
func (opt OptSort) ConvertFindOption() option.FindOptioner {
	return option.OptSort(opt)
}

// ConvertFindOneOption implements the One interface.
func (opt OptSort) ConvertFindOneOption() option.FindOptioner {
	return option.OptSort(opt)
}

// ConvertDeleteOneOption implements the DeleteOne interface.
func (opt OptSort) ConvertDeleteOneOption() option.FindOneAndDeleteOptioner {
	return option.OptSort(opt)
}

// ConvertReplaceOneOption implements the ReplaceOne interface.
func (opt OptSort) ConvertReplaceOneOption() option.FindOneAndReplaceOptioner {
	return option.OptSort(opt)
}

// ConvertUpdateOneOption implements the UpdateOne interface.
func (opt OptSort) ConvertUpdateOneOption() option.FindOneAndUpdateOptioner {
	return option.OptSort(opt)
}

// OptUpsert specifies whether a document should be inserted if no match is found.
type OptUpsert option.OptUpsert

func (OptUpsert) replaceOne() {}
func (OptUpsert) updateOne()  {}

// ConvertReplaceOneOption implements the ReplaceOne interface.
func (opt OptUpsert) ConvertReplaceOneOption() option.FindOneAndReplaceOptioner {
	return option.OptUpsert(opt)
}

// ConvertUpdateOneOption implements the UpdateOne interface.
func (opt OptUpsert) ConvertUpdateOneOption() option.FindOneAndUpdateOptioner {
	return option.OptUpsert(opt)
}
