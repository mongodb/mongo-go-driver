// Copyright (C) MongoDB, Inc. 2023-present.
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
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// dropSearchIndexOp performs an dropSearchIndex operation.
type dropSearchIndexOp struct {
	authenticator driver.Authenticator
	index         string
	session       *session.Client
	clock         *session.ClusterClock
	collection    string
	monitor       *event.CommandMonitor
	crypt         driver.Crypt
	database      string
	deployment    driver.Deployment
	selector      description.ServerSelector
	serverAPI     *driver.ServerAPIOptions
	timeout       *time.Duration
}

func (dsi *dropSearchIndexOp) processResponse(context.Context, bsoncore.Document, driver.ResponseInfo) error {
	return nil
}

// execute runs this operation and returns an error if the operation did not execute successfully.
func (dsi *dropSearchIndexOp) execute(ctx context.Context) error {
	if dsi.deployment == nil {
		return errors.New("the dropSearchIndex operation must have a Deployment set before execute can be called")
	}

	return driver.Operation{
		CommandFn:         dsi.command,
		ProcessResponseFn: dsi.processResponse,
		Client:            dsi.session,
		Clock:             dsi.clock,
		CommandMonitor:    dsi.monitor,
		Crypt:             dsi.crypt,
		Database:          dsi.database,
		Deployment:        dsi.deployment,
		Selector:          dsi.selector,
		ServerAPI:         dsi.serverAPI,
		Timeout:           dsi.timeout,
		Authenticator:     dsi.authenticator,
	}.Execute(ctx)
}

func (dsi *dropSearchIndexOp) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "dropSearchIndex", dsi.collection)
	dst = bsoncore.AppendStringElement(dst, "name", dsi.index)
	return dst, nil
}
