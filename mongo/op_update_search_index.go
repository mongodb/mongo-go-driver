// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// updateSearchIndexOp performs a updateSearchIndex operation.
type updateSearchIndexOp struct {
	authenticator driver.Authenticator
	index         string
	definition    bsoncore.Document
	session       *session.Client
	clock         *session.ClusterClock
	collection    string
	monitor       *event.CommandMonitor
	crypt         driver.Crypt
	database      string
	deployment    driver.Deployment
	selector      description.ServerSelector
	res           updateSearchIndexResult
	serverAPI     *driver.ServerAPIOptions
	timeout       *time.Duration
}

// updateSearchIndexResult represents a single index in the updateSearchIndexResult result.
type updateSearchIndexResult struct {
	Ok int32
}

func buildUpdateSearchIndexResult(response bsoncore.Document) (updateSearchIndexResult, error) {
	elements, err := response.Elements()
	if err != nil {
		return updateSearchIndexResult{}, err
	}
	usir := updateSearchIndexResult{}
	for _, element := range elements {
		if element.Key() == "ok" {
			var ok bool
			usir.Ok, ok = element.Value().AsInt32OK()
			if !ok {
				return usir, fmt.Errorf("response field 'ok' is type int32, but received BSON type %s", element.Value().Type)
			}
		}
	}
	return usir, nil
}

func (usi *updateSearchIndexOp) processResponse(_ context.Context, resp bsoncore.Document, _ driver.ResponseInfo) error {
	var err error
	usi.res, err = buildUpdateSearchIndexResult(resp)
	return err
}

// execute runs this operation and returns an error if the operation did not execute successfully.
func (usi *updateSearchIndexOp) execute(ctx context.Context) error {
	if usi.deployment == nil {
		return errors.New("the updateSearchIndex operation must have a Deployment set before execute can be called")
	}

	return driver.Operation{
		CommandFn:         usi.command,
		ProcessResponseFn: usi.processResponse,
		Client:            usi.session,
		Clock:             usi.clock,
		CommandMonitor:    usi.monitor,
		Crypt:             usi.crypt,
		Database:          usi.database,
		Deployment:        usi.deployment,
		Selector:          usi.selector,
		ServerAPI:         usi.serverAPI,
		Timeout:           usi.timeout,
		Authenticator:     usi.authenticator,
	}.Execute(ctx)
}

func (usi *updateSearchIndexOp) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "updateSearchIndex", usi.collection)
	dst = bsoncore.AppendStringElement(dst, "name", usi.index)
	dst = bsoncore.AppendDocumentElement(dst, "definition", usi.definition)
	return dst, nil
}
