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
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// listDatabasesOp performs a listDatabases operation.
type listDatabasesOp struct {
	authenticator             driver.Authenticator
	filter                    bsoncore.Document
	authorizedDatabases       *bool
	nameOnly                  *bool
	session                   *session.Client
	clock                     *session.ClusterClock
	monitor                   *event.CommandMonitor
	database                  string
	deployment                driver.Deployment
	readPreference            *readpref.ReadPref
	retry                     *driver.RetryMode
	maxAdaptiveRetries        uint
	enableOverloadRetargeting bool
	selector                  description.ServerSelector
	crypt                     driver.Crypt
	serverAPI                 *driver.ServerAPIOptions
	timeout                   *time.Duration

	res listDatabasesResult
}

// listDatabasesResult represents a listDatabases result returned by the server.
type listDatabasesResult struct {
	// An array of documents, one document for each database
	Databases []databaseRecord
	// The sum of the size of all the database files on disk in bytes.
	TotalSize int64
}

type databaseRecord struct {
	Name       string
	SizeOnDisk int64 `bson:"sizeOnDisk"`
	Empty      bool
}

func buildListDatabasesResult(response bsoncore.Document) (listDatabasesResult, error) {
	elements, err := response.Elements()
	if err != nil {
		return listDatabasesResult{}, err
	}
	ir := listDatabasesResult{}
	for _, element := range elements {
		switch element.Key() {
		case "totalSize":
			var ok bool
			ir.TotalSize, ok = element.Value().AsInt64OK()
			if !ok {
				return ir, fmt.Errorf("response field 'totalSize' is type int64, but received BSON type %s: %s", element.Value().Type, element.Value())
			}
		case "databases":
			arr, ok := element.Value().ArrayOK()
			if !ok {
				return ir, fmt.Errorf("response field 'databases' is type array, but received BSON type %s", element.Value().Type)
			}

			var tmp bsoncore.Document
			err := bson.Unmarshal(arr, &tmp)
			if err != nil {
				return ir, err
			}

			records, err := tmp.Elements()
			if err != nil {
				return ir, err
			}

			ir.Databases = make([]databaseRecord, len(records))
			for i, val := range records {
				valueDoc, ok := val.Value().DocumentOK()
				if !ok {
					return ir, fmt.Errorf("'databases' element is type document, but received BSON type %s", val.Value().Type)
				}

				elems, err := valueDoc.Elements()
				if err != nil {
					return ir, err
				}

				for _, elem := range elems {
					switch elem.Key() {
					case "name":
						ir.Databases[i].Name, ok = elem.Value().StringValueOK()
						if !ok {
							return ir, fmt.Errorf("response field 'name' is type string, but received BSON type %s", elem.Value().Type)
						}
					case "sizeOnDisk":
						ir.Databases[i].SizeOnDisk, ok = elem.Value().AsInt64OK()
						if !ok {
							return ir, fmt.Errorf("response field 'sizeOnDisk' is type int64, but received BSON type %s", elem.Value().Type)
						}
					case "empty":
						ir.Databases[i].Empty, ok = elem.Value().BooleanOK()
						if !ok {
							return ir, fmt.Errorf("response field 'empty' is type bool, but received BSON type %s", elem.Value().Type)
						}
					}
				}
			}
		}
	}
	return ir, nil
}

// result returns the result of executing this operation.
func (ld *listDatabasesOp) result() listDatabasesResult { return ld.res }

func (ld *listDatabasesOp) processResponse(_ context.Context, resp bsoncore.Document, _ driver.ResponseInfo) error {
	var err error

	ld.res, err = buildListDatabasesResult(resp)
	return err
}

// execute runs this operations and returns an error if the operation did not execute successfully.
func (ld *listDatabasesOp) execute(ctx context.Context) error {
	if ld.deployment == nil {
		return errors.New("the ListDatabases operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:         ld.command,
		ProcessResponseFn: ld.processResponse,

		Client:                    ld.session,
		Clock:                     ld.clock,
		CommandMonitor:            ld.monitor,
		Database:                  ld.database,
		Deployment:                ld.deployment,
		ReadPreference:            ld.readPreference,
		RetryMode:                 ld.retry,
		MaxAdaptiveRetries:        ld.maxAdaptiveRetries,
		EnableOverloadRetargeting: ld.enableOverloadRetargeting,
		Type:                      driver.Read,
		Selector:                  ld.selector,
		Crypt:                     ld.crypt,
		ServerAPI:                 ld.serverAPI,
		Timeout:                   ld.timeout,
		Name:                      driverutil.ListDatabasesOp,
		Authenticator:             ld.authenticator,
	}.Execute(ctx)
}

func (ld *listDatabasesOp) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendInt32Element(dst, "listDatabases", 1)
	if ld.filter != nil {
		dst = bsoncore.AppendDocumentElement(dst, "filter", ld.filter)
	}
	if ld.nameOnly != nil {
		dst = bsoncore.AppendBooleanElement(dst, "nameOnly", *ld.nameOnly)
	}
	if ld.authorizedDatabases != nil {
		dst = bsoncore.AppendBooleanElement(dst, "authorizedDatabases", *ld.authorizedDatabases)
	}

	return dst, nil
}
