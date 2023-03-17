// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
)

func TestBatchCursor(t *testing.T) {
	t.Parallel()

	t.Run("setBatchSize", func(t *testing.T) {
		t.Parallel()

		var size int32
		bc := &BatchCursor{
			batchSize: size,
		}
		assert.Equal(t, size, bc.batchSize, "expected batchSize %v, got %v", size, bc.batchSize)

		size = int32(4)
		bc.SetBatchSize(size)
		assert.Equal(t, size, bc.batchSize, "expected batchSize %v, got %v", size, bc.batchSize)
	})

	t.Run("getMoreBatchSize", func(t *testing.T) {
		t.Parallel()

		for _, tcase := range []struct {
			name                               string
			size, limit, numReturned, expected int32
			expectedError                      error
		}{
			{
				name:     "empty",
				expected: 0,
			},
			{
				name:     "batchSize NEQ 0",
				size:     4,
				expected: 4,
			},
			{
				name:     "limit NEQ 0",
				limit:    4,
				expected: 0,
			},
			{
				name:        "limit NEQ and batchSize + numReturned EQ limit",
				size:        4,
				limit:       8,
				numReturned: 4,
				expected:    4,
			},
		} {
			tcase := tcase
			t.Run(tcase.name, func(t *testing.T) {
				t.Parallel()
				ctx := context.Background()

				bc := &BatchCursor{
					limit:     tcase.limit,
					batchSize: tcase.size,
				}

				bc.SetBatchSize(tcase.size)

				size, _, err := bc.getMoreBatchSize(ctx)
				if !errors.Is(err, tcase.expectedError) {
					t.Errorf("expected error %v, got %v", tcase.expectedError, err)
				}

				assert.Equal(t, tcase.expected, size, "expected batchSize %v, got %v",
					tcase.expected, size)
			})
		}
	})
}
