// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"time"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/mongo/options"
)

// In order to facilitate users not having to supply a default value (e.g. nil or an uninitialized
// struct) when not using any options for an operation, options are defined as functions that take
// the necessary state and return a private type which implements an interface for the given
// operation. This API will allow users to use the following syntax for operations:
//
// coll.UpdateOne(filter, update)
// coll.UpdateOne(filter, update, mongo.Upsert(true))
// coll.UpdateOne(filter, update, mongo.Upsert(true), mongo.bypassDocumentValidation(false))
//
// To see which options are available on each operation, see the following files:
//
// - delete_options.go
// - insert_options.go
// - update_options.go

// AllowDiskUse enables writing to temporary files.
func AllowDiskUse(b bool) options.OptAllowDiskUse {
	opt := options.OptAllowDiskUse(b)
	return opt
}

// AllowPartialResults gets partial results from a mongos if some shards are down (instead of
// throwing an error).
func AllowPartialResults(b bool) options.OptAllowPartialResults {
	opt := options.OptAllowPartialResults(b)
	return opt
}

// ArrayFilters specifies to which array elements an update should apply.
//
// This function uses TransformDocument to turn the document parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// document.
func ArrayFilters(filters ...interface{}) (options.OptArrayFilters, error) {
	docs := make([]*bson.Document, 0, len(filters))
	for _, f := range filters {
		d, err := TransformDocument(f)
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	opt := options.OptArrayFilters(docs)
	return opt, nil
}

// BatchSize specifies the number of documents to return per batch.
func BatchSize(i int32) options.OptBatchSize {
	opt := options.OptBatchSize(i)
	return opt
}

// BypassDocumentValidation is used to opt out of document-level validation for a given write.
func BypassDocumentValidation(b bool) options.OptBypassDocumentValidation {
	opt := options.OptBypassDocumentValidation(b)
	return opt
}

// Collation allows users to specify language-specific rules for string comparison, such as rules
// for lettercase and accent marks.
//
// See https://docs.mongodb.com/manual/reference/collation/.
func Collation(collation *options.CollationOptions) options.OptCollation {
	opt := options.OptCollation{Collation: collation}
	return opt
}

// Comment specifies a comment to attach to the query to help attach and trace profile data.
//
// See https://docs.mongodb.com/manual/reference/command/profile.
func Comment(s string) options.OptComment {
	opt := options.OptComment(s)
	return opt
}

// CursorType indicates the type of cursor to use.
func CursorType(cursorType options.CursorType) options.OptCursorType {
	opt := options.OptCursorType(cursorType)
	return opt
}

// Hint specifies the index to use.
//
// The hint parameter must be either a string or a document. If it is a
// document, this function uses TransformDocument to turn the document into a
// *bson.Document. See TransformDocument for the list of valid types.
func Hint(hint interface{}) (options.OptHint, error) {
	switch hint.(type) {
	case string:
	default:
		var err error
		hint, err = TransformDocument(hint)
		if err != nil {
			return options.OptHint{}, err
		}
	}
	opt := options.OptHint{Hint: hint}
	return opt, nil
}

// Limit specifies the maximum number of documents to return.
func Limit(i int64) options.OptLimit {
	opt := options.OptLimit(i)
	return opt
}

// Max specifies the exclusive upper bound for a specific index.
//
// This function uses TransformDocument to turn the max parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// max.
func Max(max interface{}) (options.OptMax, error) {
	doc, err := TransformDocument(max)
	if err != nil {
		return options.OptMax{}, err
	}
	opt := options.OptMax{Max: doc}
	return opt, nil
}

// MaxAwaitTime specifies the maximum amount of time for the server to wait on new documents to
// satisfy a tailable-await cursor query.
func MaxAwaitTime(duration time.Duration) options.OptMaxAwaitTime {
	opt := options.OptMaxAwaitTime(duration)
	return opt
}

// MaxScan specifies the maximum number of documents or index keys to scan when executing the query.
func MaxScan(i int64) options.OptMaxScan {
	opt := options.OptMaxScan(i)
	return opt
}

// MaxTime specifies the maximum amount of time to allow the query to run.
func MaxTime(duration time.Duration) options.OptMaxTime {
	opt := options.OptMaxTime(duration)
	return opt
}

// Min specifies the inclusive lower bound for a specific index.
//
// This function uses TransformDocument to turn the min parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// min.
func Min(min interface{}) (options.OptMin, error) {
	doc, err := TransformDocument(min)
	if err != nil {
		return options.OptMin{}, err
	}
	opt := options.OptMin{Min: doc}
	return opt, nil
}

// NoCursorTimeout specifies whether to prevent the server from timing out idle cursors after an
// inactivity period.
func NoCursorTimeout(b bool) options.OptNoCursorTimeout {
	opt := options.OptNoCursorTimeout(b)
	return opt
}

// OplogReplay is for internal replication use only.
func OplogReplay(b bool) options.OptOplogReplay {
	opt := options.OptOplogReplay(b)
	return opt
}

// Ordered specifies whether the remaining writes should be aborted if one of the earlier ones fails.
func Ordered(b bool) options.OptOrdered {
	opt := options.OptOrdered(b)
	return opt
}

// Projection limits the fields to return for all matching documents.
//
// This function uses TransformDocument to turn the projection parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// projection.
func Projection(projection interface{}) (options.OptProjection, error) {
	doc, err := TransformDocument(projection)
	if err != nil {
		return options.OptProjection{}, nil
	}
	opt := options.OptProjection{Projection: doc}
	return opt, nil
}

// ReturnDocument specifies whether a findAndUpdate should return the document as it was before the
// update or as it is after.
func ReturnDocument(returnDocument options.ReturnDocument) options.OptReturnDocument {
	opt := options.OptReturnDocument(returnDocument)
	return opt
}

// ReturnKey specifies whether to only return the index keys in the resulting documents.
func ReturnKey(b bool) options.OptReturnKey {
	opt := options.OptReturnKey(b)
	return opt
}

// ShowRecordID determines whether to return the record identifier for each document.
func ShowRecordID(b bool) options.OptShowRecordID {
	opt := options.OptShowRecordID(b)
	return opt
}

// Skip specifies the number of documents to skip before returning.
func Skip(i int64) options.OptSkip {
	opt := options.OptSkip(i)
	return opt
}

// Snapshot specifies whether to prevent the cursor from returning a document more than once
// because of an intervening write operation.
func Snapshot(b bool) options.OptSnapshot {
	opt := options.OptSnapshot(b)
	return opt
}

// Sort specifies order in which to return matching documents.
//
// This function uses TransformDocument to turn the sort parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// sort.
func Sort(sort interface{}) (options.OptSort, error) {
	doc, err := TransformDocument(sort)
	if err != nil {
		return options.OptSort{}, err
	}
	opt := options.OptSort{Sort: doc}
	return opt, nil
}

// Upsert specifies that a new document should be inserted if no document matches the update
// filter.
func Upsert(b bool) options.OptUpsert {
	opt := options.OptUpsert(b)
	return opt
}
