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

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/internal/bsonutil"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// This file contains helpers to execute database operations.

func executeCreateView(ctx context.Context, operation *operation) (*operationResult, error) {
	db, err := entities(ctx).database(operation.Object)
	if err != nil {
		return nil, err
	}

	var collName string
	var cvo options.CreateViewOptions
	var viewOn string
	pipeline := make([]interface{}, 0)

	elems, err := operation.Arguments.Elements()
	if err != nil {
		return nil, err
	}

	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "collection":
			collName = val.StringValue()
		case "pipeline":
			pipeline = bsonutil.RawToInterfaces(bsonutil.RawToDocuments(val.Array())...)
		case "viewOn":
			viewOn = val.StringValue()
		default:
			return nil, fmt.Errorf("unrecognized createView option %q", key)
		}
	}
	if collName == "" {
		return nil, newMissingArgumentError("collection")
	}
	if viewOn == "" {
		return nil, newMissingArgumentError("viewOn")
	}

	err = db.CreateView(ctx, collName, viewOn, pipeline, &cvo)
	return newErrorResult(err), nil
}

func executeCreateCollection(ctx context.Context, operation *operation) (*operationResult, error) {
	// In the Go driver there is a separate method for creating views.  However, the unified test CRUD format does not
	// make this distinction.  If necessary, here we branch to create a view.
	createView, err := operation.isCreateView()
	if err != nil {
		return nil, err
	}
	if createView {
		return executeCreateView(ctx, operation)
	}

	db, err := entities(ctx).database(operation.Object)
	if err != nil {
		return nil, err
	}

	var collName string
	var cco options.CreateCollectionOptions
	elems, _ := operation.Arguments.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "collection":
			collName = val.StringValue()
		case "changeStreamPreAndPostImages":
			cco.SetChangeStreamPreAndPostImages(val.Document())
		case "expireAfterSeconds":
			cco.SetExpireAfterSeconds(int64(val.Int32()))
		case "capped":
			cco.SetCapped(val.Boolean())
		case "size":
			cco.SetSizeInBytes(val.AsInt64())
		case "max":
			cco.SetMaxDocuments(val.AsInt64())
		case "timeseries":
			tsElems, err := elem.Value().Document().Elements()
			if err != nil {
				return nil, err
			}

			tso := options.TimeSeries()
			for _, elem := range tsElems {
				key := elem.Key()
				val := elem.Value()

				switch key {
				case "timeField":
					tso.SetTimeField(val.StringValue())
				case "metaField":
					tso.SetMetaField(val.StringValue())
				case "granularity":
					tso.SetGranularity(val.StringValue())
				case "bucketMaxSpanSeconds":
					tso.SetBucketMaxSpan(time.Duration(val.Int32()) * time.Second)
				case "bucketRoundingSeconds":
					tso.SetBucketRounding(time.Duration(val.Int32()) * time.Second)
				default:
					return nil, fmt.Errorf("unrecognized timeseries option %q", key)
				}
			}
			cco.SetTimeSeriesOptions(tso)
		case "clusteredIndex":
			cco.SetClusteredIndex(val.Document())
		default:
			return nil, fmt.Errorf("unrecognized createCollection option %q", key)
		}
	}
	if collName == "" {
		return nil, newMissingArgumentError("collection")
	}

	err = db.CreateCollection(ctx, collName, &cco)
	if err != nil {
		return newErrorResult(err), nil
	}

	if collID := operation.ResultEntityID; collID != nil {
		collEntityOpts := newCollectionEntityOptions(*collID, operation.Object, collName, nil)

		err := entities(ctx).addCollectionEntity(collEntityOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to save collection as entity: %w", err)
		}
	}

	return newEmptyResult(), nil
}

func executeDropCollection(ctx context.Context, operation *operation) (*operationResult, error) {
	db, err := entities(ctx).database(operation.Object)
	if err != nil {
		return nil, err
	}

	var collName string
	elems, _ := operation.Arguments.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "collection":
			collName = val.StringValue()
		default:
			return nil, fmt.Errorf("unrecognized dropCollection option %q", key)
		}
	}
	if collName == "" {
		return nil, newMissingArgumentError("collection")
	}

	err = db.Collection(collName).Drop(ctx)
	return newErrorResult(err), nil
}

func executeListCollections(ctx context.Context, operation *operation) (*operationResult, error) {
	db, err := entities(ctx).database(operation.Object)
	if err != nil {
		return nil, err
	}

	listCollArgs, err := createListCollectionsArguments(operation.Arguments)
	if err != nil {
		return nil, err
	}

	cursor, err := db.ListCollections(ctx, listCollArgs.filter, listCollArgs.opts)
	if err != nil {
		return newErrorResult(err), nil
	}
	defer cursor.Close(ctx)

	var docs []bson.Raw
	if err := cursor.All(ctx, &docs); err != nil {
		return newErrorResult(err), nil
	}
	return newCursorResult(docs), nil
}

func executeListCollectionNames(ctx context.Context, operation *operation) (*operationResult, error) {
	db, err := entities(ctx).database(operation.Object)
	if err != nil {
		return nil, err
	}

	listCollArgs, err := createListCollectionsArguments(operation.Arguments)
	if err != nil {
		return nil, err
	}

	names, err := db.ListCollectionNames(ctx, listCollArgs.filter, listCollArgs.opts)
	if err != nil {
		return newErrorResult(err), nil
	}
	_, data, err := bson.MarshalValue(names)
	if err != nil {
		return nil, fmt.Errorf("error converting collection names slice to BSON: %w", err)
	}
	return newValueResult(bsontype.Array, data, nil), nil
}

