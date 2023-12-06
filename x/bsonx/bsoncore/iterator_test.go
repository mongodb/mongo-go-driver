// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"io"
	"testing"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
)

func TestIterator_Reset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values []Value
	}{
		{
			name: "documents",
			values: []Value{
				{
					Type: bsontype.EmbeddedDocument,
					Data: BuildDocument(nil, AppendDoubleElement(nil, "pi", 3.14159)),
				},
				{
					Type: bsontype.EmbeddedDocument,
					Data: BuildDocument(nil, AppendDoubleElement(nil, "grav", 9.8)),
				},
			},
		},
		{
			name: "strings",
			values: []Value{
				{
					Type: bsontype.String,
					Data: AppendString(nil, "foo"),
				},
				{
					Type: bsontype.String,
					Data: AppendString(nil, "bar"),
				},
			},
		},
		{
			name: "mixed types",
			values: []Value{
				{
					Type: bsontype.String,
					Data: AppendString(nil, "foo"),
				},
				{
					Type: bsontype.Boolean,
					Data: AppendBoolean(nil, true),
				},
				{
					Type: bsontype.EmbeddedDocument,
					Data: BuildDocument(nil, AppendDoubleElement(nil, "pi", 3.14159)),
				},
			},
		},
	}

	for _, tcase := range tests {
		tcase := tcase

		t.Run(tcase.name, func(t *testing.T) {
			t.Parallel()

			array := BuildArray(nil, tcase.values...)
			iter := &Iterator{List: array}

			// 2. Read one of the documents using Next()
			_, err := iter.Next()
			assert.NoError(t, err)

			// 3. Reset the position
			iter.Reset()

			// 4. Assert that we get the first value when re-running Next.
			got, err := iter.Next()

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tcase.values[0], *got)
		})
	}
}

func TestIterator_Count(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values []Value
		want   int
	}{
		{
			name:   "empty",
			values: []Value{},
			want:   0,
		},
		{
			name:   "nil",
			values: nil,
			want:   0,
		},
		{
			name: "singleton",
			values: []Value{
				{
					Type: bsontype.String,
					Data: AppendString(nil, "foo"),
				},
			},
			want: 1,
		},
		{
			name: "non singleton",
			values: []Value{
				{
					Type: bsontype.String,
					Data: AppendString(nil, "foo"),
				},
				{
					Type: bsontype.String,
					Data: AppendString(nil, "bar"),
				},
			},
			want: 2,
		},
		{
			name: "document bearing",
			values: []Value{
				{
					Type: bsontype.EmbeddedDocument,
					Data: BuildDocument(nil, AppendDoubleElement(nil, "pi", 3.14159)),
				},
			},
			want: 1,
		},
		{
			name: "type mixing",
			values: []Value{
				{
					Type: bsontype.String,
					Data: AppendString(nil, "foo"),
				},
				{
					Type: bsontype.Boolean,
					Data: AppendBoolean(nil, true),
				},
				{
					Type: bsontype.EmbeddedDocument,
					Data: BuildDocument(nil, AppendDoubleElement(nil, "pi", 3.14159)),
				},
			},
			want: 3,
		},
	}

	for _, tcase := range tests {
		tcase := tcase

		t.Run(tcase.name, func(t *testing.T) {
			t.Parallel()

			var array Array
			if tcase.values != nil {
				array = BuildArray(nil, tcase.values...)
			}

			got := (&Iterator{List: array}).Count()
			assert.Equal(t, tcase.want, got)
		})
	}
}

func TestIterator_Next(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values []Value
		err    error
	}{
		{
			name:   "empty",
			values: []Value{},
			err:    io.EOF,
		},
		{
			name:   "nil",
			values: nil,
			err:    io.EOF,
		},
		{
			name: "singleton",
			values: []Value{
				{
					Type: bsontype.String,
					Data: AppendString(nil, "foo"),
				},
			},
		},
		{
			name: "document bearing",
			values: []Value{
				{
					Type: bsontype.EmbeddedDocument,
					Data: BuildDocument(nil, AppendDoubleElement(nil, "pi", 3.14159)),
				},
			},
		},
		{
			name: "type mixing",
			values: []Value{
				{
					Type: bsontype.String,
					Data: AppendString(nil, "foo"),
				},
				{
					Type: bsontype.Boolean,
					Data: AppendBoolean(nil, true),
				},
				{
					Type: bsontype.EmbeddedDocument,
					Data: BuildDocument(nil, AppendDoubleElement(nil, "pi", 3.14159)),
				},
			},
		},
	}

	for _, tcase := range tests {
		tcase := tcase

		t.Run(tcase.name, func(t *testing.T) {
			t.Parallel()

			var array Array
			if tcase.values != nil {
				array = BuildArray(nil, tcase.values...)
			}

			iter := &Iterator{List: array}

			for _, want := range tcase.values {
				got, err := iter.Next()
				require.NoErrorf(t, err, "failed to parse the next value")

				assert.Equal(t, want.Type, got.Type)
				assert.Equal(t, want.Data, got.Data)
			}

			// Make sure the last call to next results in an EOF.
			_, err := iter.Next()
			assert.ErrorIs(t, err, io.EOF)
		})
	}

}
