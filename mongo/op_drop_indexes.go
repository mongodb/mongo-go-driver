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
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// dropIndexesOp performs a dropIndexes operation.
type dropIndexesOp struct {
	authenticator             driver.Authenticator
	index                     any
	session                   *session.Client
	clock                     *session.ClusterClock
	collection                string
	monitor                   *event.CommandMonitor
	maxAdaptiveRetries        uint
	enableOverloadRetargeting bool
	crypt                     driver.Crypt
	database                  string
	deployment                driver.Deployment
	selector                  description.ServerSelector
	writeConcern              *writeconcern.WriteConcern
	serverAPI                 *driver.ServerAPIOptions
	timeout                   *time.Duration
	rawData                   *bool
}

func (di *dropIndexesOp) processResponse(context.Context, bsoncore.Document, driver.ResponseInfo) error {
	return nil
}

// execute runs this operation and returns an error if the operation did not execute successfully.
func (di *dropIndexesOp) execute(ctx context.Context) error {
	if di.deployment == nil {
		return errors.New("the dropIndexes operation must have a Deployment set before execute can be called")
	}

	return driver.Operation{
		CommandFn:                 di.command,
		ProcessResponseFn:         di.processResponse,
		Client:                    di.session,
		Clock:                     di.clock,
		CommandMonitor:            di.monitor,
		MaxAdaptiveRetries:        di.maxAdaptiveRetries,
		EnableOverloadRetargeting: di.enableOverloadRetargeting,
		Crypt:                     di.crypt,
		Database:                  di.database,
		Deployment:                di.deployment,
		Selector:                  di.selector,
		WriteConcern:              di.writeConcern,
		ServerAPI:                 di.serverAPI,
		Timeout:                   di.timeout,
		Name:                      driverutil.DropIndexesOp,
		Authenticator:             di.authenticator,
		SendAfterClusterTime:      true,
	}.Execute(ctx)
}

func (di *dropIndexesOp) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "dropIndexes", di.collection)

	switch t := di.index.(type) {
	case string:
		dst = bsoncore.AppendStringElement(dst, "index", t)
	case bsoncore.Document:
		if di.index != nil {
			dst = bsoncore.AppendDocumentElement(dst, "index", t)
		}
	}
	// Set rawData for 8.2+ servers.
	if di.rawData != nil && desc.WireVersion != nil && driverutil.VersionRangeIncludes(*desc.WireVersion, 27) {
		dst = bsoncore.AppendBooleanElement(dst, "rawData", *di.rawData)
	}

	return dst, nil
}
