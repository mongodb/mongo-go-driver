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
	"go.mongodb.org/mongo-driver/v2/internal/logger"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// findOp performs a find operation.
type findOp struct {
	authenticator             driver.Authenticator
	allowDiskUse              *bool
	allowPartialResults       *bool
	awaitData                 *bool
	batchSize                 *int32
	collation                 bsoncore.Document
	comment                   bsoncore.Value
	filter                    bsoncore.Document
	hint                      bsoncore.Value
	let                       bsoncore.Document
	limit                     *int64
	max                       bsoncore.Document
	min                       bsoncore.Document
	noCursorTimeout           *bool
	oplogReplay               *bool
	projection                bsoncore.Document
	returnKey                 *bool
	showRecordID              *bool
	singleBatch               *bool
	skip                      *int64
	snapshot                  *bool
	sort                      bsoncore.Document
	tailable                  *bool
	session                   *session.Client
	clock                     *session.ClusterClock
	collection                string
	monitor                   *event.CommandMonitor
	crypt                     driver.Crypt
	database                  string
	deployment                driver.Deployment
	readConcern               *readconcern.ReadConcern
	readPreference            *readpref.ReadPref
	selector                  description.ServerSelector
	retry                     *driver.RetryMode
	maxAdaptiveRetries        uint
	enableOverloadRetargeting bool
	cursorRes                 driver.CursorResponse
	serverAPI                 *driver.ServerAPIOptions
	timeout                   *time.Duration
	rawData                   *bool
	logger                    *logger.Logger
	omitMaxTimeMS             bool
}

// result returns the result of executing this operation.
func (f *findOp) result(opts driver.CursorOptions) (*driver.BatchCursor, error) {
	opts.ServerAPI = f.serverAPI
	return driver.NewBatchCursor(f.cursorRes, f.session, f.clock, opts)
}

func (f *findOp) processResponse(_ context.Context, resp bsoncore.Document, info driver.ResponseInfo) error {
	curDoc, err := driver.ExtractCursorDocument(resp)
	if err != nil {
		return err
	}
	f.cursorRes, err = driver.NewCursorResponse(curDoc, info)
	return err
}

// execute runs this operation and returns an error if the operation did not execute successfully.
func (f *findOp) execute(ctx context.Context) error {
	if f.deployment == nil {
		return errors.New("the find operation must have a Deployment set before execute can be called")
	}

	return driver.Operation{
		CommandFn:                 f.command,
		ProcessResponseFn:         f.processResponse,
		RetryMode:                 f.retry,
		MaxAdaptiveRetries:        f.maxAdaptiveRetries,
		EnableOverloadRetargeting: f.enableOverloadRetargeting,
		Type:                      driver.Read,
		Client:                    f.session,
		Clock:                     f.clock,
		CommandMonitor:            f.monitor,
		Crypt:                     f.crypt,
		Database:                  f.database,
		Deployment:                f.deployment,
		ReadConcern:               f.readConcern,
		ReadPreference:            f.readPreference,
		Selector:                  f.selector,
		Legacy:                    driver.LegacyFind,
		ServerAPI:                 f.serverAPI,
		Timeout:                   f.timeout,
		Logger:                    f.logger,
		Name:                      driverutil.FindOp,
		Authenticator:             f.authenticator,
		OmitMaxTimeMS:             f.omitMaxTimeMS,
	}.Execute(ctx)
}

func (f *findOp) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "find", f.collection)
	if f.allowDiskUse != nil {
		if desc.WireVersion == nil || !driverutil.VersionRangeIncludes(*desc.WireVersion, 4) {
			return nil, errors.New("the 'allowDiskUse' command parameter requires a minimum server wire version of 4")
		}
		dst = bsoncore.AppendBooleanElement(dst, "allowDiskUse", *f.allowDiskUse)
	}
	if f.allowPartialResults != nil {
		dst = bsoncore.AppendBooleanElement(dst, "allowPartialResults", *f.allowPartialResults)
	}
	if f.awaitData != nil {
		dst = bsoncore.AppendBooleanElement(dst, "awaitData", *f.awaitData)
	}
	if f.batchSize != nil {
		dst = bsoncore.AppendInt32Element(dst, "batchSize", *f.batchSize)
	}
	if f.collation != nil {
		if desc.WireVersion == nil || !driverutil.VersionRangeIncludes(*desc.WireVersion, 5) {
			return nil, errors.New("the 'collation' command parameter requires a minimum server wire version of 5")
		}
		dst = bsoncore.AppendDocumentElement(dst, "collation", f.collation)
	}
	if f.comment.Type != bsoncore.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "comment", f.comment)
	}
	if f.filter != nil {
		dst = bsoncore.AppendDocumentElement(dst, "filter", f.filter)
	}
	if f.hint.Type != bsoncore.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "hint", f.hint)
	}
	if f.let != nil {
		dst = bsoncore.AppendDocumentElement(dst, "let", f.let)
	}
	if f.limit != nil {
		dst = bsoncore.AppendInt64Element(dst, "limit", *f.limit)
	}
	if f.max != nil {
		dst = bsoncore.AppendDocumentElement(dst, "max", f.max)
	}
	if f.min != nil {
		dst = bsoncore.AppendDocumentElement(dst, "min", f.min)
	}
	if f.noCursorTimeout != nil {
		dst = bsoncore.AppendBooleanElement(dst, "noCursorTimeout", *f.noCursorTimeout)
	}
	if f.oplogReplay != nil {
		dst = bsoncore.AppendBooleanElement(dst, "oplogReplay", *f.oplogReplay)
	}
	if f.projection != nil {
		dst = bsoncore.AppendDocumentElement(dst, "projection", f.projection)
	}
	if f.returnKey != nil {
		dst = bsoncore.AppendBooleanElement(dst, "returnKey", *f.returnKey)
	}
	if f.showRecordID != nil {
		dst = bsoncore.AppendBooleanElement(dst, "showRecordId", *f.showRecordID)
	}
	if f.singleBatch != nil {
		dst = bsoncore.AppendBooleanElement(dst, "singleBatch", *f.singleBatch)
	}
	if f.skip != nil {
		dst = bsoncore.AppendInt64Element(dst, "skip", *f.skip)
	}
	if f.snapshot != nil {
		dst = bsoncore.AppendBooleanElement(dst, "snapshot", *f.snapshot)
	}
	if f.sort != nil {
		dst = bsoncore.AppendDocumentElement(dst, "sort", f.sort)
	}
	if f.tailable != nil {
		dst = bsoncore.AppendBooleanElement(dst, "tailable", *f.tailable)
	}
	// Set rawData for 8.2+ servers.
	if f.rawData != nil && desc.WireVersion != nil && driverutil.VersionRangeIncludes(*desc.WireVersion, 27) {
		dst = bsoncore.AppendBooleanElement(dst, "rawData", *f.rawData)
	}
	return dst, nil
}
