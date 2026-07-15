// Copyright (C) MongoDB, Inc. 2019-present.
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
	"go.mongodb.org/mongo-driver/v2/internal/driverutil"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// createIndexesOp performs a createIndexes operation.
type createIndexesOp struct {
	authenticator             driver.Authenticator
	commitQuorum              bsoncore.Value
	indexes                   bsoncore.Document
	session                   *session.Client
	clock                     *session.ClusterClock
	collection                string
	monitor                   *event.CommandMonitor
	maxAdaptiveRetries        uint
	enableOverloadRetargeting bool
	crypt                     driver.Crypt
	database                  string
	deployment                driver.Deployment
	selector                  description.ServerSelector
	writeConcern              *writeconcern.WriteConcern
	result                    createIndexesResult
	serverAPI                 *driver.ServerAPIOptions
	timeout                   *time.Duration
	rawData                   *bool
}

// createIndexesResult represents a createIndexes result returned by the server.
type createIndexesResult struct {
	// If the collection was created automatically.
	CreatedCollectionAutomatically bool
	// The number of indexes existing after this command.
	IndexesAfter int32
	// The number of indexes existing before this command.
	IndexesBefore int32
}

func buildCreateIndexesResult(response bsoncore.Document) (createIndexesResult, error) {
	elements, err := response.Elements()
	if err != nil {
		return createIndexesResult{}, err
	}
	cir := createIndexesResult{}
	for _, element := range elements {
		switch element.Key() {
		case "createdCollectionAutomatically":
			var ok bool
			cir.CreatedCollectionAutomatically, ok = element.Value().BooleanOK()
			if !ok {
				return cir, fmt.Errorf("response field 'createdCollectionAutomatically' is type bool, but received BSON type %s", element.Value().Type)
			}
		case "indexesAfter":
			var ok bool
			cir.IndexesAfter, ok = element.Value().AsInt32OK()
			if !ok {
				return cir, fmt.Errorf("response field 'indexesAfter' is type int32, but received BSON type %s", element.Value().Type)
			}
		case "indexesBefore":
			var ok bool
			cir.IndexesBefore, ok = element.Value().AsInt32OK()
			if !ok {
				return cir, fmt.Errorf("response field 'indexesBefore' is type int32, but received BSON type %s", element.Value().Type)
			}
		}
	}
	return cir, nil
}

// Result returns the result of executing this operation.
func (ci *createIndexesOp) Result() createIndexesResult { return ci.result }

func (ci *createIndexesOp) processResponse(_ context.Context, resp bsoncore.Document, _ driver.ResponseInfo) error {
	var err error
	ci.result, err = buildCreateIndexesResult(resp)
	return err
}

// Execute runs this operations and returns an error if the operation did not execute successfully.
func (ci *createIndexesOp) Execute(ctx context.Context) error {
	if ci.deployment == nil {
		return errors.New("the CreateIndexes operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:                 ci.command,
		ProcessResponseFn:         ci.processResponse,
		Client:                    ci.session,
		Clock:                     ci.clock,
		CommandMonitor:            ci.monitor,
		MaxAdaptiveRetries:        ci.maxAdaptiveRetries,
		EnableOverloadRetargeting: ci.enableOverloadRetargeting,
		Crypt:                     ci.crypt,
		Database:                  ci.database,
		Deployment:                ci.deployment,
		Selector:                  ci.selector,
		WriteConcern:              ci.writeConcern,
		ServerAPI:                 ci.serverAPI,
		Timeout:                   ci.timeout,
		Name:                      driverutil.CreateIndexesOp,
		Authenticator:             ci.authenticator,
		SendAfterClusterTime:      true,
	}.Execute(ctx)
}

func (ci *createIndexesOp) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "createIndexes", ci.collection)
	if ci.commitQuorum.Type != bsoncore.Type(0) {
		if desc.WireVersion == nil || !driverutil.VersionRangeIncludes(*desc.WireVersion, 9) {
			return nil, errors.New("the 'commitQuorum' command parameter requires a minimum server wire version of 9")
		}
		dst = bsoncore.AppendValueElement(dst, "commitQuorum", ci.commitQuorum)
	}
	if ci.indexes != nil {
		dst = bsoncore.AppendArrayElement(dst, "indexes", ci.indexes)
	}
	// Set rawData for 8.2+ servers.
	if ci.rawData != nil && desc.WireVersion != nil && driverutil.VersionRangeIncludes(*desc.WireVersion, 27) {
		dst = bsoncore.AppendBooleanElement(dst, "rawData", *ci.rawData)
	}
	return dst, nil
}
