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
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// listCollectionsOp performs a listCollections operation.
type listCollectionsOp struct {
	authenticator             driver.Authenticator
	filter                    bsoncore.Document
	nameOnly                  *bool
	authorizedCollections     *bool
	session                   *session.Client
	clock                     *session.ClusterClock
	monitor                   *event.CommandMonitor
	crypt                     driver.Crypt
	database                  string
	deployment                driver.Deployment
	readPreference            *readpref.ReadPref
	selector                  description.ServerSelector
	retry                     *driver.RetryMode
	maxAdaptiveRetries        uint
	enableOverloadRetargeting bool
	res                       driver.CursorResponse
	batchSize                 *int32
	serverAPI                 *driver.ServerAPIOptions
	timeout                   *time.Duration
	rawData                   *bool
}

// result returns the result of executing this operation.
func (lc *listCollectionsOp) result(opts driver.CursorOptions) (*driver.BatchCursor, error) {
	opts.ServerAPI = lc.serverAPI

	return driver.NewBatchCursor(lc.res, lc.session, lc.clock, opts)
}

func (lc *listCollectionsOp) processResponse(_ context.Context, resp bsoncore.Document, info driver.ResponseInfo) error {
	curDoc, err := driver.ExtractCursorDocument(resp)
	if err != nil {
		return err
	}
	lc.res, err = driver.NewCursorResponse(curDoc, info)
	return err
}

// execute runs this operations and returns an error if the operation did not execute successfully.
func (lc *listCollectionsOp) execute(ctx context.Context) error {
	if lc.deployment == nil {
		return errors.New("the ListCollections operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:                 lc.command,
		ProcessResponseFn:         lc.processResponse,
		RetryMode:                 lc.retry,
		MaxAdaptiveRetries:        lc.maxAdaptiveRetries,
		EnableOverloadRetargeting: lc.enableOverloadRetargeting,
		Type:                      driver.Read,
		Client:                    lc.session,
		Clock:                     lc.clock,
		CommandMonitor:            lc.monitor,
		Crypt:                     lc.crypt,
		Database:                  lc.database,
		Deployment:                lc.deployment,
		ReadPreference:            lc.readPreference,
		Selector:                  lc.selector,
		Legacy:                    driver.LegacyListCollections,
		ServerAPI:                 lc.serverAPI,
		Timeout:                   lc.timeout,
		Name:                      driverutil.ListCollectionsOp,
		Authenticator:             lc.authenticator,
	}.Execute(ctx)
}

func (lc *listCollectionsOp) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendInt32Element(dst, "listCollections", 1)
	if lc.filter != nil {
		dst = bsoncore.AppendDocumentElement(dst, "filter", lc.filter)
	}
	if lc.nameOnly != nil {
		dst = bsoncore.AppendBooleanElement(dst, "nameOnly", *lc.nameOnly)
	}
	if lc.authorizedCollections != nil {
		dst = bsoncore.AppendBooleanElement(dst, "authorizedCollections", *lc.authorizedCollections)
	}

	cursorDoc := bsoncore.NewDocumentBuilder()
	if lc.batchSize != nil {
		cursorDoc.AppendInt32("batchSize", *lc.batchSize)
	}
	dst = bsoncore.AppendDocumentElement(dst, "cursor", cursorDoc.Build())

	// Set rawData for 8.2+ servers.
	if lc.rawData != nil && desc.WireVersion != nil && driverutil.VersionRangeIncludes(*desc.WireVersion, 27) {
		dst = bsoncore.AppendBooleanElement(dst, "rawData", *lc.rawData)
	}

	return dst, nil
}
