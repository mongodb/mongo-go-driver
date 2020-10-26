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
	"go.mongodb.org/mongo-driver/mongo/options"
)

func executeAbortTransaction(ctx context.Context, operation *Operation) (*OperationResult, error) {
	sess, err := Entities(ctx).Session(operation.Object)
	if err != nil {
		return nil, err
	}

	// AbortTransaction takes no options, so the arguments doc must be nil or empty.
	elems, _ := operation.Arguments.Elements()
	if len(elems) > 0 {
		return nil, fmt.Errorf("unrecognized abortTransaction options %v", operation.Arguments)
	}

	return NewErrorResult(sess.AbortTransaction(ctx)), nil
}

func executeEndSession(ctx context.Context, operation *Operation) error {
	sess, err := Entities(ctx).Session(operation.Object)
	if err != nil {
		return err
	}

	// EnsSession takes no options, so the arguments doc must be nil or empty.
	elems, _ := operation.Arguments.Elements()
	if len(elems) > 0 {
		return fmt.Errorf("unrecognized endSession options %v", operation.Arguments)
	}

	sess.EndSession(ctx)
	return nil
}

func executeCommitTransaction(ctx context.Context, operation *Operation) (*OperationResult, error) {
	sess, err := Entities(ctx).Session(operation.Object)
	if err != nil {
		return nil, err
	}

	// CommitTransaction takes no options, so the arguments doc must be nil or empty.
	elems, _ := operation.Arguments.Elements()
	if len(elems) > 0 {
		return nil, fmt.Errorf("unrecognized commitTransaction options %v", operation.Arguments)
	}

	return NewErrorResult(sess.CommitTransaction(ctx)), nil
}

func executeStartTransaction(ctx context.Context, operation *Operation) (*OperationResult, error) {
	sess, err := Entities(ctx).Session(operation.Object)
	if err != nil {
		return nil, err
	}

	opts := options.Transaction()
	if operation.Arguments != nil {
		var temp TransactionOptions
		if err := bson.Unmarshal(operation.Arguments, &temp); err != nil {
			return nil, fmt.Errorf("error unmarshalling arguments to TransactionOptions: %v", err)
		}

		opts = temp.TransactionOptions
	}

	return NewErrorResult(sess.StartTransaction(opts)), nil
}

func executeWithTransaction(ctx context.Context, operation *Operation) error {
	sess, err := Entities(ctx).Session(operation.Object)
	if err != nil {
		return err
	}

	// Process the "callback" argument. This is an array of Operation objects, each of which should be executed inside
	// the transaction.
	callback, err := operation.Arguments.LookupErr("callback")
	if err != nil {
		return newMissingArgumentError("callback")
	}
	var operations []*Operation
	if err := callback.Unmarshal(&operations); err != nil {
		return fmt.Errorf("error transforming callback option to slice of operations: %v", err)
	}

	// Remove the "callback" field and process the other options.
	var temp TransactionOptions
	if err := bson.Unmarshal(RemoveFieldsFromDocument(operation.Arguments, "callback"), &temp); err != nil {
		return fmt.Errorf("error unmarshalling arguments to TransactionOptions: %v", err)
	}

	_, err = sess.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		for idx, op := range operations {
			if err := op.Execute(ctx); err != nil {
				return nil, fmt.Errorf("error executing operation %q at index %d: %v", op.Name, idx, err)
			}
		}
		return nil, nil
	}, temp.TransactionOptions)
	return err
}
