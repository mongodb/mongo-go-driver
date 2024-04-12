// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestSingleResult(t *testing.T) {
	t.Run("Decode", func(t *testing.T) {
		t.Run("decode twice", func(t *testing.T) {
			// Test that Decode and Raw can be called more than once
			c, err := newCursor(newTestBatchCursor(1, 1), nil, bson.DefaultRegistry)
			assert.Nil(t, err, "newCursor error: %v", err)

			sr := &SingleResult[bson.Raw]{cur: c, reg: bson.DefaultRegistry}
			var firstDecode, secondDecode bson.Raw
			err = sr.Decode(&firstDecode)
			assert.Nil(t, err, "Decode error: %v", err)
			err = sr.Decode(&secondDecode)
			assert.Nil(t, err, "Decode error: %v", err)

			rawBytes, err := sr.Raw()
			assert.Nil(t, err, "Raw error: %v", err)

			assert.Equal(t, firstDecode, secondDecode, "expected contents %v, got %v", firstDecode, secondDecode)
			assert.Equal(t, firstDecode, rawBytes, "expected contents %v, got %v", firstDecode, rawBytes)
		})
		t.Run("decode with error", func(t *testing.T) {
			r := []byte("foo")
			sr := &SingleResult[bson.Raw]{
				rdr: bson.RawValue{
					Value: r,
					Type:  bson.TypeEmbeddedDocument,
				},
				err: errors.New("Raw error"),
			}
			res, err := sr.Raw()
			resBytes := []byte(res)
			assert.Equal(t, r, resBytes, "expected contents %v, got %v", r, resBytes)
			assert.Equal(t, sr.err, err, "expected error %v, got %v", sr.err, err)
		})
		t.Run("with BSONOptions", func(t *testing.T) {
			c, err := newCursor(newTestBatchCursor(1, 1), nil, bson.DefaultRegistry)
			require.NoError(t, err, "newCursor error")

			sr := &SingleResult[bson.Raw]{
				cur: c,
				bsonOpts: &options.BSONOptions{
					UseJSONStructTags: true,
				},
				reg: bson.DefaultRegistry,
			}

			type myDocument struct {
				A *int32 `json:"foo"`
			}

			var got myDocument
			err = sr.Decode(&got)
			require.NoError(t, err, "Decode error")

			i := int32(0)
			want := myDocument{A: &i}

			assert.Equal(t, want, got, "expected and actual Decode results are different")
		})
	})

	t.Run("Err", func(t *testing.T) {
		sr := &SingleResult[bson.Raw]{}
		assert.Equal(t, ErrNoDocuments, sr.Err(), "expected error %v, got %v", ErrNoDocuments, sr.Err())
	})
}

func TestNewSingleResultFromDocument(t *testing.T) {
	// Mock a document returned by FindOne in SingleResult[bson.Raw].
	t.Run("mock FindOne", func(t *testing.T) {
		findOneResult := bson.D{{"_id", 2}, {"foo", "bar"}}
		res := NewSingleResultFromDocument[bson.Raw](findOneResult, nil, nil)

		// Assert that first, decoded document is as expected.
		findOneResultBytes, err := bson.Marshal(findOneResult)
		assert.Nil(t, err, "Marshal error: %v", err)
		expectedRawBytes := bson.Raw(findOneResultBytes)
		rawBytes, err := res.Raw()
		assert.Nil(t, err, "Raw error: %v", err)
		assert.Equal(t, expectedRawBytes, rawBytes,
			"expected decoded SingleResult to be %v, got %v", expectedRawBytes, rawBytes)

		// Assert that RDR contents are set correctly after Decode.
		assert.NotNil(t, res.rdr, "expected non-nil rdr contents")
		assert.Equal(t, findOneResultBytes, res.rdr.Value,
			"expected RDR contents to be %v, got %v", expectedRawBytes, res.rdr)

		// Assert that a call to cur.Next will return false, as there was only one document in
		// the slice passed to NewSingleResultFromDocument[bson.Raw].
		next := res.cur.Next(context.Background())
		assert.False(t, next, "expected call to Next to return false, got true")

		// Check for error on SingleResult.
		assert.Nil(t, res.Err(), "SingleResult error: %v", res.Err())

		// Assert that a call to cur.Close will not fail.
		err = res.cur.Close(context.Background())
		assert.Nil(t, err, "Close error: %v", err)
	})

	// Mock an error in SingleResult[bson.Raw].
	t.Run("mock FindOne with error", func(t *testing.T) {
		mockErr := fmt.Errorf("mock error")
		res := NewSingleResultFromDocument[bson.Raw](bson.D{}, mockErr, nil)

		// Assert that the raw bytes returns the mocked error.
		_, err := res.Raw()
		assert.NotNil(t, err, "expected Raw error, got nil")
		assert.Equal(t, mockErr, err, "expected error %v, got %v", mockErr, err)

		// Check for error on SingleResult[bson.Raw].
		assert.NotNil(t, res.Err(), "expected SingleResult[bson.Raw] error, got nil")
		assert.Equal(t, mockErr, res.Err(), "expected SingleResult[bson.Raw] error %v, got %v",
			mockErr, res.Err())

		// Assert that error is propagated to underlying cursor.
		assert.NotNil(t, res.cur.err, "expected underlying cursor, got nil")
		assert.Equal(t, mockErr, res.cur.err, "expected underlying cursor %v, got %v",
			mockErr, res.cur.err)
	})
}
