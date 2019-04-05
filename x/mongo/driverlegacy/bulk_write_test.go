// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driverlegacy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBulkWrite(t *testing.T) {
	t.Run("TestCreateBatches", func(t *testing.T) {
		models := []WriteModel{
			InsertOneModel{},
			InsertOneModel{},
			UpdateOneModel{},
			UpdateManyModel{},
			DeleteOneModel{},
			DeleteManyModel{},
			InsertOneModel{},
			UpdateManyModel{},
			DeleteOneModel{},
		}

		expectedOrdered := []bulkWriteBatch{
			{[]WriteModel{InsertOneModel{}, InsertOneModel{}}, true},
			{[]WriteModel{UpdateOneModel{}, UpdateManyModel{}}, false},
			{[]WriteModel{DeleteOneModel{}, DeleteManyModel{}}, false},
			{[]WriteModel{InsertOneModel{}}, true},
			{[]WriteModel{UpdateManyModel{}}, false},
			{[]WriteModel{DeleteOneModel{}}, true},
		}

		expectedUnordered := []bulkWriteBatch{
			{[]WriteModel{InsertOneModel{}, InsertOneModel{}, InsertOneModel{}}, true},
			{[]WriteModel{UpdateOneModel{}, UpdateManyModel{}, UpdateManyModel{}}, false},
			{[]WriteModel{DeleteOneModel{}, DeleteManyModel{}, DeleteOneModel{}}, false},
		}

		testCases := []struct {
			name     string
			models   []WriteModel
			ordered  bool
			expected []bulkWriteBatch
		}{
			{"Ordered", models, true, expectedOrdered},
			{"Unordered", models, false, expectedUnordered},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				batches := createBatches(tc.models, tc.ordered)
				require.Equal(t, len(batches), len(tc.expected))

				for i, batch := range batches {
					expectedBatch := tc.expected[i]
					require.Equal(t, expectedBatch, batch)
				}
			})
		}
	})
}
