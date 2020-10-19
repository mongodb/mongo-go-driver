// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import "context"

func executeIterateUntilDocumentOrError(ctx context.Context, operation *Operation) (*OperationResult, error) {
	stream, err := Entities(ctx).ChangeStream(operation.Object)
	if err != nil {
		return nil, err
	}

	for {
		if stream.TryNext(ctx) {
			return NewDocumentResult(stream.Current, nil), nil
		}
		if stream.Err() != nil {
			return NewErrorResult(stream.Err()), nil
		}
	}
}
