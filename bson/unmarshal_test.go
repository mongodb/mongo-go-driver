// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func TestUnmarshal(t *testing.T) {
	for _, tc := range unmarshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.reg != nil {
				t.Skip() // test requires custom registry
			}
			got := reflect.New(tc.sType).Interface()
			err := Unmarshal(tc.data, got)
			noerr(t, err)
			if !cmp.Equal(got, tc.want) {
				t.Errorf("Did not unmarshal as expected. got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestUnmarshalWithRegistry(t *testing.T) {
	for _, tc := range unmarshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			var reg *bsoncodec.Registry
			if tc.reg != nil {
				reg = tc.reg
			} else {
				reg = DefaultRegistry
			}
			got := reflect.New(tc.sType).Interface()
			err := UnmarshalWithRegistry(reg, tc.data, got)
			noerr(t, err)
			if !cmp.Equal(got, tc.want) {
				t.Errorf("Did not unmarshal as expected. got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestUnmarshalWithContext(t *testing.T) {
	for _, tc := range unmarshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			var reg *bsoncodec.Registry
			if tc.reg != nil {
				reg = tc.reg
			} else {
				reg = DefaultRegistry
			}
			dc := bsoncodec.DecodeContext{Registry: reg}
			got := reflect.New(tc.sType).Interface()
			err := UnmarshalWithContext(dc, tc.data, got)
			noerr(t, err)
			if !cmp.Equal(got, tc.want) {
				t.Errorf("Did not unmarshal as expected. got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestUnmarshalExtJSONWithRegistry(t *testing.T) {
	t.Run("UnmarshalExtJSONWithContext", func(t *testing.T) {
		type teststruct struct{ Foo int }
		var got teststruct
		data := []byte("{\"foo\":1}")
		err := UnmarshalExtJSONWithRegistry(DefaultRegistry, data, true, &got)
		noerr(t, err)
		want := teststruct{1}
		if !cmp.Equal(got, want) {
			t.Errorf("Did not unmarshal as expected. got %v; want %v", got, want)
		}
	})

	t.Run("UnmarshalExtJSONInvalidInput", func(t *testing.T) {
		data := []byte("invalid")
		err := UnmarshalExtJSONWithRegistry(DefaultRegistry, data, true, &M{})
		if err != bsonrw.ErrInvalidJSON {
			t.Fatalf("wanted ErrInvalidJSON, got %v", err)
		}
	})
}

func TestUnmarshalExtJSONWithContext(t *testing.T) {
	t.Run("UnmarshalExtJSONWithContext", func(t *testing.T) {
		type teststruct struct{ Foo int }
		var got teststruct
		data := []byte("{\"foo\":1}")
		dc := bsoncodec.DecodeContext{Registry: DefaultRegistry}
		err := UnmarshalExtJSONWithContext(dc, data, true, &got)
		noerr(t, err)
		want := teststruct{1}
		if !cmp.Equal(got, want) {
			t.Errorf("Did not unmarshal as expected. got %v; want %v", got, want)
		}
	})
}

func TestCachingDecodersNotSharedAcrossRegistries(t *testing.T) {
	// Decoders that have caches for recursive decoder lookup should not be shared across Registry instances. Otherwise,
	// the first DecodeValue call would cache an decoder and a subsequent call would see that decoder even if a
	// different Registry is used.

	// Create a custom Registry that negates BSON int32 values when decoding.
	var decodeInt32 bsoncodec.ValueDecoderFunc = func(_ bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}

		val.SetInt(int64(-1 * i32))
		return nil
	}
	customReg := NewRegistryBuilder().
		RegisterTypeDecoder(tInt32, bsoncodec.ValueDecoderFunc(decodeInt32)).
		Build()

	docBytes := bsoncore.BuildDocumentFromElements(
		nil,
		bsoncore.AppendInt32Element(nil, "x", 1),
	)

	// For all sub-tests, unmarshal docBytes into a struct and assert that value for "x" is 1 when using the default
	// registry and -1 when using the custom registry.
	t.Run("struct", func(t *testing.T) {
		type Struct struct {
			X int32
		}

		var first Struct
		err := Unmarshal(docBytes, &first)
		assert.Nil(t, err, "Unmarshal error: %v", err)
		assert.Equal(t, int32(1), first.X, "expected X value to be 1, got %v", first.X)

		var second Struct
		err = UnmarshalWithRegistry(customReg, docBytes, &second)
		assert.Nil(t, err, "Unmarshal error: %v", err)
		assert.Equal(t, int32(-1), second.X, "expected X value to be -1, got %v", second.X)
	})
	t.Run("pointer", func(t *testing.T) {
		type Struct struct {
			X *int32
		}

		var first Struct
		err := Unmarshal(docBytes, &first)
		assert.Nil(t, err, "Unmarshal error: %v", err)
		assert.Equal(t, int32(1), *first.X, "expected X value to be 1, got %v", *first.X)

		var second Struct
		err = UnmarshalWithRegistry(customReg, docBytes, &second)
		assert.Nil(t, err, "Unmarshal error: %v", err)
		assert.Equal(t, int32(-1), *second.X, "expected X value to be -1, got %v", *second.X)
	})
}

func TestUnmarshalExtJSONWithUndefinedField(t *testing.T) {
	// When unmarshalling, fields that are undefined in the destination struct are skipped.
	// This process must not skip other, defined fields and must not raise errors.
	type expectedResponse struct {
		DefinedField string
	}

	unmarshalExpectedResponse := func(t *testing.T, extJSON string) *expectedResponse {
		t.Helper()
		responseDoc := expectedResponse{}
		err := UnmarshalExtJSON([]byte(extJSON), false, &responseDoc)
		assert.Nil(t, err, "UnmarshalExtJSON error: %v", err)
		return &responseDoc
	}

	testCases := []struct {
		name     string
		testJSON string
	}{
		{
			"no array",
			`{
				"UndefinedField": {"key": 1},
				"DefinedField": "value"
			}`,
		},
		{
			"outer array",
			`{
				"UndefinedField": [{"key": 1}],
				"DefinedField": "value"
			}`,
		},
		{
			"embedded array",
			`{
				"UndefinedField": {"keys": [2]},
				"DefinedField": "value"
			}`,
		},
		{
			"outer array and embedded array",
			`{
				"UndefinedField": [{"keys": [2]}],
				"DefinedField": "value"
			}`,
		},
		{
			"embedded document",
			`{
				"UndefinedField": {"key": {"one": "two"}},
				"DefinedField": "value"
			}`,
		},
		{
			"doubly embedded document",
			`{
				"UndefinedField": {"key": {"one": {"two": "three"}}},
				"DefinedField": "value"
			}`,
		},
		{
			"embedded document and embedded array",
			`{
				"UndefinedField": {"key": {"one": {"two": [3]}}},
				"DefinedField": "value"
			}`,
		},
		{
			"embedded document and embedded array in outer array",
			`{
				"UndefinedField": [{"key": {"one": [3]}}],
				"DefinedField": "value"
			}`,
		},
		{
			"code with scope",
			`{
				"UndefinedField": {"logic": {"$code": "foo", "$scope": {"bar": 1}}},
				"DefinedField": "value"
			}`,
		},
		{
			"embedded array of code with scope",
			`{
				"UndefinedField": {"logic": [{"$code": "foo", "$scope": {"bar": 1}}]},
				"DefinedField": "value"
			}`,
		},
		{
			"type definition embedded document",
			`{
				"UndefinedField": {"myDouble": {"$numberDouble": "1.24"}},
				"DefinedField": "value"
			}`,
		},
		{
			"empty embedded document",
			`{
				"UndefinedField": {"empty": {}},
				"DefinedField": "value"
			}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			responseDoc := unmarshalExpectedResponse(t, tc.testJSON)
			assert.Equal(t, "value", responseDoc.DefinedField, "expected DefinedField to be 'value', got %q", responseDoc.DefinedField)
		})
	}
}
