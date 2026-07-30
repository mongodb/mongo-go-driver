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

// dropCollectionOp performs a drop operation.
type dropCollectionOp struct {
	authenticator driver.Authenticator
	session       *session.Client
	clock         *session.ClusterClock
	collection    string
	monitor       *event.CommandMonitor
	crypt         driver.Crypt
	database      string
	deployment    driver.Deployment
	selector      description.ServerSelector
	writeConcern  *writeconcern.WriteConcern
	serverAPI     *driver.ServerAPIOptions
	timeout       *time.Duration
}

// execute runs this operation and returns an error if the operation did not execute successfully.
func (dc *dropCollectionOp) execute(ctx context.Context) error {
	if dc.deployment == nil {
		return errors.New("the dropCollection operation must have a Deployment set before execute can be called")
	}

	return driver.Operation{
		CommandFn:            dc.command,
		Client:               dc.session,
		Clock:                dc.clock,
		CommandMonitor:       dc.monitor,
		Crypt:                dc.crypt,
		Database:             dc.database,
		Deployment:           dc.deployment,
		Selector:             dc.selector,
		WriteConcern:         dc.writeConcern,
		ServerAPI:            dc.serverAPI,
		Timeout:              dc.timeout,
		Name:                 driverutil.DropOp,
		Authenticator:        dc.authenticator,
		SendAfterClusterTime: true,
	}.Execute(ctx)
}

func (dc *dropCollectionOp) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "drop", dc.collection)
	return dst, nil
}
