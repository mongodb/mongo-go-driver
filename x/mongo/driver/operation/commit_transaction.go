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

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/driverutil"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
)

// CommitTransaction attempts to commit a transaction.
type CommitTransaction struct {
	// MaxTime is the maximum amount of time to allow the query to run on the
	// server.
	MaxTime *time.Duration

	// RecoveryToken is the recovery token to use when committing or aborting a
	// sharded transaction.
	RecoveryToken bsoncore.Document

	// Session is the session for this operation.
	Session *session.Client

	// Clock is the cluster clock for this operation.
	Clock *session.ClusterClock

	// Monitor is the monitor to use for APM events.
	Monitor *event.CommandMonitor

	// Crypt is the Crypt object to use for automatic encryption and decryption.
	Crypt driver.Crypt

	// Deployment is the deployment to use for this operation.
	Deployment driver.Deployment

	// Selector is the selector used to retrieve a server.
	Selector description.ServerSelector

	// WriteConcern is the write concern for this operation.
	WriteConcern *writeconcern.WriteConcern

	// Retry enables retryable mode for this operation. Retries are handled
	// automatically in driver.Operation.Execute based on how the operation is
	// set.
	Retry *driver.RetryMode

	// ServerAPI is the server API version for this operation.
	ServerAPI *driver.ServerAPIOptions
}

func (ct *CommitTransaction) processResponse(driver.ResponseInfo) error {
	var err error
	return err
}

// Execute runs this operations and returns an error if the operation did not execute successfully.
func (ct *CommitTransaction) Execute(ctx context.Context) error {
	if ct.Deployment == nil {
		return errors.New("the CommitTransaction operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:         ct.command,
		ProcessResponseFn: ct.processResponse,
		RetryMode:         ct.Retry,
		Type:              driver.Write,
		Client:            ct.Session,
		Clock:             ct.Clock,
		CommandMonitor:    ct.Monitor,
		Crypt:             ct.Crypt,
		Database:          "admin",
		Deployment:        ct.Deployment,
		MaxTime:           ct.MaxTime,
		Selector:          ct.Selector,
		WriteConcern:      ct.WriteConcern,
		ServerAPI:         ct.ServerAPI,
		Name:              driverutil.CommitTransactionOp,
	}.Execute(ctx)

}

func (ct *CommitTransaction) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendInt32Element(dst, "commitTransaction", 1)
	if ct.RecoveryToken != nil {
		dst = bsoncore.AppendDocumentElement(dst, "recoveryToken", ct.RecoveryToken)
	}
	return dst, nil
}
