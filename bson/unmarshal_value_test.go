// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

func TestUnmarshalValue(t *testing.T) {
	t.Parallel()

	unmarshalValueTestCases := newMarshalValueTestCases(t)

	t.Run("UnmarshalValue", func(t *testing.T) {
		t.Parallel()

		for _, tc := range unmarshalValueTestCases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				gotValue := reflect.New(reflect.TypeOf(tc.val))
				err := UnmarshalValue(tc.bsontype, tc.bytes, gotValue.Interface())
				assert.Nil(t, err, "UnmarshalValueWithRegistry error: %v", err)
				assert.Equal(t, tc.val, gotValue.Elem().Interface(), "value mismatch; expected %s, got %s", tc.val, gotValue.Elem())
			})
		}
	})
}

func TestInitializedPointerDataWithBSONNull(t *testing.T) {
	// Set up the test case with initialized pointers.
	tc := unmarshalBehaviorTestCase{
		BSONValuePtrTracker: &unmarshalBSONValueCallTracker{},
		BSONPtrTracker:      &unmarshalBSONCallTracker{},
	}
	// Create BSON data where the '*_ptr_tracker' fields are explicitly set to
	// null.
	bytes := docToBytes(D{
		{Key: "bv_ptr_tracker", Value: nil},
		{Key: "b_ptr_tracker", Value: nil},
	})
	// Unmarshal the BSON data into the test case struct. This should set the
	// pointer fields to nil due to the BSON null value.
	err := Unmarshal(bytes, &tc)
	require.NoError(t, err)
	assert.Nil(t, tc.BSONValuePtrTracker)
	assert.Nil(t, tc.BSONPtrTracker)
}

// tests covering GODRIVER-2779
func BenchmarkSliceCodecUnmarshal(b *testing.B) {
	benchmarks := []struct {
		name     string
		bsontype Type
		bytes    []byte
	}{
		{
			name:     "SliceCodec binary",
			bsontype: TypeBinary,
			bytes:    bsoncore.AppendBinary(nil, TypeBinaryGeneric, []byte(strings.Repeat("t", 4096))),
		},
		{
			name:     "SliceCodec string",
			bsontype: TypeString,
			bytes:    bsoncore.AppendString(nil, strings.Repeat("t", 4096)),
		},
	}
	reg := NewRegistry()
	reg.RegisterTypeDecoder(reflect.TypeOf([]byte{}), &sliceCodec{})
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				dec := NewDecoder(nil)
				dec.SetRegistry(reg)
				for pb.Next() {
					dec.Reset(newValueReader(bm.bsontype, bytes.NewReader(bm.bytes)))
					err := dec.Decode(&[]byte{})
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
