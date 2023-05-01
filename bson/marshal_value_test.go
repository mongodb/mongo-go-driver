// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/internal/assert"
)

func TestMarshalValue(t *testing.T) {
	t.Parallel()

	marshalValueTestCases := newMarshalValueTestCasesWithInterfaceCore(t)

	t.Run("MarshalValue", func(t *testing.T) {
		t.Parallel()

		for _, tc := range marshalValueTestCases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				valueType, valueBytes, err := MarshalValue(tc.val)
				assert.Nil(t, err, "MarshalValue error: %v", err)
				compareMarshalValueResults(t, tc, valueType, valueBytes)
			})
		}
	})
	t.Run("MarshalValueAppend", func(t *testing.T) {
		t.Parallel()

		for _, tc := range marshalValueTestCases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				valueType, valueBytes, err := MarshalValueAppend(nil, tc.val)
				assert.Nil(t, err, "MarshalValueAppend error: %v", err)
				compareMarshalValueResults(t, tc, valueType, valueBytes)
			})
		}
	})
	t.Run("MarshalValueWithRegistry", func(t *testing.T) {
		t.Parallel()

		for _, tc := range marshalValueTestCases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				valueType, valueBytes, err := MarshalValueWithRegistry(DefaultRegistry, tc.val)
				assert.Nil(t, err, "MarshalValueWithRegistry error: %v", err)
				compareMarshalValueResults(t, tc, valueType, valueBytes)
			})
		}
	})
	t.Run("MarshalValueWithContext", func(t *testing.T) {
		t.Parallel()

		ec := bsoncodec.EncodeContext{Registry: DefaultRegistry}
		for _, tc := range marshalValueTestCases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				valueType, valueBytes, err := MarshalValueWithContext(ec, tc.val)
				assert.Nil(t, err, "MarshalValueWithContext error: %v", err)
				compareMarshalValueResults(t, tc, valueType, valueBytes)
			})
		}
	})
	t.Run("MarshalValueAppendWithRegistry", func(t *testing.T) {
		t.Parallel()

		for _, tc := range marshalValueTestCases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				valueType, valueBytes, err := MarshalValueAppendWithRegistry(DefaultRegistry, nil, tc.val)
				assert.Nil(t, err, "MarshalValueAppendWithRegistry error: %v", err)
				compareMarshalValueResults(t, tc, valueType, valueBytes)
			})
		}
	})
	t.Run("MarshalValueAppendWithContext", func(t *testing.T) {
		t.Parallel()

		ec := bsoncodec.EncodeContext{Registry: DefaultRegistry}
		for _, tc := range marshalValueTestCases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				valueType, valueBytes, err := MarshalValueAppendWithContext(ec, nil, tc.val)
				assert.Nil(t, err, "MarshalValueWithContext error: %v", err)
				compareMarshalValueResults(t, tc, valueType, valueBytes)
			})
		}
	})
}

func compareMarshalValueResults(t *testing.T, tc marshalValueTestCase, gotType bsontype.Type, gotBytes []byte) {
	t.Helper()
	expectedValue := RawValue{Type: tc.bsontype, Value: tc.bytes}
	gotValue := RawValue{Type: gotType, Value: gotBytes}
	assert.Equal(t, expectedValue, gotValue, "value mismatch; expected %s, got %s", expectedValue, gotValue)
}

// benchmark covering GODRIVER-2779
func BenchmarkSliceCodecMarshal(b *testing.B) {
	testStruct := unmarshalerNonPtrStruct{B: []byte(strings.Repeat("t", 4096))}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, err := MarshalValue(testStruct)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
