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
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// findAndModifyOp performs a findAndModify operation.
type findAndModifyOp struct {
	authenticator             driver.Authenticator
	arrayFilters              bsoncore.Array
	bypassDocumentValidation  *bool
	collation                 bsoncore.Document
	comment                   bsoncore.Value
	fields                    bsoncore.Document
	newDocument               *bool
	query                     bsoncore.Document
	remove                    *bool
	sort                      bsoncore.Document
	update                    bsoncore.Value
	upsert                    *bool
	session                   *session.Client
	clock                     *session.ClusterClock
	collection                string
	monitor                   *event.CommandMonitor
	database                  string
	deployment                driver.Deployment
	selector                  description.ServerSelector
	writeConcern              *writeconcern.WriteConcern
	retry                     *driver.RetryMode
	maxAdaptiveRetries        uint
	enableOverloadRetargeting bool
	crypt                     driver.Crypt
	hint                      bsoncore.Value
	serverAPI                 *driver.ServerAPIOptions
	let                       bsoncore.Document
	timeout                   *time.Duration
	rawData                   *bool
	additionalCmd             bson.D

	res findAndModifyResult
}

// lastErrorObject represents information about updates and upserts returned by the server.
type lastErrorObject struct {
	// True if an update modified an existing document
	UpdatedExisting bool
	// Object ID of the upserted document.
	Upserted any
}

// findAndModifyResult represents a findAndModify result returned by the server.
type findAndModifyResult struct {
	// Either the old or modified document, depending on the value of the new parameter.
	Value bsoncore.Document
	// Contains information about updates and upserts.
	LastErrorObject lastErrorObject
}

func buildFindAndModifyResult(response bsoncore.Document) (findAndModifyResult, error) {
	elements, err := response.Elements()
	if err != nil {
		return findAndModifyResult{}, err
	}
	famr := findAndModifyResult{}
	for _, element := range elements {
		switch element.Key() {
		case "value":
			var ok bool
			famr.Value, ok = element.Value().DocumentOK()

			// The 'value' field returned by a FindAndModify can be null in the case that no document was found.
			if element.Value().Type != bsoncore.TypeNull && !ok {
				return famr, fmt.Errorf("response field 'value' is type document or null, but received BSON type %s", element.Value().Type)
			}
		case "lastErrorObject":
			valDoc, ok := element.Value().DocumentOK()
			if !ok {
				return famr, fmt.Errorf("response field 'lastErrorObject' is type document, but received BSON type %s", element.Value().Type)
			}

			var leo lastErrorObject
			if err = bson.Unmarshal(valDoc, &leo); err != nil {
				return famr, err
			}
			famr.LastErrorObject = leo
		}
	}
	return famr, nil
}

// result returns the result of executing this operation.
func (fam *findAndModifyOp) result() findAndModifyResult { return fam.res }

func (fam *findAndModifyOp) processResponse(_ context.Context, resp bsoncore.Document, _ driver.ResponseInfo) error {
	var err error

	fam.res, err = buildFindAndModifyResult(resp)
	return err
}

// execute runs this operations and returns an error if the operation did not execute successfully.
func (fam *findAndModifyOp) execute(ctx context.Context) error {
	if fam.deployment == nil {
		return errors.New("the FindAndModify operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:         fam.command,
		ProcessResponseFn: fam.processResponse,

		RetryMode:                 fam.retry,
		MaxAdaptiveRetries:        fam.maxAdaptiveRetries,
		EnableOverloadRetargeting: fam.enableOverloadRetargeting,
		Type:                      driver.Write,
		Client:                    fam.session,
		Clock:                     fam.clock,
		CommandMonitor:            fam.monitor,
		Database:                  fam.database,
		Deployment:                fam.deployment,
		Selector:                  fam.selector,
		WriteConcern:              fam.writeConcern,
		Crypt:                     fam.crypt,
		ServerAPI:                 fam.serverAPI,
		Timeout:                   fam.timeout,
		Name:                      driverutil.FindAndModifyOp,
		Authenticator:             fam.authenticator,
		SendAfterClusterTime:      true,
	}.Execute(ctx)
}

func (fam *findAndModifyOp) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "findAndModify", fam.collection)
	if fam.arrayFilters != nil {

		if desc.WireVersion == nil || !driverutil.VersionRangeIncludes(*desc.WireVersion, 6) {
			return nil, errors.New("the 'arrayFilters' command parameter requires a minimum server wire version of 6")
		}
		dst = bsoncore.AppendArrayElement(dst, "arrayFilters", fam.arrayFilters)
	}
	if fam.bypassDocumentValidation != nil {
		dst = bsoncore.AppendBooleanElement(dst, "bypassDocumentValidation", *fam.bypassDocumentValidation)
	}
	if fam.collation != nil {

		if desc.WireVersion == nil || !driverutil.VersionRangeIncludes(*desc.WireVersion, 5) {
			return nil, errors.New("the 'collation' command parameter requires a minimum server wire version of 5")
		}
		dst = bsoncore.AppendDocumentElement(dst, "collation", fam.collation)
	}
	if fam.comment.Type != bsoncore.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "comment", fam.comment)
	}
	if fam.fields != nil {
		dst = bsoncore.AppendDocumentElement(dst, "fields", fam.fields)
	}
	if fam.newDocument != nil {
		dst = bsoncore.AppendBooleanElement(dst, "new", *fam.newDocument)
	}
	if fam.query != nil {
		dst = bsoncore.AppendDocumentElement(dst, "query", fam.query)
	}
	if fam.remove != nil {
		dst = bsoncore.AppendBooleanElement(dst, "remove", *fam.remove)
	}
	if fam.sort != nil {
		dst = bsoncore.AppendDocumentElement(dst, "sort", fam.sort)
	}
	if fam.update.Data != nil {
		dst = bsoncore.AppendValueElement(dst, "update", fam.update)
	}
	if fam.upsert != nil {
		dst = bsoncore.AppendBooleanElement(dst, "upsert", *fam.upsert)
	}
	if fam.hint.Type != bsoncore.Type(0) {

		if desc.WireVersion == nil || !driverutil.VersionRangeIncludes(*desc.WireVersion, 8) {
			return nil, errors.New("the 'hint' command parameter requires a minimum server wire version of 8")
		}
		if !fam.writeConcern.Acknowledged() {
			return nil, errUnacknowledgedHint
		}
		dst = bsoncore.AppendValueElement(dst, "hint", fam.hint)
	}
	if fam.let != nil {
		dst = bsoncore.AppendDocumentElement(dst, "let", fam.let)
	}
	// Set rawData for 8.2+ servers.
	if fam.rawData != nil && desc.WireVersion != nil && driverutil.VersionRangeIncludes(*desc.WireVersion, 27) {
		dst = bsoncore.AppendBooleanElement(dst, "rawData", *fam.rawData)
	}
	if len(fam.additionalCmd) > 0 {
		doc, err := bson.Marshal(fam.additionalCmd)
		if err != nil {
			return nil, err
		}
		dst = append(dst, doc[4:len(doc)-1]...)
	}

	return dst, nil
}
