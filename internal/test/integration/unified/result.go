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

// operationResult holds the result and/or error returned by an op.
type operationResult struct {
	// For operations that return a single result, this field holds a BSON representation.
	Result bson.RawValue

	// CursorResult holds the documents retrieved by iterating a Cursor.
	CursorResult []bson.Raw

	// Err holds the error returned by an operation. This is mutually exclusive with CursorResult but not with Result
	// because some operations (e.g. bulkWrite) can return both a result and an error.
	Err error
}

// newEmptyResult returns an operationResult with no fields set. This should be used if the operation does not check
// results or errors.
func newEmptyResult() *operationResult {
	return &operationResult{}
}

// newDocumentResult is a helper to create a value result where the value is a BSON document.
func newDocumentResult(result []byte, err error) *operationResult {
	return newValueResult(bsontype.EmbeddedDocument, result, err)
}

// newValueResult creates an operationResult where the result is a BSON value of an arbitrary type. Because some
// operations can return both a result and an error (e.g. bulkWrite), the err parameter should be the error returned
// by the op, if any.
func newValueResult(valueType bsontype.Type, data []byte, err error) *operationResult {
	return &operationResult{
		Result: bson.RawValue{Type: valueType, Value: data},
		Err:    err,
	}
}

// newCursorResult creates an operationResult that contains documents retrieved by fully iterating a cursor.
func newCursorResult(arr []bson.Raw) *operationResult {
	// If the operation returned no documents, the array might be nil. It isn't possible to distinguish between this
	// case and the case where there is no cursor result, so we overwrite the result with an non-nil empty slice.
	result := arr
	if result == nil {
		result = make([]bson.Raw, 0)
	}

	return &operationResult{
		CursorResult: result,
	}
}

// newErrorResult creates an operationResult that only holds an error. This should only be used when executing an
// operation that can return a result or an error, but not both.
func newErrorResult(err error) *operationResult {
	return &operationResult{
		Err: err,
	}
}

// verifyOperationResult checks that the actual and expected results match
func verifyOperationResult(ctx context.Context, expected bson.RawValue, actual *operationResult) error {
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
	return verifyValuesMatch(ctx, expected, actualVal, extraKeysAllowed)
}
