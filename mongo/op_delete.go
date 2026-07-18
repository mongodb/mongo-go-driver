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
	"go.mongodb.org/mongo-driver/v2/internal/logger"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// deleteOp performs a delete operation.
type deleteOp struct {
	authenticator             driver.Authenticator
	comment                   bsoncore.Value
	deletes                   []bsoncore.Document
	ordered                   *bool
	session                   *session.Client
	clock                     *session.ClusterClock
	collection                string
	monitor                   *event.CommandMonitor
	crypt                     driver.Crypt
	database                  string
	deployment                driver.Deployment
	selector                  description.ServerSelector
	writeConcern              *writeconcern.WriteConcern
	retry                     *driver.RetryMode
	maxAdaptiveRetries        uint
	enableOverloadRetargeting bool
	hint                      *bool
	res                       deleteResult
	serverAPI                 *driver.ServerAPIOptions
	let                       bsoncore.Document
	timeout                   *time.Duration
	rawData                   *bool
	logger                    *logger.Logger
}

// deleteResult represents a delete result returned by the server.
type deleteResult struct {
	// Number of documents successfully deleted.
	N int64
}

func buildDeleteResult(response bsoncore.Document) (deleteResult, error) {
	elements, err := response.Elements()
	if err != nil {
		return deleteResult{}, err
	}
	dr := deleteResult{}
	for _, element := range elements {
		if element.Key() == "n" {
			var ok bool
			dr.N, ok = element.Value().AsInt64OK()
			if !ok {
				return dr, fmt.Errorf("response field 'n' is type int32 or int64, but received BSON type %s", element.Value().Type)
			}
		}
	}
	return dr, nil
}

// result returns the result of executing this operation.
func (d *deleteOp) result() deleteResult { return d.res }

func (d *deleteOp) processResponse(_ context.Context, resp bsoncore.Document, _ driver.ResponseInfo) error {
	dr, err := buildDeleteResult(resp)
	d.res.N += dr.N
	return err
}

// execute runs this operation and returns an error if the operation did not execute successfully.
func (d *deleteOp) execute(ctx context.Context) error {
	if d.deployment == nil {
		return errors.New("the delete operation must have a Deployment set before execute can be called")
	}
	batches := &driver.Batches{
		Identifier: "deletes",
		Documents:  d.deletes,
		Ordered:    d.ordered,
	}

	return driver.Operation{
		CommandFn:                 d.command,
		ProcessResponseFn:         d.processResponse,
		Batches:                   batches,
		RetryMode:                 d.retry,
		MaxAdaptiveRetries:        d.maxAdaptiveRetries,
		EnableOverloadRetargeting: d.enableOverloadRetargeting,
		Type:                      driver.Write,
		Client:                    d.session,
		Clock:                     d.clock,
		CommandMonitor:            d.monitor,
		Crypt:                     d.crypt,
		Database:                  d.database,
		Deployment:                d.deployment,
		Selector:                  d.selector,
		WriteConcern:              d.writeConcern,
		ServerAPI:                 d.serverAPI,
		Timeout:                   d.timeout,
		Logger:                    d.logger,
		Name:                      driverutil.DeleteOp,
		Authenticator:             d.authenticator,
		SendAfterClusterTime:      true,
	}.Execute(ctx)
}

func (d *deleteOp) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "delete", d.collection)
	if d.comment.Type != bsoncore.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "comment", d.comment)
	}
	if d.ordered != nil {
		dst = bsoncore.AppendBooleanElement(dst, "ordered", *d.ordered)
	}
	if d.hint != nil && *d.hint {
		if desc.WireVersion == nil || !driverutil.VersionRangeIncludes(*desc.WireVersion, 5) {
			return nil, errors.New("the 'hint' command parameter requires a minimum server wire version of 5")
		}
		if !d.writeConcern.Acknowledged() {
			return nil, errUnacknowledgedHint
		}
	}
	if d.let != nil {
		dst = bsoncore.AppendDocumentElement(dst, "let", d.let)
	}
	// Set rawData for 8.2+ servers.
	if d.rawData != nil && desc.WireVersion != nil && driverutil.VersionRangeIncludes(*desc.WireVersion, 27) {
		dst = bsoncore.AppendBooleanElement(dst, "rawData", *d.rawData)
	}
	return dst, nil
}
