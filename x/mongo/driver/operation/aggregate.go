// Copyright (C) MongoDB, Inc. 2019-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Code generated by operationgen. DO NOT EDIT.

package operation

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
)

// Performs an aggregate operation
type Aggregate struct {
	allowDiskUse             *bool
	batchSize                *int32
	bypassDocumentValidation *bool
	collation                bsoncore.Document
	comment                  *string
	hint                     bsoncore.Value
	maxTimeMS                *int64
	pipeline                 bsoncore.Document
	session                  *session.Client
	clock                    *session.ClusterClock
	collection               string
	monitor                  *event.CommandMonitor
	database                 string
	deployment               driver.Deployment
	readConcern              *readconcern.ReadConcern
	readPreference           *readpref.ReadPref
	retry                    *driver.RetryMode
	selector                 description.ServerSelector
	writeConcern             *writeconcern.WriteConcern
	crypt                    *driver.Crypt

	result driver.CursorResponse
}

// NewAggregate constructs and returns a new Aggregate.
func NewAggregate(pipeline bsoncore.Document) *Aggregate {
	return &Aggregate{
		pipeline: pipeline,
	}
}

// Result returns the result of executing this operation.
func (a *Aggregate) Result(opts driver.CursorOptions) (*driver.BatchCursor, error) {

	clientSession := a.session

	clock := a.clock
	return driver.NewBatchCursor(a.result, clientSession, clock, opts)
}

func (a *Aggregate) ResultCursorResponse() driver.CursorResponse {
	return a.result
}

func (a *Aggregate) processResponse(response bsoncore.Document, srvr driver.Server, desc description.Server) error {
	var err error

	a.result, err = driver.NewCursorResponse(response, srvr, desc)
	return err

}

// Execute runs this operations and returns an error if the operaiton did not execute successfully.
func (a *Aggregate) Execute(ctx context.Context) error {
	if a.deployment == nil {
		return errors.New("the Aggregate operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:         a.command,
		ProcessResponseFn: a.processResponse,

		Client:                         a.session,
		Clock:                          a.clock,
		CommandMonitor:                 a.monitor,
		Database:                       a.database,
		Deployment:                     a.deployment,
		ReadConcern:                    a.readConcern,
		ReadPreference:                 a.readPreference,
		Type:                           driver.Read,
		RetryMode:                      a.retry,
		Selector:                       a.selector,
		WriteConcern:                   a.writeConcern,
		Crypt:                          a.crypt,
		MinimumWriteConcernWireVersion: 5,
	}.Execute(ctx, nil)

}

func (a *Aggregate) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	header := bsoncore.Value{Type: bsontype.String, Data: bsoncore.AppendString(nil, a.collection)}
	if a.collection == "" {
		header = bsoncore.Value{Type: bsontype.Int32, Data: []byte{0x01, 0x00, 0x00, 0x00}}
	}
	dst = bsoncore.AppendValueElement(dst, "aggregate", header)

	cursorIdx, cursorDoc := bsoncore.AppendDocumentStart(nil)
	if a.allowDiskUse != nil {

		dst = bsoncore.AppendBooleanElement(dst, "allowDiskUse", *a.allowDiskUse)
	}
	if a.batchSize != nil {
		cursorDoc = bsoncore.AppendInt32Element(cursorDoc, "batchSize", *a.batchSize)
	}
	if a.bypassDocumentValidation != nil {

		dst = bsoncore.AppendBooleanElement(dst, "bypassDocumentValidation", *a.bypassDocumentValidation)
	}
	if a.collation != nil {

		if desc.WireVersion == nil || !desc.WireVersion.Includes(5) {
			return nil, errors.New("the 'collation' command parameter requires a minimum server wire version of 5")
		}
		dst = bsoncore.AppendDocumentElement(dst, "collation", a.collation)
	}
	if a.comment != nil {

		dst = bsoncore.AppendStringElement(dst, "comment", *a.comment)
	}
	if a.hint.Type != bsontype.Type(0) {

		dst = bsoncore.AppendValueElement(dst, "hint", a.hint)
	}
	if a.maxTimeMS != nil {

		dst = bsoncore.AppendInt64Element(dst, "maxTimeMS", *a.maxTimeMS)
	}
	if a.pipeline != nil {

		dst = bsoncore.AppendArrayElement(dst, "pipeline", a.pipeline)
	}
	cursorDoc, _ = bsoncore.AppendDocumentEnd(cursorDoc, cursorIdx)
	dst = bsoncore.AppendDocumentElement(dst, "cursor", cursorDoc)

	return dst, nil
}

