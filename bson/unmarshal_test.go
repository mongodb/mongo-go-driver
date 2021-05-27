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
	type fooInt struct {
		Foo int
	}

	type fooString struct {
		Foo string
	}

	var cases = []struct {
		name  string
		sType reflect.Type
		want  interface{}
		data  []byte
	}{
		{
			name:  "Small struct",
			sType: reflect.TypeOf(fooInt{}),
			data:  []byte(`{"foo":1}`),
			want:  &fooInt{Foo: 1},
		},
		{
			name:  "Valid surrogate pair",
			sType: reflect.TypeOf(fooString{}),
			data:  []byte(`{"foo":"\uD834\uDd1e"}`),
			want:  &fooString{Foo: "𝄞"},
		},
		{
			name:  "Valid surrogate pair with other values",
			sType: reflect.TypeOf(fooString{}),
			data:  []byte(`{"foo":"abc \uD834\uDd1e 123"}`),
			want:  &fooString{Foo: "abc 𝄞 123"},
		},
		{
			name:  "High surrogate value with no following low surrogate value",
			sType: reflect.TypeOf(fooString{}),
			data:  []byte(`{"foo":"abc \uD834 123"}`),
			want:  &fooString{Foo: "abc � 123"},
		},
		{
			name:  "High surrogate value at end of string",
			sType: reflect.TypeOf(fooString{}),
			data:  []byte(`{"foo":"\uD834"}`),
			want:  &fooString{Foo: "�"},
		},
		{
			name:  "Low surrogate value with no preceeding high surrogate value",
			sType: reflect.TypeOf(fooString{}),
			data:  []byte(`{"foo":"abc \uDd1e 123"}`),
			want:  &fooString{Foo: "abc � 123"},
		},
		{
			name:  "Low surrogate value at end of string",
			sType: reflect.TypeOf(fooString{}),
			data:  []byte(`{"foo":"\uDd1e"}`),
			want:  &fooString{Foo: "�"},
		},
		{
			name:  "High surrogate value with non-surrogate unicode value",
			sType: reflect.TypeOf(fooString{}),
			data:  []byte(`{"foo":"\uD834\u00BF"}`),
			want:  &fooString{Foo: "�¿"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := reflect.New(tc.sType).Interface()
			dc := bsoncodec.DecodeContext{Registry: DefaultRegistry}
			err := UnmarshalExtJSONWithContext(dc, tc.data, true, got)
			noerr(t, err)
			if !cmp.Equal(got, tc.want) {
				t.Errorf("Did not unmarshal as expected. got %+v; want %+v", got, tc.want)
			}
		})
	}
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
	// When unmarshalling extJSON, fields that are undefined in the destination struct are skipped.
	// This process must not skip other, defined fields and must not raise errors.
	type expectedResponse struct {
		DefinedField interface{}
	}

	unmarshalExpectedResponse := func(t *testing.T, extJSON string) *expectedResponse {
		t.Helper()
		responseDoc := expectedResponse{}
		err := UnmarshalExtJSON([]byte(extJSON), false, &responseDoc)
		assert.Nil(t, err, "UnmarshalExtJSON error: %v", err)
		return &responseDoc
	}

	testCases := []struct {
		name          string
		testJSON      string
		expectedValue interface{}
	}{
		{
			"no array",
			`{
				"UndefinedField": {"key": 1},
				"DefinedField": "value"
			}`,
			"value",
		},
		{
			"outer array",
			`{
				"UndefinedField": [{"key": 1}],
				"DefinedField": "value"
			}`,
			"value",
		},
		{
			"embedded array",
			`{
				"UndefinedField": {"keys": [2]},
				"DefinedField": "value"
			}`,
			"value",
		},
		{
			"outer array and embedded array",
			`{
				"UndefinedField": [{"keys": [2]}],
				"DefinedField": "value"
			}`,
			"value",
		},
		{
			"embedded document",
			`{
				"UndefinedField": {"key": {"one": "two"}},
				"DefinedField": "value"
			}`,
			"value",
		},
		{
			"doubly embedded document",
			`{
				"UndefinedField": {"key": {"one": {"two": "three"}}},
				"DefinedField": "value"
			}`,
			"value",
		},
		{
			"embedded document and embedded array",
			`{
				"UndefinedField": {"key": {"one": {"two": [3]}}},
				"DefinedField": "value"
			}`,
			"value",
		},
		{
			"embedded document and embedded array in outer array",
			`{
				"UndefinedField": [{"key": {"one": [3]}}],
				"DefinedField": "value"
			}`,
			"value",
		},
		{
			"code with scope",
			`{
				"UndefinedField": {"logic": {"$code": "foo", "$scope": {"bar": 1}}},
				"DefinedField": "value"
			}`,
			"value",
		},
		{
			"embedded array of code with scope",
			`{
				"UndefinedField": {"logic": [{"$code": "foo", "$scope": {"bar": 1}}]},
				"DefinedField": "value"
			}`,
			"value",
		},
		{
			"type definition embedded document",
			`{
				"UndefinedField": {"myDouble": {"$numberDouble": "1.24"}},
				"DefinedField": "value"
			}`,
			"value",
		},
		{
			"empty embedded document",
			`{
				"UndefinedField": {"empty": {}, "key": 1},
				"DefinedField": "value"
			}`,
			"value",
		},
		{
			"empty object before",
			`{
				"UndefinedField": {},
				"DefinedField": {"value": "a"}
			}`,
			D{{"value", "a"}},
		},
		{
			"empty object after",
			`{
				"DefinedField": {"value": "a"},
				"UndefinedField": {}
			}`,
			D{{"value", "a"}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			responseDoc := unmarshalExpectedResponse(t, tc.testJSON)
			assert.Equal(t, tc.expectedValue, responseDoc.DefinedField, "expected DefinedField to be %v, got %q",
				tc.expectedValue, responseDoc.DefinedField)
		})
	}
}

