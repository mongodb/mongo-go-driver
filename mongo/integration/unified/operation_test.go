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

type Operation struct {
	Name           string         `bson:"name"`
	Object         string         `bson:"object"`
	Arguments      bson.Raw       `bson:"arguments"`
	ExpectedError  *ExpectedError `bson:"expectError"`
	ExpectedResult *bson.RawValue `bson:"expectResult"`
	ResultEntityID *string        `bson:"saveResultAsEntity"`
}

// Execute runs the operation and verifies the returned result and/or error. If the result needs to be saved as
// an entity, it also updates the EntityMap associated with ctx to do so.
func (op *Operation) Execute(ctx context.Context) error {
	res, err := op.run(ctx)
	if err != nil {
		return fmt.Errorf("execution failed: %v", err)
	}

	if err := VerifyOperationError(ctx, op.ExpectedError, res); err != nil {
		return fmt.Errorf("error verification failed: %v", err)
	}

	if op.ExpectedResult != nil {
		if err := VerifyOperationResult(ctx, *op.ExpectedResult, res); err != nil {
			return fmt.Errorf("result verification failed: %v", err)
		}
	}
	return nil
}

func (op *Operation) run(ctx context.Context) (*OperationResult, error) {
	if op.Object == "testRunner" {
		// testRunner operations don't have results or expected errors, so we use NewEmptyResult to fake a result.
		return NewEmptyResult(), executeTestRunnerOperation(ctx, op)
	}

	// Special handling for the "session" field because it applies to all operations.
	if id, ok := op.Arguments.Lookup("session").StringValueOK(); ok {
		sess, err := Entities(ctx).Session(id)
		if err != nil {
			return nil, err
		}
		ctx = mongo.NewSessionContext(ctx, sess)

		// Set op.Arguments to a new document that has the "session" field removed so individual operations do
		// not have to account for it.
		op.Arguments = RemoveFieldsFromDocument(op.Arguments, "session")
	}

	switch op.Name {
	// Session operations
	case "abortTransaction":
		return executeAbortTransaction(ctx, op)
	case "commitTransaction":
		return executeCommitTransaction(ctx, op)
	case "endSession":
		// The EndSession() method doesn't return a result, so we return a non-nil empty result.
		return NewEmptyResult(), executeEndSession(ctx, op)
	case "startTransaction":
		return executeStartTransaction(ctx, op)
	case "withTransaction":
		// executeWithTransaction internally verifies results/errors for each operation, so it doesn't return a result.
		return NewEmptyResult(), executeWithTransaction(ctx, op)

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
	case "findOneAndUpdate":
		return executeFindOneAndUpdate(ctx, op)
	case "insertMany":
		return executeInsertMany(ctx, op)
	case "insertOne":
		return executeInsertOne(ctx, op)
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

	// Change Stream operations
	case "iterateUntilDocumentOrError":
		return executeIterateUntilDocumentOrError(ctx, op)
	default:
		return nil, fmt.Errorf("unrecognized entity operation %q", op.Name)
	}
}
