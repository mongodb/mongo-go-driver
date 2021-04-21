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
)

func executeClose(ctx context.Context, operation *operation) error {
	cursor, err := entities(ctx).cursor(operation.Object)
	if err != nil {
		return err
	}

	// Per the spec, we ignore all errors from Close.
	_ = cursor.Close(ctx)
	return nil
}

func executeIterateUntilDocumentOrError(ctx context.Context, operation *operation) (*operationResult, error) {
	cursor, err := entities(ctx).cursor(operation.Object)
	if err != nil {
		return nil, err
	}

	// Next will loop until there is either a result or an error.
	if cursor.Next(ctx) {
		// We don't expect the server to return malformed documents, so any errors from Decode are treated as fatal.
		var res bson.Raw
		if err := cursor.Decode(&res); err != nil {
			return nil, fmt.Errorf("error decoding cursor result: %v", err)
		}

		return newDocumentResult(res, nil), nil
	}
	return newErrorResult(cursor.Err()), nil
}