func executeRunCommand(ctx context.Context, operation *operation) (*operationResult, error) {
	db, err := entities(ctx).database(operation.Object)
	if err != nil {
		return nil, err
	}

	var command bson.Raw
	opts := options.RunCmd()

	elems, _ := operation.Arguments.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "command":
			command = val.Document()
		case "commandName":
			// This is only necessary for languages that cannot preserve key order in the command document, so we can
			// ignore it.
		case "readConcern":
			// GODRIVER-1774: We currently don't support overriding read concern for RunCommand.
			return nil, fmt.Errorf("readConcern in runCommand not supported")
		case "readPreference":
			var temp ReadPreference
			if err := bson.Unmarshal(val.Document(), &temp); err != nil {
				return nil, fmt.Errorf("error unmarshalling readPreference option: %w", err)
			}

			rp, err := temp.ToReadPrefOption()
			if err != nil {
				return nil, fmt.Errorf("error creating readpref.ReadPref object: %w", err)
			}
			opts.SetReadPreference(rp)
		case "writeConcern":
			// GODRIVER-1774: We currently don't support overriding write concern for RunCommand.
			return nil, fmt.Errorf("writeConcern in runCommand not supported")
		default:
			return nil, fmt.Errorf("unrecognized runCommand option %q", key)
		}
	}
	if command == nil {
		return nil, newMissingArgumentError("command")
	}

	res, err := db.RunCommand(ctx, command, opts).Raw()
	return newDocumentResult(res, err), nil
}

// executeRunCursorCommand proxies the database's runCursorCommand method and
// supports the same arguments and options.
func executeRunCursorCommand(ctx context.Context, operation *operation) (*operationResult, error) {
	db, err := entities(ctx).database(operation.Object)
	if err != nil {
		return nil, err
	}

	var (
		batchSize int32
		command   bson.Raw
		comment   bson.Raw
		maxTime   time.Duration
	)

	opts := options.RunCmd()

	elems, _ := operation.Arguments.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "batchSize":
			batchSize = val.Int32()
		case "command":
			command = val.Document()
		case "commandName":
			// This is only necessary for languages that cannot
			// preserve key order in the command document, so we can
			// ignore it.
		case "comment":
			comment = val.Document()
		case "maxTimeMS":
			maxTime = time.Duration(val.AsInt64()) * time.Millisecond
		case "cursorTimeout":
			return nil, newSkipTestError("cursorTimeout not supported")
		case "timeoutMode":
			return nil, newSkipTestError("timeoutMode not supported")
		default:
			return nil, fmt.Errorf("unrecognized runCursorCommand option: %q", key)
		}
	}

	if command == nil {
		return nil, newMissingArgumentError("command")
	}

	cursor, err := db.RunCommandCursor(ctx, command, opts)
	if err != nil {
		return newErrorResult(err), nil
	}

	if batchSize > 0 {
		cursor.SetBatchSize(batchSize)
	}

	if maxTime > 0 {
		cursor.SetMaxTime(maxTime)
	}

	if len(comment) > 0 {
		cursor.SetComment(comment)
	}

	// When executing the provided command, the test runner MUST fully
	// iterate the cursor. This will ensure consistent behavior between
	// drivers that eagerly create a server-side cursor and those that do
	// so lazily when iteration begins.
	var docs []bson.Raw
	if err := cursor.All(ctx, &docs); err != nil {
		return newErrorResult(err), nil
	}

	return newCursorResult(docs), nil
}

// executeCreateRunCursorCommand proxies the database's runCursorCommand method
// and supports the same arguments and options.
func executeCreateRunCursorCommand(ctx context.Context, operation *operation) (*operationResult, error) {
	db, err := entities(ctx).database(operation.Object)
	if err != nil {
		return nil, err
	}

	var (
		batchSize int32
		command   bson.Raw
	)

	opts := options.RunCmd()

	elems, _ := operation.Arguments.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "batchSize":
			batchSize = val.Int32()
		case "command":
			command = val.Document()
		case "commandName":
			// This is only necessary for languages that cannot
			// preserve key order in the command document, so we can
			// ignore it.
		case "cursorType":
			return nil, newSkipTestError("cursorType not supported")
		case "timeoutMode":
			return nil, newSkipTestError("timeoutMode not supported")
		default:
			return nil, fmt.Errorf("unrecognized createRunCursorCommand option: %q", key)
		}
	}

	if command == nil {
		return nil, newMissingArgumentError("command")
	}

	// Test runners MUST ensure that the server-side cursor is created (i.e.
	// the command document has executed) as part of this operation.
	cursor, err := db.RunCommandCursor(ctx, command, opts)
	if err != nil {
		return newErrorResult(err), nil
	}

	if batchSize > 0 {
		cursor.SetBatchSize(batchSize)
	}

	if cursorID := operation.ResultEntityID; cursorID != nil {
		err := entities(ctx).addCursorEntity(*cursorID, cursor)
		if err != nil {
			return nil, fmt.Errorf("failed to store result as cursor entity: %w", err)
		}
	}

	return newEmptyResult(), nil
}