// AllowDiskUse enables writing to temporary files. When true, aggregation stages can write to the dbPath/_tmp directory.
func (a *Aggregate) AllowDiskUse(allowDiskUse bool) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.allowDiskUse = &allowDiskUse
	return a
}

// BatchSize specifies the number of documents to return in every batch.
func (a *Aggregate) BatchSize(batchSize int32) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.batchSize = &batchSize
	return a
}

// BypassDocumentValidation allows the write to opt-out of document level validation. This only applies when the $out stage is specified.
func (a *Aggregate) BypassDocumentValidation(bypassDocumentValidation bool) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.bypassDocumentValidation = &bypassDocumentValidation
	return a
}

// Collation specifies a collation. This option is only valid for server versions 3.4 and above.
func (a *Aggregate) Collation(collation bsoncore.Document) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.collation = collation
	return a
}

// Comment specifies an arbitrary string to help trace the operation through the database profiler, currentOp, and logs.
func (a *Aggregate) Comment(comment string) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.comment = &comment
	return a
}

// Hint specifies the index to use.
func (a *Aggregate) Hint(hint bsoncore.Value) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.hint = hint
	return a
}

// MaxTimeMS specifies the maximum amount of time to allow the query to run.
func (a *Aggregate) MaxTimeMS(maxTimeMS int64) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.maxTimeMS = &maxTimeMS
	return a
}

// Pipeline determines how data is transformed for an aggregation.
func (a *Aggregate) Pipeline(pipeline bsoncore.Document) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.pipeline = pipeline
	return a
}

// Session sets the session for this operation.
func (a *Aggregate) Session(session *session.Client) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.session = session
	return a
}

// ClusterClock sets the cluster clock for this operation.
func (a *Aggregate) ClusterClock(clock *session.ClusterClock) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.clock = clock
	return a
}

// Collection sets the collection that this command will run against.
func (a *Aggregate) Collection(collection string) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.collection = collection
	return a
}

// CommandMonitor sets the monitor to use for APM events.
func (a *Aggregate) CommandMonitor(monitor *event.CommandMonitor) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.monitor = monitor
	return a
}

// Database sets the database to run this operation against.
func (a *Aggregate) Database(database string) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.database = database
	return a
}

// Deployment sets the deployment to use for this operation.
func (a *Aggregate) Deployment(deployment driver.Deployment) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.deployment = deployment
	return a
}

// ReadConcern specifies the read concern for this operation.
func (a *Aggregate) ReadConcern(readConcern *readconcern.ReadConcern) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.readConcern = readConcern
	return a
}

// ReadPreference set the read prefernce used with this operation.
func (a *Aggregate) ReadPreference(readPreference *readpref.ReadPref) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.readPreference = readPreference
	return a
}

// ServerSelector sets the selector used to retrieve a server.
func (a *Aggregate) ServerSelector(selector description.ServerSelector) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.selector = selector
	return a
}

// WriteConcern sets the write concern for this operation.
func (a *Aggregate) WriteConcern(writeConcern *writeconcern.WriteConcern) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.writeConcern = writeConcern
	return a
}

// Retry enables retryable writes for this operation. Retries are not handled automatically,
// instead a boolean is returned from Execute and SelectAndExecute that indicates if the
// operation can be retried. Retrying is handled by calling RetryExecute.
func (a *Aggregate) Retry(retry driver.RetryMode) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.retry = &retry
	return a
}

// Crypt sets the Crypt object to use for automatic encryption and decryption.
func (a *Aggregate) Crypt(crypt *driver.Crypt) *Aggregate {
	if a == nil {
		a = new(Aggregate)
	}

	a.crypt = crypt
	return a
}
