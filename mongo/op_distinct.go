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
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// distinctOp performs a distinct operation.
type distinctOp struct {
	authenticator             driver.Authenticator
	collation                 bsoncore.Document
	key                       *string
	query                     bsoncore.Document
	session                   *session.Client
	clock                     *session.ClusterClock
	collection                string
	comment                   bsoncore.Value
	hint                      bsoncore.Value
	monitor                   *event.CommandMonitor
	crypt                     driver.Crypt
	database                  string
	deployment                driver.Deployment
	readConcern               *readconcern.ReadConcern
	readPreference            *readpref.ReadPref
	selector                  description.ServerSelector
	retry                     *driver.RetryMode
	maxAdaptiveRetries        uint
	enableOverloadRetargeting bool
	res                       distinctResult
	serverAPI                 *driver.ServerAPIOptions
	timeout                   *time.Duration
	rawData                   *bool
}

// distinctResult represents a distinct result returned by the server.
type distinctResult struct {
	// The distinct values for the field.
	Values bsoncore.Value
}

func buildDistinctResult(response bsoncore.Document) (distinctResult, error) {
	elements, err := response.Elements()
	if err != nil {
		return distinctResult{}, err
	}
	dr := distinctResult{}
	for _, element := range elements {
		if element.Key() == "values" {
			dr.Values = element.Value()
		}
	}
	return dr, nil
}

// result returns the result of executing this operation.
func (d *distinctOp) result() distinctResult { return d.res }

func (d *distinctOp) processResponse(_ context.Context, resp bsoncore.Document, _ driver.ResponseInfo) error {
	var err error
	d.res, err = buildDistinctResult(resp)
	return err
}

// execute runs this operation and returns an error if the operation did not execute successfully.
func (d *distinctOp) execute(ctx context.Context) error {
	if d.deployment == nil {
		return errors.New("the distinct operation must have a Deployment set before execute can be called")
	}

	return driver.Operation{
		CommandFn:                 d.command,
		ProcessResponseFn:         d.processResponse,
		RetryMode:                 d.retry,
		MaxAdaptiveRetries:        d.maxAdaptiveRetries,
		EnableOverloadRetargeting: d.enableOverloadRetargeting,
		Type:                      driver.Read,
		Client:                    d.session,
		Clock:                     d.clock,
		CommandMonitor:            d.monitor,
		Crypt:                     d.crypt,
		Database:                  d.database,
		Deployment:                d.deployment,
		ReadConcern:               d.readConcern,
		ReadPreference:            d.readPreference,
		Selector:                  d.selector,
		ServerAPI:                 d.serverAPI,
		Timeout:                   d.timeout,
		Name:                      driverutil.DistinctOp,
		Authenticator:             d.authenticator,
	}.Execute(ctx)
}

func (d *distinctOp) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "distinct", d.collection)
	if d.collation != nil {
		if desc.WireVersion == nil || !driverutil.VersionRangeIncludes(*desc.WireVersion, 5) {
			return nil, errors.New("the 'collation' command parameter requires a minimum server wire version of 5")
		}
		dst = bsoncore.AppendDocumentElement(dst, "collation", d.collation)
	}
	if d.comment.Type != bsoncore.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "comment", d.comment)
	}
	if d.hint.Type != bsoncore.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "hint", d.hint)
	}
	if d.key != nil {
		dst = bsoncore.AppendStringElement(dst, "key", *d.key)
	}
	if d.query != nil {
		dst = bsoncore.AppendDocumentElement(dst, "query", d.query)
	}
	// Set rawData for 8.2+ servers.
	if d.rawData != nil && desc.WireVersion != nil && driverutil.VersionRangeIncludes(*desc.WireVersion, 27) {
		dst = bsoncore.AppendBooleanElement(dst, "rawData", *d.rawData)
	}
	return dst, nil
}
