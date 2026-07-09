// Copyright (C) MongoDB, Inc. 2019-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/driverutil"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// aggregateOp represents an aggregate operation.
type aggregateOp struct {
	authenticator             driver.Authenticator
	allowDiskUse              *bool
	batchSize                 *int32
	bypassDocumentValidation  *bool
	collation                 bsoncore.Document
	comment                   bsoncore.Value
	hint                      bsoncore.Value
	pipeline                  bsoncore.Document
	session                   *session.Client
	clock                     *session.ClusterClock
	collection                string
	monitor                   *event.CommandMonitor
	database                  string
	deployment                driver.Deployment
	readConcern               *readconcern.ReadConcern
	readPreference            *readpref.ReadPref
	retry                     *driver.RetryMode
	maxAdaptiveRetries        uint
	enableOverloadRetargeting bool
	selector                  description.ServerSelector
	writeConcern              *writeconcern.WriteConcern
	crypt                     driver.Crypt
	serverAPI                 *driver.ServerAPIOptions
	let                       bsoncore.Document
	hasOutputStage            bool
	customOptions             map[string]bsoncore.Value
	timeout                   *time.Duration
	omitMaxTimeMS             bool
	rawData                   *bool

	result driver.CursorResponse
}

// Result returns the result of executing this operation.
func (a *aggregateOp) Result(opts driver.CursorOptions) (*driver.BatchCursor, error) {
	clientSession := a.session

	clock := a.clock
	opts.ServerAPI = a.serverAPI
	return driver.NewBatchCursor(a.result, clientSession, clock, opts)
}

// ResultCursorResponse returns the underlying CursorResponse result of executing this
// operation.
func (a *aggregateOp) ResultCursorResponse() driver.CursorResponse {
	return a.result
}

func (a *aggregateOp) processResponse(_ context.Context, resp bsoncore.Document, info driver.ResponseInfo) error {
	curDoc, err := driver.ExtractCursorDocument(resp)
	if err != nil {
		return err
	}
	a.result, err = driver.NewCursorResponse(curDoc, info)
	return err
}

// Execute runs this operations and returns an error if the operation did not execute successfully.
func (a *aggregateOp) Execute(ctx context.Context) error {
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
		MaxAdaptiveRetries:             a.maxAdaptiveRetries,
		EnableOverloadRetargeting:      a.enableOverloadRetargeting,
		Selector:                       a.selector,
		WriteConcern:                   a.writeConcern,
		Crypt:                          a.crypt,
		MinimumWriteConcernWireVersion: 5,
		ServerAPI:                      a.serverAPI,
		IsOutputAggregate:              a.hasOutputStage,
		Timeout:                        a.timeout,
		Name:                           driverutil.AggregateOp,
		Authenticator:                  a.authenticator,
		OmitMaxTimeMS:                  a.omitMaxTimeMS,
	}.Execute(ctx)
}

func (a *aggregateOp) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	header := bsoncore.Value{Type: bsoncore.TypeString, Data: bsoncore.AppendString(nil, a.collection)}
	if a.collection == "" {
		header = bsoncore.Value{Type: bsoncore.TypeInt32, Data: []byte{0x01, 0x00, 0x00, 0x00}}
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
		if desc.WireVersion == nil || !driverutil.VersionRangeIncludes(*desc.WireVersion, 5) {
			return nil, errors.New("the 'collation' command parameter requires a minimum server wire version of 5")
		}
		dst = bsoncore.AppendDocumentElement(dst, "collation", a.collation)
	}
	if a.comment.Type != bsoncore.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "comment", a.comment)
	}
	if a.hint.Type != bsoncore.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "hint", a.hint)
	}
	if a.pipeline != nil {
		dst = bsoncore.AppendArrayElement(dst, "pipeline", a.pipeline)
	}
	if a.let != nil {
		dst = bsoncore.AppendDocumentElement(dst, "let", a.let)
	}
	// Set rawData for 8.2+ servers.
	if a.rawData != nil && desc.WireVersion != nil && driverutil.VersionRangeIncludes(*desc.WireVersion, 27) {
		dst = bsoncore.AppendBooleanElement(dst, "rawData", *a.rawData)
	}
	for optionName, optionValue := range a.customOptions {
		dst = bsoncore.AppendValueElement(dst, optionName, optionValue)
	}
	cursorDoc, _ = bsoncore.AppendDocumentEnd(cursorDoc, cursorIdx)
	dst = bsoncore.AppendDocumentElement(dst, "cursor", cursorDoc)

	return dst, nil
}
