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

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// createSearchIndexesOp performs a createSearchIndexes operation.
type createSearchIndexesOp struct {
	authenticator driver.Authenticator
	indexes       bsoncore.Document
	session       *session.Client
	clock         *session.ClusterClock
	collection    string
	monitor       *event.CommandMonitor
	crypt         driver.Crypt
	database      string
	deployment    driver.Deployment
	selector      description.ServerSelector
	result        createSearchIndexesResult
	serverAPI     *driver.ServerAPIOptions
	timeout       *time.Duration
}

// createSearchIndexResult represents a single search index result in createSearchIndexesResult.
type createSearchIndexResult struct {
	Name string
}

// createSearchIndexesResult represents a createSearchIndexes result returned by the server.
type createSearchIndexesResult struct {
	IndexesCreated []createSearchIndexResult
}

func buildCreateSearchIndexesResult(response bsoncore.Document) (createSearchIndexesResult, error) {
	elements, err := response.Elements()
	if err != nil {
		return createSearchIndexesResult{}, err
	}
	csir := createSearchIndexesResult{}
	for _, element := range elements {
		switch element.Key() {
		case "indexesCreated":
			arr, ok := element.Value().ArrayOK()
			if !ok {
				return csir, fmt.Errorf("response field 'indexesCreated' is type array, but received BSON type %s", element.Value().Type)
			}

			var values []bsoncore.Value
			values, err = arr.Values()
			if err != nil {
				break
			}

			for _, val := range values {
				valDoc, ok := val.DocumentOK()
				if !ok {
					return csir, fmt.Errorf("indexesCreated value is type document, but received BSON type %s", val.Type)
				}
				var indexesCreated createSearchIndexResult
				if err = bson.Unmarshal(valDoc, &indexesCreated); err != nil {
					return csir, err
				}
				csir.IndexesCreated = append(csir.IndexesCreated, indexesCreated)
			}
		}
	}
	return csir, nil
}

// Result returns the result of executing this operation.
func (csi *createSearchIndexesOp) Result() createSearchIndexesResult { return csi.result }

func (csi *createSearchIndexesOp) processResponse(_ context.Context, resp bsoncore.Document, _ driver.ResponseInfo) error {
	var err error
	csi.result, err = buildCreateSearchIndexesResult(resp)
	return err
}

// Execute runs this operations and returns an error if the operation did not execute successfully.
func (csi *createSearchIndexesOp) Execute(ctx context.Context) error {
	if csi.deployment == nil {
		return errors.New("the CreateSearchIndexes operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:         csi.command,
		ProcessResponseFn: csi.processResponse,
		Client:            csi.session,
		Clock:             csi.clock,
		CommandMonitor:    csi.monitor,
		Crypt:             csi.crypt,
		Database:          csi.database,
		Deployment:        csi.deployment,
		Selector:          csi.selector,
		ServerAPI:         csi.serverAPI,
		Timeout:           csi.timeout,
		Authenticator:     csi.authenticator,
	}.Execute(ctx)
}

func (csi *createSearchIndexesOp) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "createSearchIndexes", csi.collection)
	if csi.indexes != nil {
		dst = bsoncore.AppendArrayElement(dst, "indexes", csi.indexes)
	}
	return dst, nil
}
