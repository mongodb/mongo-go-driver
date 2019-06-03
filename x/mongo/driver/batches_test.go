package driver

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func TestBatches(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testCases := []struct {
			name    string
			batches *Batches
			want    bool
		}{
			{"nil", nil, false},
			{"missing identifier", &Batches{}, false},
			{"no documents", &Batches{Identifier: "documents"}, false},
			{"valid", &Batches{Identifier: "documents", Documents: make([]bsoncore.Document, 5)}, true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				want := tc.want
				got := tc.batches.Valid()
				if got != want {
					t.Errorf("Did not get expected result from Valid. got %t; want %t", got, want)
				}
			})
		}
	})
	t.Run("ClearBatch", func(t *testing.T) {
		batches := &Batches{Identifier: "documents", Current: make([]bsoncore.Document, 2, 10)}
		if len(batches.Current) != 2 {
			t.Fatalf("Length of current batch should be 2, but is %d", len(batches.Current))
		}
		batches.ClearBatch()
		if len(batches.Current) != 0 {
			t.Fatalf("Length of current batch should be 0, but is %d", len(batches.Current))
		}
	})
	t.Run("AdvanceBatch", func(t *testing.T) {
		testCases := []struct {
			name            string
			batches         *Batches
			maxCount        int
			targetBatchSize int
			err             error
			want            *Batches
		}{
			{
				"current batch non-zero",
				&Batches{Current: make([]bsoncore.Document, 2, 10)},
				0, 0, nil,
				&Batches{Current: make([]bsoncore.Document, 2, 10)},
			},
		}

		for _, tc := range testCases {
			err := tc.batches.AdvanceBatch(tc.maxCount, tc.targetBatchSize)
			if !cmp.Equal(err, tc.err, cmp.Comparer(compareErrors)) {
				t.Errorf("Errors do not match. got %v; want %v", err, tc.err)
			}
			if !cmp.Equal(tc.batches, tc.want) {
				t.Errorf("Batches is not in correct state after AdvanceBatch. got %v; want %v", tc.batches, tc.want)
			}
		}
	})

	t.Run("killCursor", func(t *testing.T) {

		testcases := []struct {
			description    string
			bc             *BatchCursor
			want           error
			expectedCursor bsoncore.DocumentSequence
			ctx            context.Context
		}{
			{"Empty Batch Cursor: TODO", NewEmptyBatchCursor(), nil, bsoncore.DocumentSequence{}, context.TODO()},
			{"Empty Batch Cursor: background", NewEmptyBatchCursor(), nil, bsoncore.DocumentSequence{}, context.Background()},
		}

		for _, test := range testcases {
			if err := test.bc.killCursor(test.ctx); err != test.want {
				t.Errorf("test: %s: killCursor was not correctly killed. Got: %v Expected: %v", test.description, err, test.want)
			}

			test.bc.getMore(test.ctx)
			if !cmp.Equal(*test.bc.Batch(), test.expectedCursor) {
				t.Errorf("test: %s: killCursor returned but didn't kill cursor: got: %v, expected: %v", test.description, test.bc.Batch(), test.expectedCursor)
			}

		}

	})

}
