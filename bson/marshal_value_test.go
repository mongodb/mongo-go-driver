// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"slices"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

func TestMarshalValue(t *testing.T) {
	t.Parallel()

	testCases := slices.Clone(marshalValueTestCases)
	testCases = append(testCases, marshalValueTestCase{
		name: "interface",
		val: marshalValueInterfaceOuter{
			Reader: marshalValueInterfaceInner{
				Foo: 10,
			},
		},
		bsontype: TypeEmbeddedDocument,
		bytes: bsoncore.NewDocumentBuilder().
			AppendDocument("reader", bsoncore.NewDocumentBuilder().
				AppendInt32("foo", 10).
				Build()).
			Build(),
	})

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			valueType, valueBytes, err := MarshalValue(tc.val)
			assert.Nil(t, err, "MarshalValue error: %v", err)
			compareMarshalValueResults(t, tc, valueType, valueBytes)
		})
	}

	t.Run("returns distinct address ranges", func(t *testing.T) {
		// Call MarshalValue in a loop with the same large value (make sure to
		// trigger the buffer pooling, which currently doesn't happen for very
		// small values). Compare the previous and current BSON byte slices and
		// make sure they always have distinct memory ranges.
		//
		// Don't run this test in parallel to maximize the chance that we get
		// the same pooled buffer for most/all calls.
		largeVal := strings.Repeat("1234567890", 100_000)
		var prev []byte
		for i := 0; i < 20; i++ {
			_, b, err := MarshalValue(largeVal)
			require.NoError(t, err)

			assert.DifferentAddressRanges(t, b, prev)
			prev = b
		}
	})
}

func compareMarshalValueResults(t *testing.T, tc marshalValueTestCase, gotType Type, gotBytes []byte) {
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
