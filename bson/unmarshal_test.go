// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"math/rand"
	"reflect"
	"testing"
	"unsafe"

	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func TestUnmarshal(t *testing.T) {
	for _, tc := range unmarshalingTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the test data so we can modify it later.
			data := make([]byte, len(tc.data))
			copy(data, tc.data)

			// Assert that unmarshaling the input data results in the expected value.
			got := reflect.New(tc.sType).Interface()
			err := Unmarshal(data, got)
			noerr(t, err)
			assert.Equal(t, tc.want, got, "Did not unmarshal as expected.")

			// Fill the input data slice with random bytes and then assert that the result still
			// matches the expected value.
			_, err = rand.Read(data)
			noerr(t, err)
			assert.Equal(t, tc.want, got, "unmarshaled value does not match expected after modifying the input bytes")
		})
	}
}

func TestUnmarshalWithRegistry(t *testing.T) {
	for _, tc := range unmarshalingTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the test data so we can modify it later.
			data := make([]byte, len(tc.data))
			copy(data, tc.data)

			// Assert that unmarshaling the input data results in the expected value.
			got := reflect.New(tc.sType).Interface()
			err := UnmarshalWithRegistry(DefaultRegistry, data, got)
			noerr(t, err)
			assert.Equal(t, tc.want, got, "Did not unmarshal as expected.")

			// Fill the input data slice with random bytes and then assert that the result still
			// matches the expected value.
			_, err = rand.Read(data)
			noerr(t, err)
			assert.Equal(t, tc.want, got, "unmarshaled value does not match expected after modifying the input bytes")
		})
	}
}

