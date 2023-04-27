// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"reflect"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
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
	t.Run("UnmarshalValueWithRegistry with DefaultRegistry", func(t *testing.T) {
		t.Parallel()

		for _, tc := range unmarshalValueTestCases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				gotValue := reflect.New(reflect.TypeOf(tc.val))
				err := UnmarshalValueWithRegistry(DefaultRegistry, tc.bsontype, tc.bytes, gotValue.Interface())
				assert.Nil(t, err, "UnmarshalValueWithRegistry error: %v", err)
				assert.Equal(t, tc.val, gotValue.Elem().Interface(), "value mismatch; expected %s, got %s", tc.val, gotValue.Elem())
			})
		}
	})
	// tests covering GODRIVER-2779
	t.Run("UnmarshalValueWithRegistry with custom registry", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name     string
			val      interface{}
			bsontype bsontype.Type
			bytes    []byte
		}{
			{
				name:     "SliceCodec binary",
				val:      []byte("hello world"),
				bsontype: bsontype.Binary,
				bytes:    bsoncore.AppendBinary(nil, bsontype.BinaryGeneric, []byte("hello world")),
			},
			{
				name:     "SliceCodec string",
				val:      []byte("hello world"),
				bsontype: bsontype.String,
				bytes:    bsoncore.AppendString(nil, "hello world"),
			},
		}
		rb := NewRegistryBuilder().RegisterTypeDecoder(reflect.TypeOf([]byte{}), bsoncodec.NewSliceCodec()).Build()
		for _, tc := range testCases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				gotValue := reflect.New(reflect.TypeOf(tc.val))
				err := UnmarshalValueWithRegistry(rb, tc.bsontype, tc.bytes, gotValue.Interface())
				assert.Nil(t, err, "UnmarshalValueWithRegistry error: %v", err)
				assert.Equal(t, tc.val, gotValue.Elem().Interface(), "value mismatch; expected %s, got %s", tc.val, gotValue.Elem())
			})
		}
	})
}

// tests covering GODRIVER-2779
func BenchmarkSliceCodecUnmarshal(b *testing.B) {
	benchmarks := []struct {
		name     string
		bsontype bsontype.Type
		bytes    []byte
	}{
		{
			name:     "SliceCodec binary",
			bsontype: bsontype.Binary,
			bytes:    bsoncore.AppendBinary(nil, bsontype.BinaryGeneric, []byte(strings.Repeat("t", 4096))),
		},
		{
			name:     "SliceCodec string",
			bsontype: bsontype.String,
			bytes:    bsoncore.AppendString(nil, strings.Repeat("t", 4096)),
		},
	}
	rb := NewRegistryBuilder().RegisterTypeDecoder(reflect.TypeOf([]byte{}), bsoncodec.NewSliceCodec()).Build()
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					err := UnmarshalValueWithRegistry(rb, bm.bsontype, bm.bytes, &[]byte{})
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
