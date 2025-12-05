// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
)

type testBatchCursor struct {
	batches []*bsoncore.Iterator
	batch   *bsoncore.Iterator
	closed  bool
}

var _ batchCursor = (*testBatchCursor)(nil)

func newTestBatchCursor(numBatches, batchSize int) *testBatchCursor {
	batches := make([]*bsoncore.Iterator, 0, numBatches)

	counter := 0
	for batch := 0; batch < numBatches; batch++ {
		var values []bsoncore.Value

		for doc := 0; doc < batchSize; doc++ {
			var elem []byte
			elem = bsoncore.AppendInt32Element(elem, "foo", int32(counter))
			counter++

			var doc []byte
			doc = bsoncore.BuildDocumentFromElements(doc, elem)
			val := bsoncore.Value{
				Type: bsoncore.TypeEmbeddedDocument,
				Data: doc,
			}

			values = append(values, val)
		}

		arr := bsoncore.BuildArray(nil, values...)

		batches = append(batches, &bsoncore.Iterator{
			List: arr,
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

func (tbc *testBatchCursor) Batch() *bsoncore.Iterator {
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

func (tbc *testBatchCursor) SetBatchSize(int32)            {}
func (tbc *testBatchCursor) SetComment(any)                {}
func (tbc *testBatchCursor) SetMaxAwaitTime(time.Duration) {}
func (tbc *testBatchCursor) MaxAwaitTime() *time.Duration  { return nil }

func TestCursor(t *testing.T) {
	t.Run("TestAll", func(t *testing.T) {
		t.Run("errors if argument is not pointer to slice", func(t *testing.T) {
			cursor, err := newCursor(newTestBatchCursor(1, 5), nil, nil)
			require.NoError(t, err, "newCursor error: %v", err)
			err = cursor.All(context.Background(), []bson.D{})
			assert.Error(t, err, "expected error, got nil")
		})

		t.Run("fills slice with all documents", func(t *testing.T) {
			cursor, err := newCursor(newTestBatchCursor(1, 5), nil, nil)
			require.NoError(t, err, "newCursor error: %v", err)

			var docs []bson.D
			err = cursor.All(context.Background(), &docs)
			require.NoError(t, err, "All error: %v", err)
			assert.Len(t, docs, 5, "expected 5 docs, got %v", len(docs))

			for index, doc := range docs {
				expected := bson.D{{"foo", int32(index)}}
				assert.Equal(t, expected, doc, "expected doc %v, got %v", expected, doc)
			}
		})

		t.Run("nil slice", func(t *testing.T) {
			cursor, err := newCursor(newTestBatchCursor(0, 0), nil, nil)
			require.NoError(t, err, "newCursor error: %v", err)

			var docs []bson.D
			err = cursor.All(context.Background(), &docs)
			require.NoError(t, err, "All error: %v", err)
			assert.Nil(t, docs, "expected nil docs")
		})

		t.Run("empty slice", func(t *testing.T) {
			cursor, err := newCursor(newTestBatchCursor(0, 0), nil, nil)
			require.NoError(t, err, "newCursor error: %v", err)

			docs := []bson.D{}
			err = cursor.All(context.Background(), &docs)
			require.NoError(t, err, "All error: %v", err)
			assert.NotNil(t, docs, "expected non-nil docs")
			assert.Len(t, docs, 0, "expected 0 docs, got %v", len(docs))
		})

		t.Run("empty slice overwritten", func(t *testing.T) {
			cursor, err := newCursor(newTestBatchCursor(0, 0), nil, nil)
			require.NoError(t, err, "newCursor error: %v", err)

			docs := []bson.D{{{"foo", "bar"}}, {{"hello", "world"}, {"pi", 3.14159}}}
			err = cursor.All(context.Background(), &docs)
			require.NoError(t, err, "All error: %v", err)
			assert.NotNil(t, docs, "expected non-nil docs")
			assert.Len(t, docs, 0, "expected 0 docs, got %v", len(docs))
		})

		t.Run("decodes each document into slice type", func(t *testing.T) {
			cursor, err := newCursor(newTestBatchCursor(1, 5), nil, nil)
			require.NoError(t, err, "newCursor error: %v", err)

			type Document struct {
				Foo int32 `bson:"foo"`
			}
			var docs []Document
			err = cursor.All(context.Background(), &docs)
			require.NoError(t, err, "All error: %v", err)
			assert.Len(t, docs, 5, "expected 5 documents, got %v", len(docs))

			for index, doc := range docs {
				expected := Document{Foo: int32(index)}
				assert.Equal(t, expected, doc, "expected doc %v, got %v", expected, doc)
			}
		})

		t.Run("multiple batches are included", func(t *testing.T) {
			cursor, err := newCursor(newTestBatchCursor(2, 5), nil, nil)
			require.NoError(t, err, "newCursor error: %v", err)
			var docs []bson.D
			err = cursor.All(context.Background(), &docs)
			require.NoError(t, err, "All error: %v", err)
			assert.Len(t, docs, 10, "expected 10 docs, got %v", len(docs))

			for index, doc := range docs {
				expected := bson.D{{"foo", int32(index)}}
				assert.Equal(t, expected, doc, "expected doc %v, got %v", expected, doc)
			}
		})

		t.Run("cursor is closed after All is called", func(t *testing.T) {
			var docs []bson.D

			tbc := newTestBatchCursor(1, 5)
			cursor, err := newCursor(tbc, nil, nil)
			require.NoError(t, err, "newCursor error: %v", err)

			err = cursor.All(context.Background(), &docs)
			require.NoError(t, err, "All error: %v", err)
			assert.True(t, tbc.closed, "expected batch cursor to be closed but was not")
		})

		t.Run("does not error given interface as parameter", func(t *testing.T) {
			var docs any = []bson.D{}

			cursor, err := newCursor(newTestBatchCursor(1, 5), nil, nil)
			require.NoError(t, err, "newCursor error: %v", err)

			err = cursor.All(context.Background(), &docs)
			require.NoError(t, err, "All error: %v", err)
			assert.Len(t, docs.([]bson.D), 5, "expected 5 documents, got %v", len(docs.([]bson.D)))
		})
		t.Run("errors when not given pointer to slice", func(t *testing.T) {
			var docs any = "test"

			cursor, err := newCursor(newTestBatchCursor(1, 5), nil, nil)
			require.NoError(t, err, "newCursor error: %v", err)

			err = cursor.All(context.Background(), &docs)
			assert.Error(t, err, "expected error, got: %v", err)
		})
		t.Run("with BSONOptions", func(t *testing.T) {
			cursor, err := newCursor(
				newTestBatchCursor(1, 5),
				&options.BSONOptions{
					UseJSONStructTags: true,
				},
				nil)
			require.NoError(t, err, "newCursor error: %v", err)

			type myDocument struct {
				A int32 `json:"foo"`
			}
			var got []myDocument

			err = cursor.All(context.Background(), &got)
			require.NoError(t, err, "All error: %v", err)

			want := []myDocument{{A: 0}, {A: 1}, {A: 2}, {A: 3}, {A: 4}}

			assert.Equal(t, want, got, "expected and actual All results are different")
		})
	})
}

func TestNewCursorFromDocuments(t *testing.T) {
	// Mock documents returned by Find in a Cursor.
	t.Run("mock Find", func(t *testing.T) {
		findResult := []any{
			bson.D{{"_id", 0}, {"foo", "bar"}},
			bson.D{{"_id", 1}, {"baz", "qux"}},
			bson.D{{"_id", 2}, {"quux", "quuz"}},
		}
		cur, err := NewCursorFromDocuments(findResult, nil, nil)
		require.NoError(t, err, "NewCursorFromDocuments error: %v", err)

		// Assert that decoded documents are as expected.
		var i int
		for cur.Next(context.Background()) {
			docBytes, err := bson.Marshal(findResult[i])
			require.NoError(t, err, "Marshal error: %v", err)
			expectedDecoded := bson.Raw(docBytes)

			var decoded bson.Raw
			err = cur.Decode(&decoded)
			require.NoError(t, err, "Decode error: %v", err)
			assert.Equal(t, expectedDecoded, decoded,
				"expected decoded document %v of Cursor to be %v, got %v",
				i, expectedDecoded, decoded)
			i++
		}
		assert.Equal(t, 3, i, "expected 3 calls to cur.Next, got %v", i)

		// Check for error on Cursor.
		require.NoError(t, cur.Err(), "Cursor error: %v", cur.Err())

		// Assert that a call to cur.Close will not fail.
		err = cur.Close(context.Background())
		require.NoError(t, err, "Close error: %v", err)
	})

	// Mock an error in a Cursor.
	t.Run("mock Find with error", func(t *testing.T) {
		mockErr := fmt.Errorf("mock error")
		findResult := []any{bson.D{{"_id", 0}, {"foo", "bar"}}}
		cur, err := NewCursorFromDocuments(findResult, mockErr, nil)
		require.NoError(t, err, "NewCursorFromDocuments error: %v", err)

		// Assert that a call to Next will return false because of existing error.
		next := cur.Next(context.Background())
		assert.False(t, next, "expected call to Next to return false, got true")

		// Check for error on Cursor.
		assert.Error(t, cur.Err(), "expected Cursor error, got nil")
		assert.Equal(t, mockErr, cur.Err(), "expected Cursor error %v, got %v",
			mockErr, cur.Err())
	})
}

func TestGetDecoder(t *testing.T) {
	t.Parallel()

	decT := reflect.TypeOf((*bson.Decoder)(nil))
	ctxT := reflect.TypeOf(bson.DecodeContext{})
	for i := 0; i < decT.NumMethod(); i++ {
		m := decT.Method(i)
		// Test methods with no input/output parameter.
		if m.Type.NumIn() != 1 || m.Type.NumOut() != 0 {
			continue
		}
		t.Run(m.Name, func(t *testing.T) {
			var opts options.BSONOptions
			optsV := reflect.ValueOf(&opts).Elem()
			f, ok := optsV.Type().FieldByName(m.Name)
			require.True(t, ok, "expected %s field in %s", m.Name, optsV.Type())

			wantDec := reflect.ValueOf(bson.NewDecoder(nil))
			_ = wantDec.Method(i).Call(nil)
			wantCtx := wantDec.Elem().Field(0)
			require.Equal(t, ctxT, wantCtx.Type())

			optsV.FieldByIndex(f.Index).SetBool(true)
			gotDec := getDecoder(nil, &opts, nil)
			gotCtx := reflect.ValueOf(gotDec).Elem().Field(0)
			require.Equal(t, ctxT, gotCtx.Type())

			assert.True(t, gotCtx.Equal(wantCtx), "expected %v: %v, got: %v", ctxT, wantCtx, gotCtx)
		})
	}
}

func BenchmarkNewCursorFromDocuments(b *testing.B) {
	// Prepare sample data
	documents := []any{
		bson.D{{"_id", 0}, {"foo", "bar"}},
		bson.D{{"_id", 1}, {"baz", "qux"}},
		bson.D{{"_id", 2}, {"quux", "quuz"}},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := NewCursorFromDocuments(documents, nil, nil)
		if err != nil {
			b.Fatalf("Error creating cursor: %v", err)
		}
	}
}
