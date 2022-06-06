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
	"go.mongodb.org/mongo-driver/mongo"
)

type operation struct {
	Name                 string         `bson:"name"`
	Object               string         `bson:"object"`
	Arguments            bson.Raw       `bson:"arguments"`
	IgnoreResultAndError bool           `bson:"ignoreResultAndError"`
	ExpectedError        *expectedError `bson:"expectError"`
	ExpectedResult       *bson.RawValue `bson:"expectResult"`
	ResultEntityID       *string        `bson:"saveResultAsEntity"`
}

// execute runs the operation and verifies the returned result and/or error. If the result needs to be saved as
// an entity, it also updates the entityMap associated with ctx to do so.
func (op *operation) execute(ctx context.Context, loopDone <-chan struct{}) error {
	res, err := op.run(ctx, loopDone)
	if err != nil {
		return fmt.Errorf("execution failed: %v", err)
	}

	if op.IgnoreResultAndError {
		return nil
	}

	if err := verifyOperationError(ctx, op.ExpectedError, res); err != nil {
		return fmt.Errorf("error verification failed: %v", err)
	}

	if op.ExpectedResult != nil {
		if err := verifyOperationResult(ctx, *op.ExpectedResult, res); err != nil {
			return fmt.Errorf("result verification failed: %v", err)
		}
	}
	return nil
}

// isCreateView will return true if the operation is to create a collection with a view.
func (op *operation) isCreateView() (bool, error) {
	if op.Name != "createCollection" {
		return false, nil
	}

	elements, err := op.Arguments.Elements()
	if err != nil {
		return false, err
	}

	var has bool
	for _, elem := range elements {
		if elem.Key() == "viewOn" {
			has = true
			break
		}
	}
	return has, nil
}

func (op *operation) run(ctx context.Context, loopDone <-chan struct{}) (*operationResult, error) {
	if op.Object == "testRunner" {
		// testRunner operations don't have results or expected errors, so we use newEmptyResult to fake a result.
		return newEmptyResult(), executeTestRunnerOperation(ctx, op, loopDone)
	}

	// Special handling for the "session" field because it applies to all operations.
	if id, ok := op.Arguments.Lookup("session").StringValueOK(); ok {
		sess, err := entities(ctx).session(id)
		if err != nil {
			return nil, err
		}
		ctx = mongo.NewSessionContext(ctx, sess)

		// Set op.Arguments to a new document that has the "session" field removed so individual operations do
		// not have to account for it.
		op.Arguments = removeFieldsFromDocument(op.Arguments, "session")
	}

	switch op.Name {
	// Session operations
	case "abortTransaction":
		return executeAbortTransaction(ctx, op)
	case "commitTransaction":
		return executeCommitTransaction(ctx, op)
	case "endSession":
		// The EndSession() method doesn't return a result, so we return a non-nil empty result.
		return newEmptyResult(), executeEndSession(ctx, op)
	case "startTransaction":
		return executeStartTransaction(ctx, op)
	case "withTransaction":
		// executeWithTransaction internally verifies results/errors for each operation, so it doesn't return a result.
		return newEmptyResult(), executeWithTransaction(ctx, op, loopDone)

	// Client operations
	case "createChangeStream":
		return executeCreateChangeStream(ctx, op)
	case "listDatabases":
		return executeListDatabases(ctx, op)

	// Database operations
	case "createCollection":
		return executeCreateCollection(ctx, op)
	case "dropCollection":
		return executeDropCollection(ctx, op)
	case "listCollections":
		return executeListCollections(ctx, op)
	case "listCollectionNames":
		return executeListCollectionNames(ctx, op)
	case "runCommand":
		return executeRunCommand(ctx, op)

	// Collection operations
	case "aggregate":
		return executeAggregate(ctx, op)
	case "bulkWrite":
		return executeBulkWrite(ctx, op)
	case "countDocuments":
		return executeCountDocuments(ctx, op)
	case "createIndex":
		return executeCreateIndex(ctx, op)
	case "createFindCursor":
		return executeCreateFindCursor(ctx, op)
	case "deleteOne":
		return executeDeleteOne(ctx, op)
	case "deleteMany":
		return executeDeleteMany(ctx, op)
	case "distinct":
		return executeDistinct(ctx, op)
	case "estimatedDocumentCount":
		return executeEstimatedDocumentCount(ctx, op)
	case "find":
		return executeFind(ctx, op)
	case "findOneAndDelete":
		return executeFindOneAndDelete(ctx, op)
	case "findOneAndReplace":
		return executeFindOneAndReplace(ctx, op)
	case "findOneAndUpdate":
		return executeFindOneAndUpdate(ctx, op)
	case "insertMany":
		return executeInsertMany(ctx, op)
	case "insertOne":
		return executeInsertOne(ctx, op)
	case "listIndexes":
		return executeListIndexes(ctx, op)
	case "rename":
		return executeRenameCollection(ctx, op)
	case "replaceOne":
		return executeReplaceOne(ctx, op)
	case "updateOne":
		return executeUpdateOne(ctx, op)
	case "updateMany":
		return executeUpdateMany(ctx, op)

	// GridFS operations
	case "delete":
		return executeBucketDelete(ctx, op)
	case "download":
		return executeBucketDownload(ctx, op)
	case "upload":
		return executeBucketUpload(ctx, op)

	// Cursor operations
	case "close":
		return newEmptyResult(), executeClose(ctx, op)
	case "iterateOnce":
		return executeIterateOnce(ctx, op)
	case "iterateUntilDocumentOrError":
		return executeIterateUntilDocumentOrError(ctx, op)

	// Unsupported operations
	case "count", "listIndexNames", "modifyCollection":
		return nil, newSkipTestError(fmt.Sprintf("the %q operation is not supported", op.Name))
	case "createKey":
		return executeCreateKey(ctx, op)
	default:
		return nil, fmt.Errorf("unrecognized entity operation %q", op.Name)
	}
}
