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

// commitTransactionOp attempts to commit a transaction.
type commitTransactionOp struct {
	authenticator             driver.Authenticator
	recoveryToken             bsoncore.Document
	session                   *session.Client
	clock                     *session.ClusterClock
	monitor                   *event.CommandMonitor
	crypt                     driver.Crypt
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

func (ct *commitTransactionOp) processResponse(context.Context, bsoncore.Document, driver.ResponseInfo) error {
	return nil
}

// Execute runs this operations and returns an error if the operation did not execute successfully.
func (ct *commitTransactionOp) Execute(ctx context.Context) error {
	if ct.deployment == nil {
		return errors.New("the commitTransaction operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:                 ct.command,
		ProcessResponseFn:         ct.processResponse,
		RetryMode:                 ct.retry,
		Type:                      driver.Write,
		Client:                    ct.session,
		Clock:                     ct.clock,
		CommandMonitor:            ct.monitor,
		MaxAdaptiveRetries:        ct.maxAdaptiveRetries,
		EnableOverloadRetargeting: ct.enableOverloadRetargeting,
		Crypt:                     ct.crypt,
		Database:                  ct.database,
		Deployment:                ct.deployment,
		Selector:                  ct.selector,
		WriteConcern:              ct.writeConcern,
		ServerAPI:                 ct.serverAPI,
		Name:                      driverutil.CommitTransactionOp,
		Authenticator:             ct.authenticator,
		Logger:                    ct.logger,
	}.Execute(ctx)
}

func (ct *commitTransactionOp) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendInt32Element(dst, "commitTransaction", 1)
	if ct.recoveryToken != nil {
		dst = bsoncore.AppendDocumentElement(dst, "recoveryToken", ct.recoveryToken)
	}
	return dst, nil
}
