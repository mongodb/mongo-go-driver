package mongo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driverlegacy/topology"
)

type testBatchCursor struct {
	batches []*bsoncore.DocumentSequence
	batch   *bsoncore.DocumentSequence
}

func newTestBatchCursor(numBatches, batchSize int) *testBatchCursor {
	batches := make([]*bsoncore.DocumentSequence, 0, numBatches)

	counter := 0
	for batch := 0; batch < numBatches; batch++ {
		var docSequence []byte

		for doc := 0; doc < batchSize; doc++ {
			var elem []byte
			elem = bsoncore.AppendInt32Element(elem, "foo", int32(counter))
			counter++

			var doc []byte
			doc = bsoncore.BuildDocumentFromElements(doc, elem)
			docSequence = append(docSequence, doc...)
		}

		batches = append(batches, &bsoncore.DocumentSequence{
			Style: bsoncore.SequenceStyle,
			Data:  docSequence,
		})
	}

	return &testBatchCursor{
		batches: batches,
	}
}

func (tbc *testBatchCursor) ID() int64 {
	if len(tbc.batches) == 0 {
		return 0 // cursor exhausted
	}

	return 10
}

func (tbc *testBatchCursor) Next(context.Context) bool {
	if len(tbc.batches) == 0 {
		return false
	}

	tbc.batch = tbc.batches[0]
	tbc.batches = tbc.batches[1:]
	return true
}

func (tbc *testBatchCursor) Batch() *bsoncore.DocumentSequence {
	return tbc.batch
}

func (tbc *testBatchCursor) Server() *topology.Server {
	return nil
}

func (tbc *testBatchCursor) Err() error {
	return nil
}

func (tbc *testBatchCursor) Close(context.Context) error {
	return nil
}

func TestCursor(t *testing.T) {
	t.Run("loops until docs available", func(t *testing.T) {})
	t.Run("returns false on context cancellation", func(t *testing.T) {})
	t.Run("returns false if error occurred", func(t *testing.T) {})
	t.Run("returns false if ID is zero and no more docs", func(t *testing.T) {})

	t.Run("TestAll", func(t *testing.T) {
		t.Run("errors if argument is not pointer to slice", func(t *testing.T) {
			cursor, err := newCursor(newTestBatchCursor(1, 5), nil)
			require.Nil(t, err)
			err = cursor.All(context.Background(), []bson.D{})
			require.NotNil(t, err)
		})

		t.Run("fills slice with all documents", func(t *testing.T) {
			cursor, err := newCursor(newTestBatchCursor(1, 5), nil)
			require.Nil(t, err)

			var docs []bson.D
			err = cursor.All(context.Background(), &docs)
			require.Nil(t, err)
			require.Equal(t, 5, len(docs))

			for index, doc := range docs {
				require.Equal(t, doc, bson.D{{"foo", int32(index)}})
			}
		})

		t.Run("decodes each document into slice type", func(t *testing.T) {
			cursor, err := newCursor(newTestBatchCursor(1, 5), nil)
			require.Nil(t, err)

			type Document struct {
				Foo int32 `bson:"foo"`
			}
			var docs []Document
			err = cursor.All(context.Background(), &docs)
			require.Nil(t, err)
			require.Equal(t, 5, len(docs))

			for index, doc := range docs {
				require.Equal(t, doc, Document{Foo: int32(index)})
			}
		})

		t.Run("multiple batches are included", func(t *testing.T) {
			cursor, err := newCursor(newTestBatchCursor(2, 5), nil)
			var docs []bson.D
			err = cursor.All(context.Background(), &docs)
			require.Nil(t, err)
			require.Equal(t, 10, len(docs))

			for index, doc := range docs {
				require.Equal(t, doc, bson.D{{"foo", int32(index)}})
			}
		})
	})
}
