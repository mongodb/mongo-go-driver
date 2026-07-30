// Copyright (C) MongoDB, Inc. 2019-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/driverutil"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// endSessionsOp performs an endSessions operation.
type endSessionsOp struct {
	authenticator             driver.Authenticator
	sessionIDs                bsoncore.Document
	session                   *session.Client
	clock                     *session.ClusterClock
	monitor                   *event.CommandMonitor
	crypt                     driver.Crypt
	database                  string
	deployment                driver.Deployment
	selector                  description.ServerSelector
	serverAPI                 *driver.ServerAPIOptions
	maxAdaptiveRetries        uint
	enableOverloadRetargeting bool
}

func (es *endSessionsOp) processResponse(context.Context, bsoncore.Document, driver.ResponseInfo) error {
	return nil
}

// execute runs this operation and returns an error if the operation did not execute successfully.
func (es *endSessionsOp) execute(ctx context.Context) error {
	if es.deployment == nil {
		return errors.New("the endSessions operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:                 es.command,
		ProcessResponseFn:         es.processResponse,
		Client:                    es.session,
		Clock:                     es.clock,
		CommandMonitor:            es.monitor,
		MaxAdaptiveRetries:        es.maxAdaptiveRetries,
		EnableOverloadRetargeting: es.enableOverloadRetargeting,
		Crypt:                     es.crypt,
		Database:                  es.database,
		Deployment:                es.deployment,
		Selector:                  es.selector,
		ServerAPI:                 es.serverAPI,
		Name:                      driverutil.EndSessionsOp,
		Authenticator:             es.authenticator,
	}.Execute(ctx)
}

func (es *endSessionsOp) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	if es.sessionIDs != nil {
		dst = bsoncore.AppendArrayElement(dst, "endSessions", es.sessionIDs)
	}
	return dst, nil
}
