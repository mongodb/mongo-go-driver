// Copyright (C) MongoDB, Inc. 2019-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package operation

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/driverutil"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
)

// Aggregate represents an aggregate operation.
type Aggregate struct {
	// AllowDiskUse enables writing to temporary files. When true, aggregation
	// stages can write to the dbPath/_tmp directory.
	AllowDiskUse *bool

	// BatchSize specifies the number of documents to return in every batch.
	BatchSize *int32

	// BypassDocumentValidation allows the write to opt-out of document level
	// validation. This only applies when the $out stage is specified.
	BypassDocumentValidation *bool

	// Collation specifies a collation. This option is only valid for server
	// versions 3.4 and above.
	Collation bsoncore.Document

	// Comment specifies an arbitrary string to help trace the operation through
	// the database profiler, currentOp, and logs.
	Comment *string

	// Hint specifies the index to use.
	Hint bsoncore.Value

	// MaxTime specifies the maximum amount of time to allow the query to run on
	// the server.
	MaxTime *time.Duration

	// Pipeline determines how data is transformed for an aggregation.
	Pipeline bsoncore.Document

	// Session is the session for this operation.
	Session *session.Client

	// Clock is the cluster clock for this operation.
	Clock *session.ClusterClock

	// Collection is the collection that this command will run against.
	Collection string

	// Monitor is the monitor to use for APM events.
	Monitor *event.CommandMonitor

	// Database is the database to run this operation against.
	Database string

	// Deployment is the deployment to use for this operation.
	Deployment driver.Deployment

	// ReadConcern is the read concern for this operation.
	ReadConcern *readconcern.ReadConcern

	// ReadPreference is the read preference used with this operation.
	ReadPreference *readpref.ReadPref

	// Retry enables retryable writes for this operation. Retries are not
	// handled automatically, instead a boolean is returned from Execute and
	// SelectAndExecute that indicates if the operation can be retried. Retrying
	// is handled by calling RetryExecute.
	Retry *driver.RetryMode

	// Selector is the selector used to retrieve a server.
	Selector description.ServerSelector

	// WriteConcern is the write concern for this operation.
	WriteConcern *writeconcern.WriteConcern

	// Crypt is the Crypt object to use for automatic encryption and decryption.
	Crypt driver.Crypt

	// ServerAPI is the server API version for this operation.
	ServerAPI *driver.ServerAPIOptions

	// Let specifies the let document to use. This option is only valid for
	// server versions 5.0 and above.
	Let bsoncore.Document

	// HasOutputStage specifies whether the aggregate contains an output stage.
	// Used in determining when to append read preference at the operation
	// level.
	HasOutputStage bool

	// CustomOptions specifies extra options to use in the aggregate command.
	CustomOptions map[string]bsoncore.Value

	// Timeout is the timeout for this operation.
	Timeout *time.Duration

	result driver.CursorResponse
}

// Result returns the result of executing this operation.
func (a *Aggregate) Result(opts driver.CursorOptions) (*driver.BatchCursor, error) {

	clientSession := a.Session

	clock := a.Clock
	opts.ServerAPI = a.ServerAPI
	return driver.NewBatchCursor(a.result, clientSession, clock, opts)
}

// ResultCursorResponse returns the underlying CursorResponse result of executing this
// operation.
func (a *Aggregate) ResultCursorResponse() driver.CursorResponse {
	return a.result
}

func (a *Aggregate) processResponse(info driver.ResponseInfo) error {
	var err error

	a.result, err = driver.NewCursorResponse(info)
	return err

}

// Execute runs this operations and returns an error if the operation did not execute successfully.
func (a *Aggregate) Execute(ctx context.Context) error {
	if a.Deployment == nil {
		return errors.New("the Aggregate operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:         a.command,
		ProcessResponseFn: a.processResponse,

		Client:                         a.Session,
		Clock:                          a.Clock,
		CommandMonitor:                 a.Monitor,
		Database:                       a.Database,
		Deployment:                     a.Deployment,
		ReadConcern:                    a.ReadConcern,
		ReadPreference:                 a.ReadPreference,
		Type:                           driver.Read,
		RetryMode:                      a.Retry,
		Selector:                       a.Selector,
		WriteConcern:                   a.WriteConcern,
		Crypt:                          a.Crypt,
		MinimumWriteConcernWireVersion: 5,
		ServerAPI:                      a.ServerAPI,
		IsOutputAggregate:              a.HasOutputStage,
		MaxTime:                        a.MaxTime,
		Timeout:                        a.Timeout,
		Name:                           driverutil.AggregateOp,
	}.Execute(ctx)

}

func (a *Aggregate) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	header := bsoncore.Value{Type: bsontype.String, Data: bsoncore.AppendString(nil, a.Collection)}
	if a.Collection == "" {
		header = bsoncore.Value{Type: bsontype.Int32, Data: []byte{0x01, 0x00, 0x00, 0x00}}
	}
	dst = bsoncore.AppendValueElement(dst, "aggregate", header)

	cursorIdx, cursorDoc := bsoncore.AppendDocumentStart(nil)
	if a.AllowDiskUse != nil {

		dst = bsoncore.AppendBooleanElement(dst, "allowDiskUse", *a.AllowDiskUse)
	}
	if a.BatchSize != nil {
		cursorDoc = bsoncore.AppendInt32Element(cursorDoc, "batchSize", *a.BatchSize)
	}
	if a.BypassDocumentValidation != nil {

		dst = bsoncore.AppendBooleanElement(dst, "bypassDocumentValidation", *a.BypassDocumentValidation)
	}
	if a.Collation != nil {

		if desc.WireVersion == nil || !desc.WireVersion.Includes(5) {
			return nil, errors.New("the 'collation' command parameter requires a minimum server wire version of 5")
		}
		dst = bsoncore.AppendDocumentElement(dst, "collation", a.Collation)
	}
	if a.Comment != nil {

		dst = bsoncore.AppendStringElement(dst, "comment", *a.Comment)
	}
	if a.Hint.Type != bsontype.Type(0) {

		dst = bsoncore.AppendValueElement(dst, "hint", a.Hint)
	}
	if a.Pipeline != nil {

		dst = bsoncore.AppendArrayElement(dst, "pipeline", a.Pipeline)
	}
	if a.Let != nil {
		dst = bsoncore.AppendDocumentElement(dst, "let", a.Let)
	}
	for optionName, optionValue := range a.CustomOptions {
		dst = bsoncore.AppendValueElement(dst, optionName, optionValue)
	}
	cursorDoc, _ = bsoncore.AppendDocumentEnd(cursorDoc, cursorIdx)
	dst = bsoncore.AppendDocumentElement(dst, "cursor", cursorDoc)

	return dst, nil
}
