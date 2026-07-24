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
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// dropDatabaseOp performs a dropDatabase operation
type dropDatabaseOp struct {
	authenticator driver.Authenticator
	session       *session.Client
	clock         *session.ClusterClock
	monitor       *event.CommandMonitor
	crypt         driver.Crypt
	database      string
	deployment    driver.Deployment
	selector      description.ServerSelector
	writeConcern  *writeconcern.WriteConcern
	serverAPI     *driver.ServerAPIOptions
}

// execute runs this operation and returns an error if the operation did not execute successfully.
func (dd *dropDatabaseOp) execute(ctx context.Context) error {
	if dd.deployment == nil {
		return errors.New("the dropDatabase operation must have a Deployment set before execute can be called")
	}

	return driver.Operation{
		CommandFn:            dd.command,
		Client:               dd.session,
		Clock:                dd.clock,
		CommandMonitor:       dd.monitor,
		Crypt:                dd.crypt,
		Database:             dd.database,
		Deployment:           dd.deployment,
		Selector:             dd.selector,
		WriteConcern:         dd.writeConcern,
		ServerAPI:            dd.serverAPI,
		Name:                 driverutil.DropDatabaseOp,
		Authenticator:        dd.authenticator,
		SendAfterClusterTime: true,
	}.Execute(ctx)
}

func (dd *dropDatabaseOp) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendInt32Element(dst, "dropDatabase", 1)
	return dst, nil
}
