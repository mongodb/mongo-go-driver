// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// This file contains helpers to execute database operations.

func executeCreateCollection(ctx context.Context, operation *Operation) (*OperationResult, error) {
	db, err := Entities(ctx).Database(operation.Object)
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
			return nil, fmt.Errorf("unrecognized createCollection option %q", key)
		}
	}
	if collName == "" {
		return nil, newMissingArgumentError("collName")
	}

	err = db.CreateCollection(ctx, collName)
	return NewErrorResult(err), nil
}

func executeDropCollection(ctx context.Context, operation *Operation) (*OperationResult, error) {
	db, err := Entities(ctx).Database(operation.Object)
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
	return NewErrorResult(err), nil
}

func executeListCollections(ctx context.Context, operation *Operation) (*OperationResult, error) {
	db, err := Entities(ctx).Database(operation.Object)
	if err != nil {
		return nil, err
	}

	listCollArgs, err := createListCollectionsArguments(operation.Arguments)
	if err != nil {
		return nil, err
	}

	cursor, err := db.ListCollections(ctx, listCollArgs.filter, listCollArgs.opts)
	if err != nil {
		return NewErrorResult(err), nil
	}
	defer cursor.Close(ctx)

	var docs []bson.Raw
	if err := cursor.All(ctx, &cursor); err != nil {
		return NewErrorResult(err), nil
	}
	return NewCursorResult(docs), nil
}

func executeListCollectionNames(ctx context.Context, operation *Operation) (*OperationResult, error) {
	db, err := Entities(ctx).Database(operation.Object)
	if err != nil {
		return nil, err
	}

	listCollArgs, err := createListCollectionsArguments(operation.Arguments)
	if err != nil {
		return nil, err
	}

	names, err := db.ListCollectionNames(ctx, listCollArgs.filter, listCollArgs.opts)
	if err != nil {
		return NewErrorResult(err), nil
	}
	_, data, err := bson.MarshalValue(names)
	if err != nil {
		return nil, fmt.Errorf("error converting collection names slice to BSON: %v", err)
	}
	return NewValueResult(bsontype.Array, data, nil), nil
}

func executeRunCommand(ctx context.Context, operation *Operation) (*OperationResult, error) {
	db, err := Entities(ctx).Database(operation.Object)
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
			var temp readPreference
			if err := bson.Unmarshal(val.Document(), &temp); err != nil {
				return nil, fmt.Errorf("error unmarshalling readPreference option: %v", err)
			}

			rp, err := temp.toReadPrefOption()
			if err != nil {
				return nil, fmt.Errorf("error creating readpref.ReadPref object: %v", err)
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

	res, err := db.RunCommand(ctx, command, opts).DecodeBytes()
	return NewDocumentResult(res, err), nil
}
