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
	unmarshalValueTestCases := unmarshalValueTestCases(t)

	t.Run("UnmarshalValue", func(t *testing.T) {
		for _, tc := range unmarshalValueTestCases {
			t.Run(tc.name, func(t *testing.T) {
				gotValue := reflect.New(reflect.TypeOf(tc.val))
				err := UnmarshalValue(tc.bsontype, tc.bytes, gotValue.Interface())
				assert.Nil(t, err, "UnmarshalValueWithRegistry error: %v", err)
				assert.Equal(t, tc.val, gotValue.Elem().Interface(), "value mismatch; expected %s, got %s", tc.val, gotValue.Elem())
			})
		}
	})
	t.Run("UnmarshalValueWithRegistry with DefaultRegistry", func(t *testing.T) {
		for _, tc := range unmarshalValueTestCases {
			t.Run(tc.name, func(t *testing.T) {
				gotValue := reflect.New(reflect.TypeOf(tc.val))
				err := UnmarshalValueWithRegistry(DefaultRegistry, tc.bsontype, tc.bytes, gotValue.Interface())
				assert.Nil(t, err, "UnmarshalValueWithRegistry error: %v", err)
				assert.Equal(t, tc.val, gotValue.Elem().Interface(), "value mismatch; expected %s, got %s", tc.val, gotValue.Elem())
			})
		}
	})
	// tests covering GODRIVER-2779
	t.Run("UnmarshalValueWithRegistry with custom registry", func(t *testing.T) {
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
			t.Run(tc.name, func(t *testing.T) {
				gotValue := reflect.New(reflect.TypeOf(tc.val))
				err := UnmarshalValueWithRegistry(rb, tc.bsontype, tc.bytes, gotValue.Interface())
				assert.Nil(t, err, "UnmarshalValueWithRegistry error: %v", err)
				assert.Equal(t, tc.val, gotValue.Elem().Interface(), "value mismatch; expected %s, got %s", tc.val, gotValue.Elem())
			})
		}
	})
}

// tests covering GODRIVER-2779
func BenchmarkSliceCodec_Unmarshal(b *testing.B) {
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
			var gotValue []byte
			for n := 0; n < b.N; n++ {
				err := UnmarshalValueWithRegistry(rb, bm.bsontype, bm.bytes, &gotValue)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