func TestUnmarshalWithContext(t *testing.T) {
	for _, tc := range unmarshalingTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the test data so we can modify it later.
			data := make([]byte, len(tc.data))
			copy(data, tc.data)

			// Assert that unmarshaling the input data results in the expected value.
			dc := bsoncodec.DecodeContext{Registry: DefaultRegistry}
			got := reflect.New(tc.sType).Interface()
			err := UnmarshalWithContext(dc, data, got)
			noerr(t, err)
			assert.Equal(t, tc.want, got, "Did not unmarshal as expected.")

			// Fill the input data slice with random bytes and then assert that the result still
			// matches the expected value.
			_, err = rand.Read(data)
			noerr(t, err)
			assert.Equal(t, tc.want, got, "unmarshaled value does not match expected after modifying the input bytes")
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
		assert.Equal(t, want, got, "Did not unmarshal as expected.")
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

	type fooBytes struct {
		Foo []byte
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
			name:  "Low surrogate value with no preceding high surrogate value",
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
		// GODRIVER-2311
		// Test that ExtJSON-encoded binary unmarshals correctly to a bson.D and that the
		// unmarshaled value does not reference the same underlying byte array as the input.
		{
			name:  "bson.D with binary",
			sType: reflect.TypeOf(D{}),
			data:  []byte(`{"foo": {"$binary": {"subType": "0", "base64": "AAECAwQF"}}}`),
			want:  &D{{"foo", primitive.Binary{Subtype: 0, Data: []byte{0, 1, 2, 3, 4, 5}}}},
		},
		// GODRIVER-2311
		// Test that ExtJSON-encoded binary unmarshals correctly to a struct and that the
		// unmarshaled value does not reference thesame  underlying byte array as the input.
		{
			name:  "struct with binary",
			sType: reflect.TypeOf(fooBytes{}),
			data:  []byte(`{"foo": {"$binary": {"subType": "0", "base64": "AAECAwQF"}}}`),
			want:  &fooBytes{Foo: []byte{0, 1, 2, 3, 4, 5}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the test data so we can modify it later.
			data := make([]byte, len(tc.data))
			copy(data, tc.data)

			// Assert that unmarshaling the input data results in the expected value.
			got := reflect.New(tc.sType).Interface()
			dc := bsoncodec.DecodeContext{Registry: DefaultRegistry}
			err := UnmarshalExtJSONWithContext(dc, data, true, got)
			noerr(t, err)
			assert.Equal(t, tc.want, got, "Did not unmarshal as expected.")

			// Fill the input data slice with random bytes and then assert that the result still
			// matches the expected value.
			_, err = rand.Read(data)
			noerr(t, err)
			assert.Equal(t, tc.want, got, "unmarshaled value does not match expected after modifying the input bytes")
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
		RegisterTypeDecoder(tInt32, decodeInt32).
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

// GODRIVER-2311
// Assert that unmarshaled values containing byte slices do not reference the same underlying byte
// array as the BSON input data byte slice.
func TestUnmarshalByteSlicesUseDistinctArrays(t *testing.T) {
	type fooBytes struct {
		Foo []byte
	}

	type myBytes []byte
	type fooMyBytes struct {
		Foo myBytes
	}

	type fooBinary struct {
		Foo primitive.Binary
	}

	type fooObjectID struct {
		Foo primitive.ObjectID
	}

	type fooDBPointer struct {
		Foo primitive.DBPointer
	}

	testCases := []struct {
		description string
		data        []byte
		sType       reflect.Type
		want        interface{}

		// getByteSlice returns the byte slice from the unmarshaled value, allowing the test to
		// inspect the addresses of the underlying byte array.
		getByteSlice func(interface{}) []byte
	}{
		{
			description: "struct with byte slice",
			data: docToBytes(fooBytes{
				Foo: []byte{0, 1, 2, 3, 4, 5},
			}),
			sType: reflect.TypeOf(fooBytes{}),
			want: &fooBytes{
				Foo: []byte{0, 1, 2, 3, 4, 5},
			},
			getByteSlice: func(val interface{}) []byte {
				return (*(val.(*fooBytes))).Foo
			},
		},
		{
			description: "bson.D with byte slice",
			data: docToBytes(D{
				{"foo", []byte{0, 1, 2, 3, 4, 5}},
			}),
			sType: reflect.TypeOf(D{}),
			want: &D{
				{"foo", primitive.Binary{Subtype: 0, Data: []byte{0, 1, 2, 3, 4, 5}}},
			},
			getByteSlice: func(val interface{}) []byte {
				return (*(val.(*D)))[0].Value.(primitive.Binary).Data
			},
		},
		{
			description: "struct with custom byte slice type",
			data: docToBytes(fooMyBytes{
				Foo: myBytes{0, 1, 2, 3, 4, 5},
			}),
			sType: reflect.TypeOf(fooMyBytes{}),
			want: &fooMyBytes{
				Foo: myBytes{0, 1, 2, 3, 4, 5},
			},
			getByteSlice: func(val interface{}) []byte {
				return (*(val.(*fooMyBytes))).Foo
			},
		},
		{
			description: "bson.D with custom byte slice type",
			data: docToBytes(D{
				{"foo", myBytes{0, 1, 2, 3, 4, 5}},
			}),
			sType: reflect.TypeOf(D{}),
			want: &D{
				{"foo", primitive.Binary{Subtype: 0, Data: myBytes{0, 1, 2, 3, 4, 5}}},
			},
			getByteSlice: func(val interface{}) []byte {
				return (*(val.(*D)))[0].Value.(primitive.Binary).Data
			},
		},
		{
			description: "struct with primitive.Binary",
			data: docToBytes(fooBinary{
				Foo: primitive.Binary{Subtype: 0, Data: []byte{0, 1, 2, 3, 4, 5}},
			}),
			sType: reflect.TypeOf(fooBinary{}),
			want: &fooBinary{
				Foo: primitive.Binary{Subtype: 0, Data: []byte{0, 1, 2, 3, 4, 5}},
			},
			getByteSlice: func(val interface{}) []byte {
				return (*(val.(*fooBinary))).Foo.Data
			},
		},
		{
			description: "bson.D with primitive.Binary",
			data: docToBytes(D{
				{"foo", primitive.Binary{Subtype: 0, Data: []byte{0, 1, 2, 3, 4, 5}}},
			}),
			sType: reflect.TypeOf(D{}),
			want: &D{
				{"foo", primitive.Binary{Subtype: 0, Data: []byte{0, 1, 2, 3, 4, 5}}},
			},
			getByteSlice: func(val interface{}) []byte {
				return (*(val.(*D)))[0].Value.(primitive.Binary).Data
			},
		},
		{
			description: "struct with primitive.ObjectID",
			data: docToBytes(fooObjectID{
				Foo: primitive.ObjectID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
			}),
			sType: reflect.TypeOf(fooObjectID{}),
			want: &fooObjectID{
				Foo: primitive.ObjectID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
			},
			getByteSlice: func(val interface{}) []byte {
				return (*(val.(*fooObjectID))).Foo[:]
			},
		},
		{
			description: "bson.D with primitive.ObjectID",
			data: docToBytes(D{
				{"foo", primitive.ObjectID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}},
			}),
			sType: reflect.TypeOf(D{}),
			want: &D{
				{"foo", primitive.ObjectID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}},
			},
			getByteSlice: func(val interface{}) []byte {
				oid := (*(val.(*D)))[0].Value.(primitive.ObjectID)
				return oid[:]
			},
		},
		{
			description: "struct with primitive.DBPointer",
			data: docToBytes(fooDBPointer{
				Foo: primitive.DBPointer{
					DB:      "test",
					Pointer: primitive.ObjectID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
				},
			}),
			sType: reflect.TypeOf(fooDBPointer{}),
			want: &fooDBPointer{
				Foo: primitive.DBPointer{
					DB:      "test",
					Pointer: primitive.ObjectID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
				},
			},
			getByteSlice: func(val interface{}) []byte {
				return (*(val.(*fooDBPointer))).Foo.Pointer[:]
			},
		},
		{
			description: "bson.D with primitive.DBPointer",
			data: docToBytes(D{
				{"foo", primitive.DBPointer{
					DB:      "test",
					Pointer: primitive.ObjectID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
				}},
			}),
			sType: reflect.TypeOf(D{}),
			want: &D{
				{"foo", primitive.DBPointer{
					DB:      "test",
					Pointer: primitive.ObjectID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
				}},
			},
			getByteSlice: func(val interface{}) []byte {
				oid := (*(val.(*D)))[0].Value.(primitive.DBPointer).Pointer
				return oid[:]
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			// Make a copy of the test data so we can modify it later.
			data := make([]byte, len(tc.data))
			copy(data, tc.data)

			// Assert that unmarshaling the input data results in the expected value.
			got := reflect.New(tc.sType).Interface()
			err := Unmarshal(data, got)
			noerr(t, err)
			assert.Equal(t, tc.want, got, "unmarshaled value does not match the expected value")

			// Fill the input data slice with random bytes and then assert that the result still
			// matches the expected value.
			_, err = rand.Read(data)
			noerr(t, err)
			assert.Equal(t, tc.want, got, "unmarshaled value does not match expected after modifying the input bytes")

			// Assert that the byte slice in the unmarshaled value does not share any memory
			// addresses with the input byte slice.
			assertDifferentArrays(t, data, tc.getByteSlice(got))
		})
	}
}

// assertDifferentArrays asserts that two byte slices reference distinct memory ranges, meaning
// they reference different underlying byte arrays.
func assertDifferentArrays(t *testing.T, a, b []byte) {
	// Find the start and end memory addresses for the underlying byte array for each input byte
	// slice.
	sliceAddrRange := func(b []byte) (uintptr, uintptr) {
		sh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
		return sh.Data, sh.Data + uintptr(sh.Cap-1)
	}
	aStart, aEnd := sliceAddrRange(a)
	bStart, bEnd := sliceAddrRange(b)

	// If "b" starts after "a" ends or "a" starts after "b" ends, there is no overlap.
	if bStart > aEnd || aStart > bEnd {
		return
	}

	// Otherwise, calculate the overlap start and end and print the memory overlap error message.
	min := func(a, b uintptr) uintptr {
		if a < b {
			return a
		}
		return b
	}
	max := func(a, b uintptr) uintptr {
		if a > b {
			return a
		}
		return b
	}
	overlapLow := max(aStart, bStart)
	overlapHigh := min(aEnd, bEnd)

	t.Errorf("Byte slices point to the same the same underlying byte array:\n"+
		"\ta addresses:\t%d ... %d\n"+
		"\tb addresses:\t%d ... %d\n"+
		"\toverlap:\t%d ... %d",
		aStart, aEnd,
		bStart, bEnd,
		overlapLow, overlapHigh)
}
