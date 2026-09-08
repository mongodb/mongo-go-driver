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
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// listIndexesOp performs a listIndexes operation.
type listIndexesOp struct {
	authenticator             driver.Authenticator
	batchSize                 *int32
	session                   *session.Client
	clock                     *session.ClusterClock
	collection                string
	monitor                   *event.CommandMonitor
	database                  string
	deployment                driver.Deployment
	selector                  description.ServerSelector
	retry                     *driver.RetryMode
	maxAdaptiveRetries        uint
	enableOverloadRetargeting bool
	crypt                     driver.Crypt
	serverAPI                 *driver.ServerAPIOptions
	timeout                   *time.Duration
	rawData                   *bool

	res driver.CursorResponse
}

// result returns the result of executing this operation.
func (li *listIndexesOp) result(opts driver.CursorOptions) (*driver.BatchCursor, error) {
	clientSession := li.session

	clock := li.clock
	opts.ServerAPI = li.serverAPI
	return driver.NewBatchCursor(li.res, clientSession, clock, opts)
}

func (li *listIndexesOp) processResponse(_ context.Context, resp bsoncore.Document, info driver.ResponseInfo) error {
	curDoc, err := driver.ExtractCursorDocument(resp)
	if err != nil {
		return err
	}
	li.res, err = driver.NewCursorResponse(curDoc, info)
	return err
}

// execute runs this operations and returns an error if the operation did not execute successfully.
func (li *listIndexesOp) execute(ctx context.Context) error {
	if li.deployment == nil {
		return errors.New("the ListIndexes operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:         li.command,
		ProcessResponseFn: li.processResponse,

		Client:                    li.session,
		Clock:                     li.clock,
		CommandMonitor:            li.monitor,
		Database:                  li.database,
		Deployment:                li.deployment,
		Selector:                  li.selector,
		Crypt:                     li.crypt,
		Legacy:                    driver.LegacyListIndexes,
		RetryMode:                 li.retry,
		MaxAdaptiveRetries:        li.maxAdaptiveRetries,
		EnableOverloadRetargeting: li.enableOverloadRetargeting,
		Type:                      driver.Read,
		ServerAPI:                 li.serverAPI,
		Timeout:                   li.timeout,
		Name:                      driverutil.ListIndexesOp,
		Authenticator:             li.authenticator,
	}.Execute(ctx)
}

func (li *listIndexesOp) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "listIndexes", li.collection)
	cursorIdx, cursorDoc := bsoncore.AppendDocumentStart(nil)

	if li.batchSize != nil {
		cursorDoc = bsoncore.AppendInt32Element(cursorDoc, "batchSize", *li.batchSize)
	}
	cursorDoc, _ = bsoncore.AppendDocumentEnd(cursorDoc, cursorIdx)
	dst = bsoncore.AppendDocumentElement(dst, "cursor", cursorDoc)
	// Set rawData for 8.2+ servers.
	if li.rawData != nil && desc.WireVersion != nil && driverutil.VersionRangeIncludes(*desc.WireVersion, 27) {
		dst = bsoncore.AppendBooleanElement(dst, "rawData", *li.rawData)
	}

	return dst, nil
}
