// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// This file contains helpers to execute client operations.

func executeCreateChangeStream(ctx context.Context, operation *operation) (*operationResult, error) {
	var watcher interface {
		Watch(context.Context, interface{}, ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error)
	}
	var err error

	watcher, err = entities(ctx).client(operation.Object)
	if err != nil {
		watcher, err = entities(ctx).database(operation.Object)
	}
	if err != nil {
		watcher, err = entities(ctx).collection(operation.Object)
	}
	if err != nil {
		return nil, fmt.Errorf("no client, database, or collection entity found with ID %q", operation.Object)
	}

	var pipeline []interface{}
	opts := options.ChangeStream()

	elems, _ := operation.Arguments.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "batchSize":
			opts.SetBatchSize(val.Int32())
		case "collation":
			collation, err := createCollation(val.Document())
			if err != nil {
				return nil, fmt.Errorf("error creating collation: %v", err)
			}
			opts.SetCollation(*collation)
		case "comment":
			commentString, err := createCommentString(val)
			if err != nil {
				return nil, fmt.Errorf("error creating comment: %v", err)
			}
			opts.SetComment(commentString)
		case "fullDocument":
			switch fd := val.StringValue(); fd {
			case "default":
				opts.SetFullDocument(options.Default)
			case "required":
				opts.SetFullDocument(options.Required)
			case "updateLookup":
				opts.SetFullDocument(options.UpdateLookup)
			case "whenAvailable":
				opts.SetFullDocument(options.WhenAvailable)
			default:
				return nil, fmt.Errorf("unrecognized fullDocument value %q", fd)
			}
		case "fullDocumentBeforeChange":
			switch fdbc := val.StringValue(); fdbc {
			case "off":
				opts.SetFullDocumentBeforeChange(options.Off)
			case "required":
				opts.SetFullDocumentBeforeChange(options.Required)
			case "whenAvailable":
				opts.SetFullDocumentBeforeChange(options.WhenAvailable)
			}
		case "maxAwaitTimeMS":
			opts.SetMaxAwaitTime(time.Duration(val.Int32()) * time.Millisecond)
		case "pipeline":
			pipeline = helpers.RawToInterfaces(helpers.RawToDocuments(val.Array())...)
		case "resumeAfter":
			opts.SetResumeAfter(val.Document())
		case "showExpandedEvents":
			opts.SetShowExpandedEvents(val.Boolean())
		case "startAfter":
			opts.SetStartAfter(val.Document())
		case "startAtOperationTime":
			t, i := val.Timestamp()
			opts.SetStartAtOperationTime(&primitive.Timestamp{T: t, I: i})
		default:
			return nil, fmt.Errorf("unrecognized createChangeStream option %q", key)
		}
	}
	if pipeline == nil {
		return nil, newMissingArgumentError("pipeline")
	}

	stream, err := watcher.Watch(ctx, pipeline, opts)
	if err != nil {
		return newErrorResult(err), nil
	}

	// createChangeStream is sometimes used with no corresponding saveResultAsEntity field. Return an
	// empty result in this case.
	if operation.ResultEntityID != nil {
		if err := entities(ctx).addCursorEntity(*operation.ResultEntityID, stream); err != nil {
			return nil, fmt.Errorf("error storing result as cursor entity: %v", err)
		}
	}
	return newEmptyResult(), nil
}

func executeListDatabases(ctx context.Context, operation *operation, nameOnly bool) (*operationResult, error) {
	client, err := entities(ctx).client(operation.Object)
	if err != nil {
		return nil, err
	}

	// We set a default filter rather than erroring if the Arguments doc doesn't have a "filter" field because the
	// spec says drivers should support this field, not must.
	filter := emptyDocument
	opts := options.ListDatabases().SetNameOnly(nameOnly)

	elems, _ := operation.Arguments.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "authorizedDatabases":
			opts.SetAuthorizedDatabases(val.Boolean())
		case "filter":
			filter = val.Document()
		case "nameOnly":
			opts.SetNameOnly(val.Boolean())
		default:
			return nil, fmt.Errorf("unrecognized listDatabases option %q", key)
		}
	}

	res, err := client.ListDatabases(ctx, filter, opts)
	if err != nil {
		return newErrorResult(err), nil
	}

	specsArray := bsoncore.NewArrayBuilder()
	for _, spec := range res.Databases {
		rawSpec := bsoncore.NewDocumentBuilder().
			AppendString("name", spec.Name).
			AppendInt64("sizeOnDisk", spec.SizeOnDisk).
			AppendBoolean("empty", spec.Empty).
			Build()

		specsArray.AppendDocument(rawSpec)
	}
	raw := bsoncore.NewDocumentBuilder().
		AppendArray("databases", specsArray.Build()).
		AppendInt64("totalSize", res.TotalSize).
		Build()
	return newDocumentResult(raw, nil), nil
}
