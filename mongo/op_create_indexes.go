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

// createIndexesOp performs a createIndexes operation.
type createIndexesOp struct {
	authenticator             driver.Authenticator
	commitQuorum              bsoncore.Value
	indexes                   bsoncore.Document
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

func (ci *createIndexesOp) processResponse(context.Context, bsoncore.Document, driver.ResponseInfo) error {
	return nil
}

// execute runs this operations and returns an error if the operation did not execute successfully.
func (ci *createIndexesOp) execute(ctx context.Context) error {
	if ci.deployment == nil {
		return errors.New("the createIndexes operation must have a Deployment set before execute can be called")
	}

	return driver.Operation{
		CommandFn:                 ci.command,
		ProcessResponseFn:         ci.processResponse,
		Client:                    ci.session,
		Clock:                     ci.clock,
		CommandMonitor:            ci.monitor,
		MaxAdaptiveRetries:        ci.maxAdaptiveRetries,
		EnableOverloadRetargeting: ci.enableOverloadRetargeting,
		Crypt:                     ci.crypt,
		Database:                  ci.database,
		Deployment:                ci.deployment,
		Selector:                  ci.selector,
		WriteConcern:              ci.writeConcern,
		ServerAPI:                 ci.serverAPI,
		Timeout:                   ci.timeout,
		Name:                      driverutil.CreateIndexesOp,
		Authenticator:             ci.authenticator,
		SendAfterClusterTime:      true,
	}.Execute(ctx)
}

func (ci *createIndexesOp) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "createIndexes", ci.collection)
	if ci.commitQuorum.Type != bsoncore.Type(0) {
		if desc.WireVersion == nil || !driverutil.VersionRangeIncludes(*desc.WireVersion, 9) {
			return nil, errors.New("the 'commitQuorum' command parameter requires a minimum server wire version of 9")
		}
		dst = bsoncore.AppendValueElement(dst, "commitQuorum", ci.commitQuorum)
	}
	if ci.indexes != nil {
		dst = bsoncore.AppendArrayElement(dst, "indexes", ci.indexes)
	}
	// Set rawData for 8.2+ servers.
	if ci.rawData != nil && desc.WireVersion != nil && driverutil.VersionRangeIncludes(*desc.WireVersion, 27) {
		dst = bsoncore.AppendBooleanElement(dst, "rawData", *ci.rawData)
	}
	return dst, nil
}
