// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// Opt is a convenience variable provided for access to Options methods.
var Opt Options

// Options is used as a namespace for MongoDB operation option constructors.
type Options struct{}

// AllowDiskUse enables writing to temporary files.
func (Options) AllowDiskUse(b bool) option.OptAllowDiskUse {
	opt := option.OptAllowDiskUse(b)
	return opt
}

// AllowPartialResults gets partial results from a mongos if some shards are down (instead of
// throwing an error).
func (Options) AllowPartialResults(b bool) option.OptAllowPartialResults {
	opt := option.OptAllowPartialResults(b)
	return opt
}

// ArrayFilters specifies to which array elements an update should apply.
//
// This function uses TransformDocument to turn the document parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// document.
func (Options) ArrayFilters(filters ...interface{}) (option.OptArrayFilters, error) {
	docs := make([]*bson.Document, 0, len(filters))
	for _, f := range filters {
		d, err := TransformDocument(f)
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	opt := option.OptArrayFilters(docs)
	return opt, nil
}

// BatchSize specifies the number of documents to return per batch.
func (Options) BatchSize(i int32) option.OptBatchSize {
	opt := option.OptBatchSize(i)
	return opt
}

// BypassDocumentValidation is used to opt out of document-level validation for a given write.
func (Options) BypassDocumentValidation(b bool) option.OptBypassDocumentValidation {
	opt := option.OptBypassDocumentValidation(b)
	return opt
}

// Collation allows users to specify language-specific rules for string comparison, such as rules
// for lettercase and accent marks.
//
// See https://docs.mongodb.com/manual/reference/collation/.
func (Options) Collation(collation *option.Collation) option.OptCollation {
	opt := option.OptCollation{Collation: collation}
	return opt
}

// Comment specifies a comment to attach to the query to help attach and trace profile data.
//
// See https://docs.mongodb.com/manual/reference/command/profile.
func (Options) Comment(s string) option.OptComment {
	opt := option.OptComment(s)
	return opt
}

// CursorType indicates the type of cursor to use.
func (Options) CursorType(cursorType option.CursorType) option.OptCursorType {
	opt := option.OptCursorType(cursorType)
	return opt
}

// FullDocument indicates which changes should be returned in a change stream notification.
//
// If fullDocument is "updateLookup", then the change notification for partial updates will include
// both a delta describing the changes to the document as well as a copy of the entire document
// that was changed from some time after the change occurred.
func (Options) FullDocument(fullDocument string) option.OptFullDocument {
	opt := option.OptFullDocument(fullDocument)
	return opt
}

// Hint specifies the index to use.
//
// The hint parameter must be either a string or a document. If it is a
// document, this func (Options)tion uses TransformDocument to turn the document into a
// *bson.Document. See TransformDocument for the list of valid types.
func (Options) Hint(hint interface{}) (option.OptHint, error) {
	switch hint.(type) {
	case string:
	default:
		var err error
		hint, err = TransformDocument(hint)
		if err != nil {
			return option.OptHint{}, err
		}
	}
	opt := option.OptHint{Hint: hint}
	return opt, nil
}

// Limit specifies the maximum number of documents to return.
func (Options) Limit(i int64) option.OptLimit {
	opt := option.OptLimit(i)
	return opt
}

// Max specifies the exclusive upper bound for a specific index.
//
// This function uses TransformDocument to turn the max parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// max.
func (Options) Max(max interface{}) (option.OptMax, error) {
	doc, err := TransformDocument(max)
	if err != nil {
		return option.OptMax{}, err
	}
	opt := option.OptMax{Max: doc}
	return opt, nil
}

// MaxAwaitTime specifies the maximum amount of time for the server to wait on new documents to
// satisfy a tailable-await cursor query.
func (Options) MaxAwaitTime(duration time.Duration) option.OptMaxAwaitTime {
	opt := option.OptMaxAwaitTime(duration)
	return opt
}

// MaxScan specifies the maximum number of documents or index keys to scan when executing the query.
func (Options) MaxScan(i int64) option.OptMaxScan {
	opt := option.OptMaxScan(i)
	return opt
}

// MaxTime specifies the maximum amount of time to allow the query to run.
func (Options) MaxTime(duration time.Duration) option.OptMaxTime {
	opt := option.OptMaxTime(duration)
	return opt
}

// Min specifies the inclusive lower bound for a specific index.
//
// This function uses TransformDocument to turn the min parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// min.
func (Options) Min(min interface{}) (option.OptMin, error) {
	doc, err := TransformDocument(min)
	if err != nil {
		return option.OptMin{}, err
	}
	opt := option.OptMin{Min: doc}
	return opt, nil
}

// NoCursorTimeout specifies whether to prevent the server from timing out idle cursors after an
// inactivity period.
func (Options) NoCursorTimeout(b bool) option.OptNoCursorTimeout {
	opt := option.OptNoCursorTimeout(b)
	return opt
}

// OplogReplay is for internal replication use only.
func (Options) OplogReplay(b bool) option.OptOplogReplay {
	opt := option.OptOplogReplay(b)
	return opt
}

// Ordered specifies whether the remaining writes should be aborted if one of the earlier ones fails.
func (Options) Ordered(b bool) option.OptOrdered {
	opt := option.OptOrdered(b)
	return opt
}

// Projection limits the fields to return for all matching documents.
//
// This function uses TransformDocument to turn the projection parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// projection.
func (Options) Projection(projection interface{}) (option.OptProjection, error) {
	doc, err := TransformDocument(projection)
	if err != nil {
		return option.OptProjection{}, nil
	}
	opt := option.OptProjection{Projection: doc}
	return opt, nil
}

// ReadConcern for replica sets and replica set shards determines which data to return from a query.
func (Options) ReadConcern(readConcern *readconcern.ReadConcern) (option.OptReadConcern, error) {
	elem, err := readConcern.MarshalBSONElement()
	if err != nil {
		return option.OptReadConcern{}, err
	}

	opt := option.OptReadConcern{ReadConcern: elem}
	return opt, nil
}

// ResumeAfter specifies the logical starting point for a new change stream.
func (Options) ResumeAfter(token *bson.Document) option.OptResumeAfter {
	opt := option.OptResumeAfter{ResumeAfter: token}
	return opt
}

// ReturnDocument specifies whether a findAndUpdate should return the document as it was before the
// update or as it is after.
func (Options) ReturnDocument(returnDocument option.ReturnDocument) option.OptReturnDocument {
	opt := option.OptReturnDocument(returnDocument)
	return opt
}

// ReturnKey specifies whether to only return the index keys in the resulting documents.
func (Options) ReturnKey(b bool) option.OptReturnKey {
	opt := option.OptReturnKey(b)
	return opt
}

// ShowRecordID determines whether to return the record identifier for each document.
func (Options) ShowRecordID(b bool) option.OptShowRecordID {
	opt := option.OptShowRecordID(b)
	return opt
}

// Skip specifies the number of documents to skip before returning.
func (Options) Skip(i int64) option.OptSkip {
	opt := option.OptSkip(i)
	return opt
}

// Snapshot specifies whether to prevent the cursor from returning a document more than once
// because of an intervening write operation.
func (Options) Snapshot(b bool) option.OptSnapshot {
	opt := option.OptSnapshot(b)
	return opt
}

// Sort specifies order in which to return matching documents.
//
// This function uses TransformDocument to turn the sort parameter into a
// *bson.Document. See TransformDocument for the list of valid types for
// sort.
func (Options) Sort(sort interface{}) (option.OptSort, error) {
	doc, err := TransformDocument(sort)
	if err != nil {
		return option.OptSort{}, err
	}
	opt := option.OptSort{Sort: doc}
	return opt, nil
}

// Upsert specifies that a new document should be inserted if no document matches the update
// filter.
func (Options) Upsert(b bool) option.OptUpsert {
	opt := option.OptUpsert(b)
	return opt
}

// WriteConcern describes the level of acknowledgement requested from MongoDB for write operations
// to a standalone mongod or to replica sets or to sharded clusters.
func (Options) WriteConcern(writeConcern *writeconcern.WriteConcern) (option.OptWriteConcern, error) {
	elem, err := writeConcern.MarshalBSONElement()
	if err != nil {
		return option.OptWriteConcern{}, err
	}

	opt := option.OptWriteConcern{WriteConcern: elem, Acknowledged: writeConcern.Acknowledged()}
	return opt, nil
}
