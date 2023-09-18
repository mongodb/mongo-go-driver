// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"encoding/hex"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func TestMatches(t *testing.T) {
	ctx := context.Background()
	unmarshalExtJSONValue := func(t *testing.T, str string) bson.RawValue {
		t.Helper()

		if str == "" {
			return bson.RawValue{}
		}

		var val bson.RawValue
		err := bson.UnmarshalExtJSON([]byte(str), false, &val)
		assert.Nil(t, err, "UnmarshalExtJSON error: %v", err)
		return val
	}
	marshalValue := func(t *testing.T, val interface{}) bson.RawValue {
		t.Helper()

		valType, data, err := bson.MarshalValue(val)
		assert.Nil(t, err, "MarshalValue error: %v", err)
		return bson.RawValue{
			Type:  valType,
			Value: data,
		}
	}

	assertMatches := func(t *testing.T, expected, actual bson.RawValue, shouldMatch bool) {
		t.Helper()

		err := verifyValuesMatch(ctx, expected, actual, true)
		if shouldMatch {
			assert.Nil(t, err, "expected values to match, but got comparison error %v", err)
			return
		}
		assert.NotNil(t, err, "expected values to not match, but got no error")
	}

	t.Run("documents with extra keys allowed", func(t *testing.T) {
		expectedDoc := `{"x": 1, "y": {"a": 1, "b": 2}}`
		expectedVal := unmarshalExtJSONValue(t, expectedDoc)

		extraKeysAtRoot := `{"x": 1, "y": {"a": 1, "b": 2}, "z": 3}`
		extraKeysInEmbedded := `{"x": 1, "y": {"a": 1, "b": 2, "c": 3}}`

		testCases := []struct {
			name    string
			actual  string
			matches bool
		}{
			{"exact match", expectedDoc, true},
			{"extra keys allowed at root-level", extraKeysAtRoot, true},
			{"incorrect type for y", `{"x": 1, "y": 2}`, false},
			{"extra keys prohibited in embedded documents", extraKeysInEmbedded, false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assertMatches(t, expectedVal, unmarshalExtJSONValue(t, tc.actual), tc.matches)
			})
		}
	})
	t.Run("documents with extra keys not allowed", func(t *testing.T) {
		expected := unmarshalExtJSONValue(t, `{"x": 1}`)
		actual := unmarshalExtJSONValue(t, `{"x": 1, "y": 1}`)
		err := verifyValuesMatch(ctx, expected, actual, false)
		assert.NotNil(t, err, "expected values to not match, but got no error")
	})
	t.Run("exists operator", func(t *testing.T) {
		rootExists := unmarshalExtJSONValue(t, `{"x": {"$$exists": true}}`)
		rootNotExists := unmarshalExtJSONValue(t, `{"x": {"$$exists": false}}`)
		embeddedExists := unmarshalExtJSONValue(t, `{"x": {"y": {"$$exists": true}}}`)
		embeddedNotExists := unmarshalExtJSONValue(t, `{"x": {"y": {"$$exists": false}}}`)

		testCases := []struct {
			name     string
			expected bson.RawValue
			actual   string
			matches  bool
		}{
			{"root - should exist and does", rootExists, `{"x": 1}`, true},
			{"root - should exist and does not", rootExists, `{"y": 1}`, false},
			{"root - should not exist and does", rootNotExists, `{"x": 1}`, false},
			{"root - should not exist and does not", rootNotExists, `{"y": 1}`, true},
			{"embedded - should exist and does", embeddedExists, `{"x": {"y": 1}}`, true},
			{"embedded - should exist and does not", embeddedExists, `{"x": {}}`, false},
			{"embedded - should not exist and does", embeddedNotExists, `{"x": {"y": 1}}`, false},
			{"embedded - should not exist and does not", embeddedNotExists, `{"x": {}}`, true},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assertMatches(t, tc.expected, unmarshalExtJSONValue(t, tc.actual), tc.matches)
			})
		}
	})
	t.Run("type operator", func(t *testing.T) {
		singleType := unmarshalExtJSONValue(t, `{"x": {"$$type": "string"}}`)
		arrayTypes := unmarshalExtJSONValue(t, `{"x": {"$$type": ["string", "bool"]}}`)

		testCases := []struct {
			name     string
			expected bson.RawValue
			actual   string
			matches  bool
		}{
			{"single type matches", singleType, `{"x": "foo"}`, true},
			{"single type does not match", singleType, `{"x": 1}`, false},
			{"multiple types matches first", arrayTypes, `{"x": "foo"}`, true},
			{"multiple types matches second", arrayTypes, `{"x": true}`, true},
			{"multiple types does not match", arrayTypes, `{"x": 1}`, false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assertMatches(t, tc.expected, unmarshalExtJSONValue(t, tc.actual), tc.matches)
			})
		}
	})
	t.Run("matchesHexBytes operator", func(t *testing.T) {
		singleValue := unmarshalExtJSONValue(t, `{"$$matchesHexBytes": "DEADBEEF"}`)
		document := unmarshalExtJSONValue(t, `{"x": {"$$matchesHexBytes": "DEADBEEF"}}`)

		stringToBinary := func(str string) bson.RawValue {
			hexBytes, err := hex.DecodeString(str)
			assert.Nil(t, err, "hex.DecodeString error: %v", err)
			return bson.RawValue{
				Type:  bsontype.Binary,
				Value: bsoncore.AppendBinary(nil, 0, hexBytes),
			}
		}

		testCases := []struct {
			name     string
			expected bson.RawValue
			actual   bson.RawValue
			matches  bool
		}{
			{"single value matches", singleValue, stringToBinary("DEADBEEF"), true},
			{"single value does not match", singleValue, stringToBinary("BEEF"), false},
			{"document matches", document, marshalValue(t, bson.M{"x": stringToBinary("DEADBEEF")}), true},
			{"document does not match", document, marshalValue(t, bson.M{"x": stringToBinary("BEEF")}), false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assertMatches(t, tc.expected, tc.actual, tc.matches)
			})
		}
	})
	t.Run("unsetOrMatches operator", func(t *testing.T) {
		topLevel := unmarshalExtJSONValue(t, `{"$$unsetOrMatches": {"x": 1}}`)
		nested := unmarshalExtJSONValue(t, `{"x": {"$$unsetOrMatches": {"y": 1}}}`)

		testCases := []struct {
			name     string
			expected bson.RawValue
			actual   string
			matches  bool
		}{
			{"top-level unset", topLevel, "", true},
			{"top-level matches", topLevel, `{"x": 1}`, true},
			{"top-level matches with extra keys", topLevel, `{"x": 1, "y": 1}`, true},
			{"top-level does not match", topLevel, `{"x": 2}`, false},
			{"nested unset", nested, `{}`, true},
			{"nested matches", nested, `{"x": {"y": 1}}`, true},
			{"nested field exists but is null", nested, `{"x": null}`, false}, // null should not be considered unset
			{"nested does not match", nested, `{"x": {"y": 2}}`, false},
			{"nested does not match due to extra keys", nested, `{"x": {"y": 1, "z": 1}}`, false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assertMatches(t, tc.expected, unmarshalExtJSONValue(t, tc.actual), tc.matches)
			})
		}
	})
}
