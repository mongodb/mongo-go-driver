// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
)

type testBatchCursor struct {
	batches []*bsoncore.DocumentSequence
	batch   *bsoncore.DocumentSequence
	closed  bool
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

func (tbc *testBatchCursor) Server() driver.Server {
	return nil
}

func (tbc *testBatchCursor) Err() error {
	return nil
}

func (tbc *testBatchCursor) Close(context.Context) error {
	tbc.closed = true
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
			assert.Nil(t, err, "newCursor error: %v", err)
			err = cursor.All(context.Background(), []bson.D{})
			assert.NotNil(t, err, "expected error, got nil")
		})

		t.Run("fills slice with all documents", func(t *testing.T) {
			cursor, err := newCursor(newTestBatchCursor(1, 5), nil)
			assert.Nil(t, err, "newCursor error: %v", err)

			var docs []bson.D
			err = cursor.All(context.Background(), &docs)
			assert.Nil(t, err, "All error: %v", err)
			assert.Equal(t, 5, len(docs), "expected 5 docs, got %v", len(docs))

			for index, doc := range docs {
				expected := bson.D{{"foo", int32(index)}}
				assert.Equal(t, expected, doc, "expected doc %v, got %v", expected, doc)
			}
		})

		t.Run("decodes each document into slice type", func(t *testing.T) {
			cursor, err := newCursor(newTestBatchCursor(1, 5), nil)
			assert.Nil(t, err, "newCursor error: %v", err)

			type Document struct {
				Foo int32 `bson:"foo"`
			}
			var docs []Document
			err = cursor.All(context.Background(), &docs)
			assert.Nil(t, err, "All error: %v", err)
			assert.Equal(t, 5, len(docs), "expected 5 documents, got %v", len(docs))

			for index, doc := range docs {
				expected := Document{Foo: int32(index)}
				assert.Equal(t, expected, doc, "expected doc %v, got %v", expected, doc)
			}
		})

		t.Run("multiple batches are included", func(t *testing.T) {
			cursor, err := newCursor(newTestBatchCursor(2, 5), nil)
			assert.Nil(t, err, "newCursor error: %v", err)
			var docs []bson.D
			err = cursor.All(context.Background(), &docs)
			assert.Nil(t, err, "All error: %v", err)
			assert.Equal(t, 10, len(docs), "expected 10 docs, got %v", len(docs))

			for index, doc := range docs {
				expected := bson.D{{"foo", int32(index)}}
				assert.Equal(t, expected, doc, "expected doc %v, got %v", expected, doc)
			}
		})

		t.Run("cursor is closed after All is called", func(t *testing.T) {
			var docs []bson.D

			tbc := newTestBatchCursor(1, 5)
			cursor, err := newCursor(tbc, nil)
			assert.Nil(t, err, "newCursor error: %v", err)

			err = cursor.All(context.Background(), &docs)
			assert.Nil(t, err, "All error: %v", err)
			assert.True(t, tbc.closed, "expected batch cursor to be closed but was not")
		})

		t.Run("does not error given interface as parameter", func(t *testing.T) {
			var docs interface{} = []bson.D{}

			cursor, err := newCursor(newTestBatchCursor(1, 5), nil)
			assert.Nil(t, err, "newCursor error: %v", err)

			err = cursor.All(context.Background(), &docs)
			assert.Nil(t, err, "expected Nil, got error: %v", err)
			assert.Equal(t, 5, len(docs.([]bson.D)), "expected 5 documents, got %v", len(docs.([]bson.D)))
		})
		t.Run("errors when not given pointer to slice", func(t *testing.T) {
			var docs interface{} = "test"

			cursor, err := newCursor(newTestBatchCursor(1, 5), nil)
			assert.Nil(t, err, "newCursor error: %v", err)

			err = cursor.All(context.Background(), &docs)
			assert.NotNil(t, err, "expected error, got: %v", err)
		})
	})
}

func TestNewCursorFromDocuments(t *testing.T) {
	// Mock documents returned by Find in a Cursor.
	t.Run("mock Find", func(t *testing.T) {
		findResult := []interface{}{
			bson.D{{"_id", 0}, {"foo", "bar"}},
			bson.D{{"_id", 1}, {"baz", "qux"}},
			bson.D{{"_id", 2}, {"quux", "quuz"}},
		}
		cur, err := NewCursorFromDocuments(findResult, nil, nil)
		assert.Nil(t, err, "NewCursorFromDocuments error: %v", err)

		// Assert that decoded documents are as expected.
		var i int
		for cur.Next(context.Background()) {
			docBytes, err := bson.Marshal(findResult[i])
			assert.Nil(t, err, "Marshal error: %v", err)
			expectedDecoded := bson.Raw(docBytes)

			var decoded bson.Raw
			err = cur.Decode(&decoded)
			assert.Nil(t, err, "Decode error: %v", err)
			assert.Equal(t, expectedDecoded, decoded,
				"expected decoded document %v of Cursor to be %v, got %v",
				i, expectedDecoded, decoded)
			i++
		}
		assert.Equal(t, 3, i, "expected 3 calls to cur.Next, got %v", i)

		// Check for error on Cursor.
		assert.Nil(t, cur.Err(), "Cursor error: %v", cur.Err())

		// Assert that a call to cur.Close will not fail.
		err = cur.Close(context.Background())
		assert.Nil(t, err, "Close error: %v", err)
	})

	// Mock an error in a Cursor.
	t.Run("mock Find with error", func(t *testing.T) {
		mockErr := fmt.Errorf("mock error")
		findResult := []interface{}{bson.D{{"_id", 0}, {"foo", "bar"}}}
		cur, err := NewCursorFromDocuments(findResult, mockErr, nil)
		assert.Nil(t, err, "NewCursorFromDocuments error: %v", err)

		// Assert that a call to Next will return false because of existing error.
		next := cur.Next(context.Background())
		assert.False(t, next, "expected call to Next to return false, got true")

		// Check for error on Cursor.
		assert.NotNil(t, cur.Err(), "expected Cursor error, got nil")
		assert.Equal(t, mockErr, cur.Err(), "expected Cursor error %v, got %v",
			mockErr, cur.Err())
	})
}
