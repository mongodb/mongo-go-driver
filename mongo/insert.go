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

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/driverutil"
	"go.mongodb.org/mongo-driver/v2/internal/logger"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// insert performs an insert operation.
type insert struct {
	authenticator            driver.Authenticator
	bypassDocumentValidation *bool
	comment                  bsoncore.Value
	documents                []bsoncore.Document
	ordered                  *bool
	session                  *session.Client
	clock                    *session.ClusterClock
	collection               string
	monitor                  *event.CommandMonitor
	crypt                    driver.Crypt
	database                 string
	deployment               driver.Deployment
	selector                 description.ServerSelector
	writeConcern             *writeconcern.WriteConcern
	retry                    *driver.RetryMode
	result                   insertResult
	serverAPI                *driver.ServerAPIOptions
	timeout                  *time.Duration
	rawData                  *bool
	additionalCmd            bson.D
	logger                   *logger.Logger
}

// insertResult represents an insert result returned by the server.
type insertResult struct {
	// Number of documents successfully inserted.
	N int64
}

func buildInsertResult(response bsoncore.Document) (insertResult, error) {
	elements, err := response.Elements()
	if err != nil {
		return insertResult{}, err
	}
	ir := insertResult{}
	for _, element := range elements {
		if element.Key() == "n" {
			var ok bool
			ir.N, ok = element.Value().AsInt64OK()
			if !ok {
				return ir, fmt.Errorf("response field 'n' is type int32 or int64, but received BSON type %s", element.Value().Type)
			}
		}
	}
	return ir, nil
}

// Result returns the result of executing this operation.
func (i *insert) Result() insertResult { return i.result }

func (i *insert) processResponse(_ context.Context, resp bsoncore.Document, _ driver.ResponseInfo) error {
	ir, err := buildInsertResult(resp)
	i.result.N += ir.N
	return err
}

// Execute runs this operations and returns an error if the operation did not execute successfully.
func (i *insert) Execute(ctx context.Context) error {
	if i.deployment == nil {
		return errors.New("the Insert operation must have a Deployment set before Execute can be called")
	}
	batches := &driver.Batches{
		Identifier: "documents",
		Documents:  i.documents,
		Ordered:    i.ordered,
	}

	return driver.Operation{
		CommandFn:         i.command,
		ProcessResponseFn: i.processResponse,
		Batches:           batches,
		RetryMode:         i.retry,
		Type:              driver.Write,
		Client:            i.session,
		Clock:             i.clock,
		CommandMonitor:    i.monitor,
		Crypt:             i.crypt,
		Database:          i.database,
		Deployment:        i.deployment,
		Selector:          i.selector,
		WriteConcern:      i.writeConcern,
		ServerAPI:         i.serverAPI,
		Timeout:           i.timeout,
		Logger:            i.logger,
		Name:              driverutil.InsertOp,
		Authenticator:     i.authenticator,
	}.Execute(ctx)
}

func (i *insert) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "insert", i.collection)
	if i.bypassDocumentValidation != nil && (desc.WireVersion != nil &&
		driverutil.VersionRangeIncludes(*desc.WireVersion, 4)) {

		dst = bsoncore.AppendBooleanElement(dst, "bypassDocumentValidation", *i.bypassDocumentValidation)
	}
	if i.comment.Type != bsoncore.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "comment", i.comment)
	}
	if i.ordered != nil {
		dst = bsoncore.AppendBooleanElement(dst, "ordered", *i.ordered)
	}
	// Set rawData for 8.2+ servers.
	if i.rawData != nil && desc.WireVersion != nil && driverutil.VersionRangeIncludes(*desc.WireVersion, 27) {
		dst = bsoncore.AppendBooleanElement(dst, "rawData", *i.rawData)
	}
	if len(i.additionalCmd) > 0 {
		doc, err := bson.Marshal(i.additionalCmd)
		if err != nil {
			return nil, err
		}
		dst = append(dst, doc[4:len(doc)-1]...)
	}
	return dst, nil
}
