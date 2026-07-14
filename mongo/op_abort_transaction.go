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
	"go.mongodb.org/mongo-driver/v2/internal/logger"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// abortTransactionOp performs an abortTransaction operation.
type abortTransactionOp struct {
	authenticator             driver.Authenticator
	recoveryToken             bsoncore.Document
	session                   *session.Client
	clock                     *session.ClusterClock
	monitor                   *event.CommandMonitor
	database                  string
	deployment                driver.Deployment
	selector                  description.ServerSelector
	writeConcern              *writeconcern.WriteConcern
	retry                     *driver.RetryMode
	maxAdaptiveRetries        uint
	enableOverloadRetargeting bool
	serverAPI                 *driver.ServerAPIOptions
	logger                    *logger.Logger
}

func (at *abortTransactionOp) processResponse(context.Context, bsoncore.Document, driver.ResponseInfo) error {
	return nil
}

// Execute runs this operation and returns an error if the operation did not execute successfully.
func (at *abortTransactionOp) Execute(ctx context.Context) error {
	if at.deployment == nil {
		return errors.New("the abortTransaction operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:                 at.command,
		ProcessResponseFn:         at.processResponse,
		RetryMode:                 at.retry,
		Type:                      driver.Write,
		Client:                    at.session,
		Clock:                     at.clock,
		CommandMonitor:            at.monitor,
		MaxAdaptiveRetries:        at.maxAdaptiveRetries,
		EnableOverloadRetargeting: at.enableOverloadRetargeting,
		Database:                  at.database,
		Deployment:                at.deployment,
		Selector:                  at.selector,
		WriteConcern:              at.writeConcern,
		ServerAPI:                 at.serverAPI,
		Name:                      driverutil.AbortTransactionOp,
		Authenticator:             at.authenticator,
		Logger:                    at.logger,
	}.Execute(ctx)
}

func (at *abortTransactionOp) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendInt32Element(dst, "abortTransaction", 1)
	if at.recoveryToken != nil {
		dst = bsoncore.AppendDocumentElement(dst, "recoveryToken", at.recoveryToken)
	}
	return dst, nil
}
