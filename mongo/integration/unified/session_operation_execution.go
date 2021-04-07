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

func executeAbortTransaction(ctx context.Context, operation *operation) (*operationResult, error) {
	sess, err := entities(ctx).session(operation.Object)
	if err != nil {
		return nil, err
	}

	// AbortTransaction takes no options, so the arguments doc must be nil or empty.
	elems, _ := operation.Arguments.Elements()
	if len(elems) > 0 {
		return nil, fmt.Errorf("unrecognized abortTransaction options %v", operation.Arguments)
	}

	return newErrorResult(sess.AbortTransaction(ctx)), nil
}

func executeEndSession(ctx context.Context, operation *operation) error {
	sess, err := entities(ctx).session(operation.Object)
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

func executeCommitTransaction(ctx context.Context, operation *operation) (*operationResult, error) {
	sess, err := entities(ctx).session(operation.Object)
	if err != nil {
		return nil, err
	}

	// CommitTransaction takes no options, so the arguments doc must be nil or empty.
	elems, _ := operation.Arguments.Elements()
	if len(elems) > 0 {
		return nil, fmt.Errorf("unrecognized commitTransaction options %v", operation.Arguments)
	}

	return newErrorResult(sess.CommitTransaction(ctx)), nil
}

func executeStartTransaction(ctx context.Context, operation *operation) (*operationResult, error) {
	sess, err := entities(ctx).session(operation.Object)
	if err != nil {
		return nil, err
	}

	opts := options.Transaction()
	if operation.Arguments != nil {
		var temp transactionOptions
		if err := bson.Unmarshal(operation.Arguments, &temp); err != nil {
			return nil, fmt.Errorf("error unmarshalling arguments to transactionOptions: %v", err)
		}

		opts = temp.TransactionOptions
	}

	return newErrorResult(sess.StartTransaction(opts)), nil
}

func executeWithTransaction(ctx context.Context, op *operation, loopDone <-chan struct{}) error {
	sess, err := entities(ctx).session(op.Object)
	if err != nil {
		return err
	}

	// Process the "callback" argument. This is an array of operation objects, each of which should be executed inside
	// the transaction.
	callback, err := op.Arguments.LookupErr("callback")
	if err != nil {
		return newMissingArgumentError("callback")
	}
	var operations []*operation
	if err := callback.Unmarshal(&operations); err != nil {
		return fmt.Errorf("error transforming callback option to slice of operations: %v", err)
	}

	// Remove the "callback" field and process the other options.
	var temp transactionOptions
	if err := bson.Unmarshal(removeFieldsFromDocument(op.Arguments, "callback"), &temp); err != nil {
		return fmt.Errorf("error unmarshalling arguments to transactionOptions: %v", err)
	}

	_, err = sess.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		for idx, oper := range operations {
			if err := oper.execute(ctx, loopDone); err != nil {
				return nil, fmt.Errorf("error executing operation %q at index %d: %v", oper.Name, idx, err)
			}
		}
		return nil, nil
	}, temp.TransactionOptions)
	return err
}
