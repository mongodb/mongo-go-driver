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

var errUnacknowledgedHint = errors.New("the 'hint' command parameter cannot be used with unacknowledged writes")

// updateOp performs an update operation.
type updateOp struct {
	authenticator             driver.Authenticator
	bypassDocumentValidation  *bool
	comment                   bsoncore.Value
	ordered                   *bool
	updates                   []bsoncore.Document
	session                   *session.Client
	clock                     *session.ClusterClock
	collection                string
	monitor                   *event.CommandMonitor
	database                  string
	deployment                driver.Deployment
	hint                      *bool
	arrayFilters              *bool
	selector                  description.ServerSelector
	writeConcern              *writeconcern.WriteConcern
	retry                     *driver.RetryMode
	maxAdaptiveRetries        uint
	enableOverloadRetargeting bool
	result                    updateResult
	crypt                     driver.Crypt
	serverAPI                 *driver.ServerAPIOptions
	let                       bsoncore.Document
	timeout                   *time.Duration
	rawData                   *bool
	additionalCmd             bson.D
	logger                    *logger.Logger
}

// upsert contains the information for an upsert in an update operation.
type upsert struct {
	Index int64
	ID    any `bson:"_id"`
}

// updateResult represents an update result returned by the server.
type updateResult struct {
	N         int64
	NModified int64
	Upserted  []upsert
}

func buildUpdateResult(response bsoncore.Document) (updateResult, error) {
	elements, err := response.Elements()
	if err != nil {
		return updateResult{}, err
	}
	ur := updateResult{}
	for _, element := range elements {
		switch element.Key() {
		case "nModified":
			var ok bool
			ur.NModified, ok = element.Value().AsInt64OK()
			if !ok {
				return ur, fmt.Errorf("response field 'nModified' is type int32 or int64, but received BSON type %s", element.Value().Type)
			}
		case "n":
			var ok bool
			ur.N, ok = element.Value().AsInt64OK()
			if !ok {
				return ur, fmt.Errorf("response field 'n' is type int32 or int64, but received BSON type %s", element.Value().Type)
			}
		case "upserted":
			arr, ok := element.Value().ArrayOK()
			if !ok {
				return ur, fmt.Errorf("response field 'upserted' is type array, but received BSON type %s", element.Value().Type)
			}

			values, err := arr.Values()
			if err != nil {
				return ur, err
			}

			for _, val := range values {
				valDoc, ok := val.DocumentOK()
				if !ok {
					return ur, fmt.Errorf("upserted value is type document, but received BSON type %s", val.Type)
				}
				var u upsert
				if err = bson.Unmarshal(valDoc, &u); err != nil {
					return ur, err
				}
				ur.Upserted = append(ur.Upserted, u)
			}
		}
	}
	return ur, nil
}

// Result returns the result of executing this operation.
func (u *updateOp) Result() updateResult { return u.result }

func (u *updateOp) processResponse(_ context.Context, resp bsoncore.Document, info driver.ResponseInfo) error {
	ur, err := buildUpdateResult(resp)

	u.result.N += ur.N
	u.result.NModified += ur.NModified
	if info.CurrentIndex > 0 {
		for ind := range ur.Upserted {
			ur.Upserted[ind].Index += int64(info.CurrentIndex)
		}
	}
	u.result.Upserted = append(u.result.Upserted, ur.Upserted...)
	return err
}

// Execute runs this operation and returns an error if the operation did not execute successfully.
func (u *updateOp) Execute(ctx context.Context) error {
	if u.deployment == nil {
		return errors.New("the update operation must have a Deployment set before Execute can be called")
	}
	batches := &driver.Batches{
		Identifier: "updates",
		Documents:  u.updates,
		Ordered:    u.ordered,
	}

	return driver.Operation{
		CommandFn:                 u.command,
		ProcessResponseFn:         u.processResponse,
		Batches:                   batches,
		RetryMode:                 u.retry,
		MaxAdaptiveRetries:        u.maxAdaptiveRetries,
		EnableOverloadRetargeting: u.enableOverloadRetargeting,
		Type:                      driver.Write,
		Client:                    u.session,
		Clock:                     u.clock,
		CommandMonitor:            u.monitor,
		Database:                  u.database,
		Deployment:                u.deployment,
		Selector:                  u.selector,
		WriteConcern:              u.writeConcern,
		Crypt:                     u.crypt,
		ServerAPI:                 u.serverAPI,
		Timeout:                   u.timeout,
		Logger:                    u.logger,
		Name:                      driverutil.UpdateOp,
		Authenticator:             u.authenticator,
		SendAfterClusterTime:      true,
	}.Execute(ctx)
}

func (u *updateOp) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "update", u.collection)
	if u.bypassDocumentValidation != nil &&
		(desc.WireVersion != nil && driverutil.VersionRangeIncludes(*desc.WireVersion, 4)) {

		dst = bsoncore.AppendBooleanElement(dst, "bypassDocumentValidation", *u.bypassDocumentValidation)
	}
	if u.comment.Type != bsoncore.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "comment", u.comment)
	}
	if u.ordered != nil {
		dst = bsoncore.AppendBooleanElement(dst, "ordered", *u.ordered)
	}
	if u.hint != nil && *u.hint {
		if desc.WireVersion == nil || !driverutil.VersionRangeIncludes(*desc.WireVersion, 5) {
			return nil, errors.New("the 'hint' command parameter requires a minimum server wire version of 5")
		}
		if !u.writeConcern.Acknowledged() {
			return nil, errUnacknowledgedHint
		}
	}
	if u.arrayFilters != nil && *u.arrayFilters {
		if desc.WireVersion == nil || !driverutil.VersionRangeIncludes(*desc.WireVersion, 6) {
			return nil, errors.New("the 'arrayFilters' command parameter requires a minimum server wire version of 6")
		}
	}
	if u.let != nil {
		dst = bsoncore.AppendDocumentElement(dst, "let", u.let)
	}
	// Set rawData for 8.2+ servers.
	if u.rawData != nil && desc.WireVersion != nil && driverutil.VersionRangeIncludes(*desc.WireVersion, 27) {
		dst = bsoncore.AppendBooleanElement(dst, "rawData", *u.rawData)
	}
	if len(u.additionalCmd) > 0 {
		doc, err := bson.Marshal(u.additionalCmd)
		if err != nil {
			return nil, err
		}
		dst = append(dst, doc[4:len(doc)-1]...)
	}

	return dst, nil
}
