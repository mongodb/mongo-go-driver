package mongo

import (
	"context"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
	"testing"
)

type testBatchCursor struct {
	batchRemaining bool
	batch          *bsoncore.DocumentSequence
}

func newTestBatchCursor(numDocs int) *testBatchCursor {
	var docSequence []byte
	for i := 0; i < numDocs; i++ {
		var elem []byte
		elem = bsoncore.AppendInt32Element(elem, "foo", int32(i))

		var doc []byte
		doc = bsoncore.BuildDocumentFromElements(doc, elem)
		docSequence = append(docSequence, doc...)
	}

	batch := &bsoncore.DocumentSequence{
		Style: bsoncore.SequenceStyle,
		Data:  docSequence,
	}

	return &testBatchCursor{
		batchRemaining: true,
		batch:          batch,
	}
}

func (tbc *testBatchCursor) ID() int64 {
	return 0
}

func (tbc *testBatchCursor) Next(context.Context) bool {
	if tbc.batchRemaining {
		tbc.batchRemaining = false
		return true
	}

	return false
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
			cursor, err := newCursor(newTestBatchCursor(0), nil)
			require.Nil(t, err)
			err = cursor.All(context.Background(), []bson.D{})
			require.NotNil(t, err)
		})

		t.Run("fills slice with all documents", func(t *testing.T) {
			cursor, err := newCursor(newTestBatchCursor(5), nil)
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
			cursor, err := newCursor(newTestBatchCursor(5), nil)
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
	})
}
