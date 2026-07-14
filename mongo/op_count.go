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
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// countOp performs a count operation.
type countOp struct {
	authenticator             driver.Authenticator
	session                   *session.Client
	clock                     *session.ClusterClock
	collection                string
	comment                   bsoncore.Value
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
	result                    countResult
	serverAPI                 *driver.ServerAPIOptions
	timeout                   *time.Duration
	rawData                   *bool
}

// countResult represents a count result returned by the server.
type countResult struct {
	// The number of documents found
	N int64
}

func buildCountResult(response bsoncore.Document) (countResult, error) {
	elements, err := response.Elements()
	if err != nil {
		return countResult{}, err
	}
	cr := countResult{}
	for _, element := range elements {
		switch element.Key() {
		case "n": // for count using original command
			var ok bool
			cr.N, ok = element.Value().AsInt64OK()
			if !ok {
				return cr, fmt.Errorf("response field 'n' is type int64, but received BSON type %s",
					element.Value().Type)
			}
		case "cursor": // for count using aggregate with $collStats
			firstBatch, err := element.Value().Document().LookupErr("firstBatch")
			if err != nil {
				return cr, err
			}

			// get count value from first batch
			val := firstBatch.Array().Index(0)
			count, err := val.Document().LookupErr("n")
			if err != nil {
				return cr, err
			}

			// use count as Int64 for result
			var ok bool
			cr.N, ok = count.AsInt64OK()
			if !ok {
				return cr, fmt.Errorf("response field 'n' is type int64, but received BSON type %s",
					element.Value().Type)
			}
		}
	}
	return cr, nil
}

// Result returns the result of executing this operation.
func (c *countOp) Result() countResult { return c.result }

func (c *countOp) processResponse(_ context.Context, resp bsoncore.Document, _ driver.ResponseInfo) error {
	var err error
	c.result, err = buildCountResult(resp)
	return err
}

// Execute runs this operation and returns an error if the operation did not execute successfully.
func (c *countOp) Execute(ctx context.Context) error {
	if c.deployment == nil {
		return errors.New("the count operation must have a Deployment set before Execute can be called")
	}

	err := driver.Operation{
		CommandFn:                 c.command,
		ProcessResponseFn:         c.processResponse,
		RetryMode:                 c.retry,
		MaxAdaptiveRetries:        c.maxAdaptiveRetries,
		EnableOverloadRetargeting: c.enableOverloadRetargeting,
		Type:                      driver.Read,
		Client:                    c.session,
		Clock:                     c.clock,
		CommandMonitor:            c.monitor,
		Crypt:                     c.crypt,
		Database:                  c.database,
		Deployment:                c.deployment,
		ReadConcern:               c.readConcern,
		ReadPreference:            c.readPreference,
		Selector:                  c.selector,
		ServerAPI:                 c.serverAPI,
		Timeout:                   c.timeout,
		Name:                      driverutil.CountOp,
		Authenticator:             c.authenticator,
	}.Execute(ctx)
	// Swallow error if NamespaceNotFound(26) is returned from aggregate on non-existent namespace
	if err != nil {
		dErr, ok := err.(driver.Error)
		if ok && dErr.Code == 26 {
			err = nil
		}
	}
	return err
}

func (c *countOp) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "count", c.collection)
	if c.comment.Type != bsoncore.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "comment", c.comment)
	}
	// Set rawData for 8.2+ servers.
	if c.rawData != nil && desc.WireVersion != nil && driverutil.VersionRangeIncludes(*desc.WireVersion, 27) {
		dst = bsoncore.AppendBooleanElement(dst, "rawData", *c.rawData)
	}
	return dst, nil
}
