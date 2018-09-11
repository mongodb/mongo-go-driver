package dispatch

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
