package mongo

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
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
}
