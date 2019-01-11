// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Code generated by drivergen. DO NOT EDIT.

package driverx

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/network/description"
)

// Find constructs and returns a new FindOperation.
func Find(filter bson.Raw) *FindOperation {
	return &FindOperation{filter: filter}
}

// AllowPartialResults when true allows partial results to be returned if some shards are down.
func (fo *FindOperation) AllowPartialResults(allowPartialResults bool) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.allowPartialResults = &allowPartialResults
	return fo
}

// AwaitData when true makes a cursor block before returning when no data is available.
func (fo *FindOperation) AwaitData(awaitData bool) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.awaitData = &awaitData
	return fo
}

// BatchSize specifies the number of documents to return in every batch.
func (fo *FindOperation) BatchSize(batchSize int64) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.batchSize = &batchSize
	return fo
}

// Session sets the session for this operation.
func (fo *FindOperation) Session(client *session.Client) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.client = client
	return fo
}

// ClusterClock sets the cluster clock for this operation.
func (fo *FindOperation) ClusterClock(clock *session.ClusterClock) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.clock = clock
	return fo
}

// Collation specifies a collation to be used.
func (fo *FindOperation) Collation(collation bson.Raw) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.collation = collation
	return fo
}

// Comment sets a string to help trace an operation.
func (fo *FindOperation) Comment(comment string) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.comment = &comment
	return fo
}

// Deployment sets the deployment to use for this operation.
func (fo *FindOperation) Deployment(d Deployment) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.d = d
	return fo
}

// Filter determines what results are returned from find.
func (fo *FindOperation) Filter(filter bson.Raw) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.filter = filter
	return fo
}

// Hint specifies the index to use.
func (fo *FindOperation) Hint(hint bson.RawValue) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.hint = hint
	return fo
}

// Limit sets a limit on the number of documents to return.
func (fo *FindOperation) Limit(limit int64) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.limit = &limit
	return fo
}

// Max sets an exclusive upper bound for a specific index.
func (fo *FindOperation) Max(max bson.Raw) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.max = max
	return fo
}

// MaxTimeMS specifies the maximum amount of time to allow the query to run.
func (fo *FindOperation) MaxTimeMS(maxTimeMS int64) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.maxTimeMS = &maxTimeMS
	return fo
}

// Min sets an inclusive lower bound for a specific index.
func (fo *FindOperation) Min(min bson.Raw) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.min = min
	return fo
}

// NoCursorTimeout when true prevents cursor from timing out after an inactivity period.
func (fo *FindOperation) NoCursorTimeout(noCursorTimeout bool) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.noCursorTimeout = &noCursorTimeout
	return fo
}

// Namespace sets the database and collection to run this operation against.
func (fo *FindOperation) Namespace(ns Namespace) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.ns = ns
	return fo
}

// OplogReplay when true replays a replica set's oplog.
func (fo *FindOperation) OplogReplay(oplogReplay bool) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.oplogReplay = &oplogReplay
	return fo
}

// Project limits the fields returned for all documents.
func (fo *FindOperation) Projection(projection bson.Raw) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.projection = projection
	return fo
}

// ReadConcern specifies the read concern for this operation.
func (fo *FindOperation) ReadConcern(readConcern *readconcern.ReadConcern) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.readConcern = readConcern
	return fo
}

// ReadPreference set the read prefernce used with this operation.
func (fo *FindOperation) ReadPreference(readPref *readpref.ReadPref) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.readPref = readPref
	return fo
}

// ReturnKey when true returns index keys for all result documents.
func (fo *FindOperation) ReturnKey(returnKey bool) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.returnKey = &returnKey
	return fo
}

// ServerSelector sets the selector used to retrieve a server.
func (fo *FindOperation) ServerSelector(serverSelector description.ServerSelector) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.serverSelector = serverSelector
	return fo
}

// ShowRecordID when true adds a $recordId field with the record identifier to returned documents.
func (fo *FindOperation) ShowRecordID(showRecordID bool) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.showRecordID = &showRecordID
	return fo
}

// SingleBatch specifies whether the results should be returned in a single batch.
func (fo *FindOperation) SingleBatch(singleBatch bool) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.singleBatch = &singleBatch
	return fo
}

// Skip specifies the number of documents to skip before returning.
func (fo *FindOperation) Skip(skip int64) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.skip = &skip
	return fo
}

// Sort specifies the order in which to return results.
func (fo *FindOperation) Sort(sort bson.Raw) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.sort = sort
	return fo
}

// Tailable keeps a cursor open and resumable after the last data has been retrieved.
func (fo *FindOperation) Tailable(tailable bool) *FindOperation {
	if fo == nil {
		fo = new(FindOperation)
	}

	fo.tailable = &tailable
	return fo
}
