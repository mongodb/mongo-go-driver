// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package assertbson

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

func TestEqualDocument(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		expected []byte
		actual   []byte
		want     bool
	}{
		{
			name:     "equal bson.Raw",
			expected: bson.Raw{5, 0, 0, 0, 0},
			actual:   bson.Raw{5, 0, 0, 0, 0},
			want:     true,
		},
		{
			name:     "different bson.Raw",
			expected: bson.Raw{8, 0, 0, 0, 10, 120, 0, 0},
			actual:   bson.Raw{5, 0, 0, 0, 0},
			want:     false,
		},
		{
			name:     "invalid bson.Raw",
			expected: bson.Raw{99, 99, 99, 99},
			actual:   bson.Raw{5, 0, 0, 0, 0},
			want:     false,
		},
		{
			name:     "nil bson.Raw",
			expected: bson.Raw(nil),
			actual:   bson.Raw(nil),
			want:     true,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := EqualDocument(new(testing.T), tc.expected, tc.actual)
			if got != tc.want {
				t.Errorf("EqualDocument(%#v, %#v) = %v, want %v", tc.expected, tc.actual, got, tc.want)
			}
		})
	}
}

func TestEqualValue(t *testing.T) {
	t.Parallel()

	t.Run("bson.RawValue", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name     string
			expected bson.RawValue
			actual   bson.RawValue
			want     bool
		}{
			{
				name: "equal",
				expected: bson.RawValue{
					Type:  bson.TypeInt32,
					Value: []byte{1, 0, 0, 0},
				},
				actual: bson.RawValue{
					Type:  bson.TypeInt32,
					Value: []byte{1, 0, 0, 0},
				},
				want: true,
			},
			{
				name: "same type, different value",
				expected: bson.RawValue{
					Type:  bson.TypeInt32,
					Value: []byte{1, 0, 0, 0},
				},
				actual: bson.RawValue{
					Type:  bson.TypeInt32,
					Value: []byte{1, 1, 1, 1},
				},
				want: false,
			},
			{
				name: "same value, different type",
				expected: bson.RawValue{
					Type:  bson.TypeDouble,
					Value: []byte{1, 0, 0, 0, 0, 0, 0, 0},
				},
				actual: bson.RawValue{
					Type:  bson.TypeInt64,
					Value: []byte{1, 0, 0, 0, 0, 0, 0, 0},
				},
				want: false,
			},
			{
				name: "different value, different type",
				expected: bson.RawValue{
					Type:  bson.TypeInt32,
					Value: []byte{1, 0, 0, 0},
				},
				actual: bson.RawValue{
					Type:  bson.TypeString,
					Value: []byte{1, 1, 1, 1},
				},
				want: false,
			},
			{
				name: "invalid",
				expected: bson.RawValue{
					Type:  bson.TypeInt64,
					Value: []byte{1, 0, 0, 0},
				},
				actual: bson.RawValue{
					Type:  bson.TypeInt32,
					Value: []byte{1, 0, 0, 0},
				},
				want: false,
			},
			{
				name:     "empty",
				expected: bson.RawValue{},
				actual:   bson.RawValue{},
				want:     true,
			},
		}

		for _, tc := range testCases {
			tc := tc // Capture range variable.

			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				got := EqualValue(new(testing.T), tc.expected, tc.actual)
				if got != tc.want {
					t.Errorf("EqualValue(%#v, %#v) = %v, want %v", tc.expected, tc.actual, got, tc.want)
				}
			})
		}
	})

	t.Run("bsoncore.Value", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name     string
			expected bsoncore.Value
			actual   bsoncore.Value
			want     bool
		}{
			{
				name: "equal",
				expected: bsoncore.Value{
					Type: bsoncore.TypeInt32,
					Data: []byte{1, 0, 0, 0},
				},
				actual: bsoncore.Value{
					Type: bsoncore.TypeInt32,
					Data: []byte{1, 0, 0, 0},
				},
				want: true,
			},
			{
				name: "same type, different value",
				expected: bsoncore.Value{
					Type: bsoncore.TypeInt32,
					Data: []byte{1, 0, 0, 0},
				},
				actual: bsoncore.Value{
					Type: bsoncore.TypeInt32,
					Data: []byte{1, 1, 1, 1},
				},
				want: false,
			},
			{
				name: "same value, different type",
				expected: bsoncore.Value{
					Type: bsoncore.TypeDouble,
					Data: []byte{1, 0, 0, 0, 0, 0, 0, 0},
				},
				actual: bsoncore.Value{
					Type: bsoncore.TypeInt64,
					Data: []byte{1, 0, 0, 0, 0, 0, 0, 0},
				},
				want: false,
			},
			{
				name: "different value, different type",
				expected: bsoncore.Value{
					Type: bsoncore.TypeInt32,
					Data: []byte{1, 0, 0, 0},
				},
				actual: bsoncore.Value{
					Type: bsoncore.TypeString,
					Data: []byte{1, 1, 1, 1},
				},
				want: false,
			},
			{
				name: "invalid",
				expected: bsoncore.Value{
					Type: bsoncore.TypeInt64,
					Data: []byte{1, 0, 0, 0},
				},
				actual: bsoncore.Value{
					Type: bsoncore.TypeInt32,
					Data: []byte{1, 0, 0, 0},
				},
				want: false,
			},
			{
				name:     "empty",
				expected: bsoncore.Value{},
				actual:   bsoncore.Value{},
				want:     true,
			},
		}

		for _, tc := range testCases {
			tc := tc // Capture range variable.

			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				got := EqualValue(new(testing.T), tc.expected, tc.actual)
				if got != tc.want {
					t.Errorf("EqualValue(%#v, %#v) = %v, want %v", tc.expected, tc.actual, got, tc.want)
				}
			})
		}
	})
}
