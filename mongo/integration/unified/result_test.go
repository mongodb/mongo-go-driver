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
)

// OperationResult holds the result and/or error returned by an op.
type OperationResult struct {
	// For operations that return a single result, this field holds a BSON representation.
	Result bson.RawValue

	// CursorResult holds the documents retrieved by iterating a Cursor.
	CursorResult []bson.Raw

	// Err holds the error returned by an operation. This is mutually exclusive with CursorResult but not with Result
	// because some operations (e.g. bulkWrite) can return both a result and an error.
	Err error
}

// NewEmptyResult returns an OperationResult with no fields set. This should be used if the operation does not check
// results or errors.
func NewEmptyResult() *OperationResult {
	return &OperationResult{}
}

// NewDocumentResult is a helper to create a value result where the value is a BSON document.
func NewDocumentResult(result []byte, err error) *OperationResult {
	return NewValueResult(bsontype.EmbeddedDocument, result, err)
}

// NewValueResult creates an OperationResult where the result is a BSON value of an arbitrary type. Because some
// operations can return both a result and an error (e.g. bulkWrite), the err parameter should be the error returned
// by the op, if any.
func NewValueResult(valueType bsontype.Type, data []byte, err error) *OperationResult {
	return &OperationResult{
		Result: bson.RawValue{Type: valueType, Value: data},
		Err:    err,
	}
}

// NewCursorResult creates an OperationResult that contains documents retrieved by fully iterating a cursor.
func NewCursorResult(arr []bson.Raw) *OperationResult {
	// If the operation returned no documents, the array might be nil. It isn't possible to distinguish between this
	// case and the case where there is no cursor result, so we overwrite the result with an non-nil empty slice.
	result := arr
	if result == nil {
		result = make([]bson.Raw, 0)
	}

	return &OperationResult{
		CursorResult: result,
	}
}

// NewErrorResult creates an OperationResult that only holds an error. This should only be used when executing an
// operation that can return a result or an error, but not both.
func NewErrorResult(err error) *OperationResult {
	return &OperationResult{
		Err: err,
	}
}

func VerifyOperationResult(ctx context.Context, expected bson.RawValue, actual *OperationResult) error {
	actualVal := actual.Result
	if actual.CursorResult != nil {
		_, data, err := bson.MarshalValue(actual.CursorResult)
		if err != nil {
			return fmt.Errorf("error converting cursorResult array to BSON: %v", err)
		}

		actualVal = bson.RawValue{
			Type:  bsontype.Array,
			Value: data,
		}
	}

	// For document results and arrays of root documents (i.e. cursor results), the actual value can have additional
	// top-level keys. Single-value array results (e.g. from distinct) must match exactly, so we set extraKeysAllowed to
	// false only for that case.
	extraKeysAllowed := actual.Result.Type != bsontype.Array
	return VerifyValuesMatch(ctx, expected, actualVal, extraKeysAllowed)
}
