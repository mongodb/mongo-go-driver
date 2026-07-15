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

// createOp performs a create operation.
type createOp struct {
	authenticator                driver.Authenticator
	capped                       *bool
	collation                    bsoncore.Document
	changeStreamPreAndPostImages bsoncore.Document
	collectionName               *string
	indexOptionDefaults          bsoncore.Document
	max                          *int64
	pipeline                     bsoncore.Document
	size                         *int64
	storageEngine                bsoncore.Document
	validationAction             *string
	validationLevel              *string
	validator                    bsoncore.Document
	viewOn                       *string
	session                      *session.Client
	clock                        *session.ClusterClock
	monitor                      *event.CommandMonitor
	crypt                        driver.Crypt
	database                     string
	deployment                   driver.Deployment
	selector                     description.ServerSelector
	writeConcern                 *writeconcern.WriteConcern
	serverAPI                    *driver.ServerAPIOptions
	expireAfterSeconds           *int64
	timeSeries                   bsoncore.Document
	encryptedFields              bsoncore.Document
	clusteredIndex               bsoncore.Document
}

func (c *createOp) processResponse(context.Context, bsoncore.Document, driver.ResponseInfo) error {
	return nil
}

// Execute runs this operations and returns an error if the operation did not execute successfully.
func (c *createOp) Execute(ctx context.Context) error {
	if c.deployment == nil {
		return errors.New("the Create operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:            c.command,
		ProcessResponseFn:    c.processResponse,
		Client:               c.session,
		Clock:                c.clock,
		CommandMonitor:       c.monitor,
		Crypt:                c.crypt,
		Database:             c.database,
		Deployment:           c.deployment,
		Selector:             c.selector,
		WriteConcern:         c.writeConcern,
		ServerAPI:            c.serverAPI,
		Authenticator:        c.authenticator,
		SendAfterClusterTime: true,
	}.Execute(ctx)
}

func (c *createOp) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	if c.collectionName != nil {
		dst = bsoncore.AppendStringElement(dst, "create", *c.collectionName)
	}
	if c.capped != nil {
		dst = bsoncore.AppendBooleanElement(dst, "capped", *c.capped)
	}
	if c.changeStreamPreAndPostImages != nil {
		dst = bsoncore.AppendDocumentElement(dst, "changeStreamPreAndPostImages", c.changeStreamPreAndPostImages)
	}
	if c.collation != nil {
		if desc.WireVersion == nil || !driverutil.VersionRangeIncludes(*desc.WireVersion, 5) {
			return nil, errors.New("the 'collation' command parameter requires a minimum server wire version of 5")
		}
		dst = bsoncore.AppendDocumentElement(dst, "collation", c.collation)
	}
	if c.indexOptionDefaults != nil {
		dst = bsoncore.AppendDocumentElement(dst, "indexOptionDefaults", c.indexOptionDefaults)
	}
	if c.max != nil {
		dst = bsoncore.AppendInt64Element(dst, "max", *c.max)
	}
	if c.pipeline != nil {
		dst = bsoncore.AppendArrayElement(dst, "pipeline", c.pipeline)
	}
	if c.size != nil {
		dst = bsoncore.AppendInt64Element(dst, "size", *c.size)
	}
	if c.storageEngine != nil {
		dst = bsoncore.AppendDocumentElement(dst, "storageEngine", c.storageEngine)
	}
	if c.validationAction != nil {
		dst = bsoncore.AppendStringElement(dst, "validationAction", *c.validationAction)
	}
	if c.validationLevel != nil {
		dst = bsoncore.AppendStringElement(dst, "validationLevel", *c.validationLevel)
	}
	if c.validator != nil {
		dst = bsoncore.AppendDocumentElement(dst, "validator", c.validator)
	}
	if c.viewOn != nil {
		dst = bsoncore.AppendStringElement(dst, "viewOn", *c.viewOn)
	}
	if c.expireAfterSeconds != nil {
		dst = bsoncore.AppendInt64Element(dst, "expireAfterSeconds", *c.expireAfterSeconds)
	}
	if c.timeSeries != nil {
		dst = bsoncore.AppendDocumentElement(dst, "timeseries", c.timeSeries)
	}
	if c.encryptedFields != nil {
		dst = bsoncore.AppendDocumentElement(dst, "encryptedFields", c.encryptedFields)
	}
	if c.clusteredIndex != nil {
		dst = bsoncore.AppendDocumentElement(dst, "clusteredIndex", c.clusteredIndex)
	}
	return dst, nil
}
