// Copyright (C) MongoDB, Inc. 2019-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package operation

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/driverutil"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
)

// AbortTransaction performs an abortTransaction operation.
type AbortTransaction struct {
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

func (at *AbortTransaction) processResponse(driver.ResponseInfo) error {
	var err error
	return err
}

// Execute runs this operations and returns an error if the operation did not execute successfully.
func (at *AbortTransaction) Execute(ctx context.Context) error {
	if at.Deployment == nil {
		return errors.New("the AbortTransaction operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:         at.command,
		ProcessResponseFn: at.processResponse,
		RetryMode:         at.Retry,
		Type:              driver.Write,
		Client:            at.Session,
		Clock:             at.Clock,
		CommandMonitor:    at.Monitor,
		Crypt:             at.Crypt,
		Database:          "admin",
		Deployment:        at.Deployment,
		Selector:          at.Selector,
		WriteConcern:      at.WriteConcern,
		ServerAPI:         at.ServerAPI,
		Name:              driverutil.AbortTransactionOp,
	}.Execute(ctx)

}

func (at *AbortTransaction) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendInt32Element(dst, "abortTransaction", 1)
	if at.RecoveryToken != nil {
		dst = bsoncore.AppendDocumentElement(dst, "recoveryToken", at.RecoveryToken)
	}
	return dst, nil
}
