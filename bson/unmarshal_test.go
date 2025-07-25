// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"math/rand"
	"reflect"
	"sync"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
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
	t.Parallel()

	testCases := []struct {
		name     string
		val      any
		bsontype Type
		bytes    []byte
	}{
		{
			name:     "SliceCodec binary",
			val:      []byte("hello world"),
			bsontype: TypeBinary,
			bytes:    bsoncore.AppendBinary(nil, TypeBinaryGeneric, []byte("hello world")),
		},
		{
			name:     "SliceCodec string",
			val:      []byte("hello world"),
			bsontype: TypeString,
			bytes:    bsoncore.AppendString(nil, "hello world"),
		},
	}
	reg := NewRegistry()
	reg.RegisterTypeDecoder(reflect.TypeOf([]byte{}), &sliceCodec{})
	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Assert that unmarshaling the input data results in the expected value.
			gotValue := reflect.New(reflect.TypeOf(tc.val))
			dec := NewDecoder(newValueReader(tc.bsontype, bytes.NewReader(tc.bytes)))
			dec.SetRegistry(reg)
			err := dec.Decode(gotValue.Interface())
			noerr(t, err)
			assert.Equal(t, tc.val, gotValue.Elem().Interface(), "value mismatch; expected %s, got %s", tc.val, gotValue.Elem())

			// Fill the input data slice with random bytes and then assert that the result still
			// matches the expected value.
			_, err = rand.Read(tc.bytes)
			noerr(t, err)
			assert.Equal(t, tc.val, gotValue.Elem().Interface(), "unmarshaled value does not match expected after modifying the input bytes")
		})
	}
}

func TestCachingDecodersNotSharedAcrossRegistries(t *testing.T) {
	// Decoders that have caches for recursive decoder lookup should not be shared across Registry instances. Otherwise,
	// the first DecodeValue call would cache an decoder and a subsequent call would see that decoder even if a
	// different Registry is used.

	// Create a custom Registry that negates BSON int32 values when decoding.
	var decodeInt32 ValueDecoderFunc = func(_ DecodeContext, vr ValueReader, val reflect.Value) error {
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}

		val.SetInt(int64(-1 * i32))
		return nil
	}
	customReg := NewRegistry()
	customReg.RegisterTypeDecoder(tInt32, decodeInt32)

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
		dec := NewDecoder(NewDocumentReader(bytes.NewReader(docBytes)))
		dec.SetRegistry(customReg)
		err = dec.Decode(&second)
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
		dec := NewDecoder(NewDocumentReader(bytes.NewReader(docBytes)))
		dec.SetRegistry(customReg)
		err = dec.Decode(&second)
		assert.Nil(t, err, "Unmarshal error: %v", err)
		assert.Equal(t, int32(-1), *second.X, "expected X value to be -1, got %v", *second.X)
	})
}