func TestUnmarshalBSONWithUndefinedField(t *testing.T) {
	// When unmarshalling BSON, fields that are undefined in the destination struct are skipped.
	// This process must not skip other, defined fields and must not raise errors.
	type expectedResponse struct {
		DefinedField string `bson:"DefinedField"`
	}

	createExpectedResponse := func(t *testing.T, doc D) *expectedResponse {
		t.Helper()

		marshalledBSON, err := Marshal(doc)
		assert.Nil(t, err, "error marshalling BSON: %v", err)

		responseDoc := expectedResponse{}
		err = Unmarshal(marshalledBSON, &responseDoc)
		assert.Nil(t, err, "error unmarshalling BSON: %v", err)
		return &responseDoc
	}

	testCases := []struct {
		name     string
		testBSON D
	}{
		{
			"no array",
			D{
				{"UndefinedField", D{
					{"key", 1},
				}},
				{"DefinedField", "value"},
			},
		},
		{
			"outer array",
			D{
				{"UndefinedField", A{D{
					{"key", 1},
				}}},
				{"DefinedField", "value"},
			},
		},
		{
			"embedded array",
			D{
				{"UndefinedField", D{
					{"key", A{1}},
				}},
				{"DefinedField", "value"},
			},
		},
		{
			"outer array and embedded array",
			D{
				{"UndefinedField", A{D{
					{"key", A{1}},
				}}},
				{"DefinedField", "value"},
			},
		},
		{
			"embedded document",
			D{
				{"UndefinedField", D{
					{"key", D{
						{"one", "two"},
					}},
				}},
				{"DefinedField", "value"},
			},
		},
		{
			"doubly embedded document",
			D{
				{"UndefinedField", D{
					{"key", D{
						{"one", D{
							{"two", "three"},
						}},
					}},
				}},
				{"DefinedField", "value"},
			},
		},
		{
			"embedded document and embedded array",
			D{
				{"UndefinedField", D{
					{"key", D{
						{"one", D{
							{"two", A{3}},
						}},
					}},
				}},
				{"DefinedField", "value"},
			},
		},
		{
			"embedded document and embedded array in outer array",
			D{
				{"UndefinedField", A{D{
					{"key", D{
						{"one", A{3}},
					}},
				}}},
				{"DefinedField", "value"},
			},
		},
		{
			"code with scope",
			D{
				{"UndefinedField", D{
					{"logic", D{
						{"$code", "foo"},
						{"$scope", D{
							{"bar", 1},
						}},
					}},
				}},
				{"DefinedField", "value"},
			},
		},
		{
			"embedded array of code with scope",
			D{
				{"UndefinedField", D{
					{"logic", A{D{
						{"$code", "foo"},
						{"$scope", D{
							{"bar", 1},
						}},
					}}},
				}},
				{"DefinedField", "value"},
			},
		},
		{
			"empty embedded document",
			D{
				{"UndefinedField", D{}},
				{"DefinedField", "value"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			responseDoc := createExpectedResponse(t, tc.testBSON)
			assert.Equal(t, "value", responseDoc.DefinedField, "expected DefinedField to be 'value', got %q", responseDoc.DefinedField)
		})
	}
}