func TestUnmarshalExtJSONWithUndefinedField(t *testing.T) {
	// When unmarshalling extJSON, fields that are undefined in the destination struct are skipped.
	// This process must not skip other, defined fields and must not raise errors.
	type expectedResponse struct {
		DefinedField any
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
		expectedValue any
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

func TestUnmarshalInterface(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		stub func() ([]byte, any, func(*testing.T))
	}
	testCases := []testCase{
		{
			name: "struct with interface containing a concrete value",
			stub: func() ([]byte, any, func(*testing.T)) {
				type testStruct struct {
					Value any
				}
				var value string

				data := docToBytes(struct {
					Value string
				}{
					Value: "foo",
				})

				receiver := testStruct{&value}

				check := func(t *testing.T) {
					t.Helper()
					assert.Equal(t, "foo", value)
				}

				return data, &receiver, check
			},
		},
		{
			name: "struct with interface containing a struct",
			stub: func() ([]byte, any, func(*testing.T)) {
				type demo struct {
					Data string
				}

				type testStruct struct {
					Value any
				}
				var value demo

				data := docToBytes(struct {
					Value demo
				}{
					Value: demo{"foo"},
				})

				receiver := testStruct{&value}

				check := func(t *testing.T) {
					t.Helper()
					assert.Equal(t, "foo", value.Data)
				}

				return data, &receiver, check
			},
		},
		{
			name: "struct with interface containing a slice",
			stub: func() ([]byte, any, func(*testing.T)) {
				type testStruct struct {
					Values any
				}
				var values []string

				data := docToBytes(struct {
					Values []string
				}{
					Values: []string{"foo", "bar"},
				})

				receiver := testStruct{&values}

				check := func(t *testing.T) {
					t.Helper()
					assert.Equal(t, []string{"foo", "bar"}, values)
				}

				return data, &receiver, check
			},
		},
		{
			name: "struct with interface containing an array",
			stub: func() ([]byte, any, func(*testing.T)) {
				type testStruct struct {
					Values any
				}
				var values [2]string

				data := docToBytes(struct {
					Values []string
				}{
					Values: []string{"foo", "bar"},
				})

				receiver := testStruct{&values}

				check := func(t *testing.T) {
					t.Helper()
					assert.Equal(t, [2]string{"foo", "bar"}, values)
				}

				return data, &receiver, check
			},
		},
		{
			name: "struct with interface array containing concrete values",
			stub: func() ([]byte, any, func(*testing.T)) {
				type testStruct struct {
					Values [3]any
				}
				var str string
				var i, j int

				data := docToBytes(struct {
					Values []any
				}{
					Values: []any{"foo", 42, nil},
				})

				receiver := testStruct{[3]any{&str, &i, &j}}

				check := func(t *testing.T) {
					t.Helper()
					assert.Equal(t, "foo", str)
					assert.Equal(t, 42, i)
					assert.Equal(t, 0, j)
					assert.Equal(t, testStruct{[3]any{&str, &i, nil}}, receiver)
				}

				return data, &receiver, check
			},
		},
		{
			name: "overwriting prepopulated slice",
			stub: func() ([]byte, any, func(*testing.T)) {
				type testStruct struct {
					Values []any
				}
				data := docToBytes(struct {
					Values []any
				}{
					Values: []any{1, 2, 3},
				})

				receiver := testStruct{[]any{7, 8}}

				check := func(t *testing.T) {
					t.Helper()
					assert.Equal(t, testStruct{[]any{1, 2, int32(3)}}, receiver)
				}

				return data, &receiver, check
			},
		},
	}
	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data, receiver, check := tc.stub()
			err := Unmarshal(data, receiver)
			noerr(t, err)
			check(t)
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
		Foo Binary
	}

	type fooObjectID struct {
		Foo ObjectID
	}

	type fooDBPointer struct {
		Foo DBPointer
	}

	testCases := []struct {
		description string
		data        []byte
		sType       reflect.Type
		want        any

		// getByteSlice returns the byte slice from the unmarshaled value, allowing the test to
		// inspect the addresses of the underlying byte array.
		getByteSlice func(any) []byte
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
			getByteSlice: func(val any) []byte {
				return (val.(*fooBytes)).Foo
			},
		},
		{
			description: "bson.D with byte slice",
			data: docToBytes(D{
				{"foo", []byte{0, 1, 2, 3, 4, 5}},
			}),
			sType: reflect.TypeOf(D{}),
			want: &D{
				{"foo", Binary{Subtype: 0, Data: []byte{0, 1, 2, 3, 4, 5}}},
			},
			getByteSlice: func(val any) []byte {
				return (*(val.(*D)))[0].Value.(Binary).Data
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
			getByteSlice: func(val any) []byte {
				return (val.(*fooMyBytes)).Foo
			},
		},
		{
			description: "bson.D with custom byte slice type",
			data: docToBytes(D{
				{"foo", myBytes{0, 1, 2, 3, 4, 5}},
			}),
			sType: reflect.TypeOf(D{}),
			want: &D{
				{"foo", Binary{Subtype: 0, Data: myBytes{0, 1, 2, 3, 4, 5}}},
			},
			getByteSlice: func(val any) []byte {
				return (*(val.(*D)))[0].Value.(Binary).Data
			},
		},
		{
			description: "struct with Binary",
			data: docToBytes(fooBinary{
				Foo: Binary{Subtype: 0, Data: []byte{0, 1, 2, 3, 4, 5}},
			}),
			sType: reflect.TypeOf(fooBinary{}),
			want: &fooBinary{
				Foo: Binary{Subtype: 0, Data: []byte{0, 1, 2, 3, 4, 5}},
			},
			getByteSlice: func(val any) []byte {
				return (val.(*fooBinary)).Foo.Data
			},
		},
		{
			description: "bson.D with Binary",
			data: docToBytes(D{
				{"foo", Binary{Subtype: 0, Data: []byte{0, 1, 2, 3, 4, 5}}},
			}),
			sType: reflect.TypeOf(D{}),
			want: &D{
				{"foo", Binary{Subtype: 0, Data: []byte{0, 1, 2, 3, 4, 5}}},
			},
			getByteSlice: func(val any) []byte {
				return (*(val.(*D)))[0].Value.(Binary).Data
			},
		},
		{
			description: "struct with ObjectID",
			data: docToBytes(fooObjectID{
				Foo: ObjectID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
			}),
			sType: reflect.TypeOf(fooObjectID{}),
			want: &fooObjectID{
				Foo: ObjectID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
			},
			getByteSlice: func(val any) []byte {
				return (val.(*fooObjectID)).Foo[:]
			},
		},
		{
			description: "bson.D with ObjectID",
			data: docToBytes(D{
				{"foo", ObjectID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}},
			}),
			sType: reflect.TypeOf(D{}),
			want: &D{
				{"foo", ObjectID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}},
			},
			getByteSlice: func(val any) []byte {
				oid := (*(val.(*D)))[0].Value.(ObjectID)
				return oid[:]
			},
		},
		{
			description: "struct with DBPointer",
			data: docToBytes(fooDBPointer{
				Foo: DBPointer{
					DB:      "test",
					Pointer: ObjectID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
				},
			}),
			sType: reflect.TypeOf(fooDBPointer{}),
			want: &fooDBPointer{
				Foo: DBPointer{
					DB:      "test",
					Pointer: ObjectID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
				},
			},
			getByteSlice: func(val any) []byte {
				return (val.(*fooDBPointer)).Foo.Pointer[:]
			},
		},
		{
			description: "bson.D with DBPointer",
			data: docToBytes(D{
				{"foo", DBPointer{
					DB:      "test",
					Pointer: ObjectID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
				}},
			}),
			sType: reflect.TypeOf(D{}),
			want: &D{
				{"foo", DBPointer{
					DB:      "test",
					Pointer: ObjectID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
				}},
			},
			getByteSlice: func(val any) []byte {
				oid := (*(val.(*D)))[0].Value.(DBPointer).Pointer
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
			assert.DifferentAddressRanges(t, data, tc.getByteSlice(got))
		})
	}
}

func TestUnmarshalConcurrently(t *testing.T) {
	t.Parallel()

	const size = 10_000

	data := []byte{16, 0, 0, 0, 10, 108, 97, 115, 116, 101, 114, 114, 111, 114, 0, 0}
	wg := sync.WaitGroup{}
	wg.Add(size)
	for i := 0; i < size; i++ {
		go func() {
			defer wg.Done()
			var res struct{ LastError error }
			_ = Unmarshal(data, &res)
		}()
	}
	wg.Wait()
}
