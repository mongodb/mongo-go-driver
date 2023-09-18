// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var tInt32 = reflect.TypeOf(int32(0))

func TestMarshalAppendWithRegistry(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			dst := make([]byte, 0, 1024)
			var reg *bsoncodec.Registry
			if tc.reg != nil {
				reg = tc.reg
			} else {
				reg = DefaultRegistry
			}
			got, err := MarshalAppendWithRegistry(reg, dst, tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", got, tc.want)
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}

func TestMarshalAppendWithContext(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			dst := make([]byte, 0, 1024)
			var reg *bsoncodec.Registry
			if tc.reg != nil {
				reg = tc.reg
			} else {
				reg = DefaultRegistry
			}
			ec := bsoncodec.EncodeContext{Registry: reg}
			got, err := MarshalAppendWithContext(ec, dst, tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", got, tc.want)
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}

func TestMarshalWithRegistry(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			var reg *bsoncodec.Registry
			if tc.reg != nil {
				reg = tc.reg
			} else {
				reg = DefaultRegistry
			}
			got, err := MarshalWithRegistry(reg, tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", got, tc.want)
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}

func TestMarshalWithContext(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			var reg *bsoncodec.Registry
			if tc.reg != nil {
				reg = tc.reg
			} else {
				reg = DefaultRegistry
			}
			ec := bsoncodec.EncodeContext{Registry: reg}
			got, err := MarshalWithContext(ec, tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", got, tc.want)
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}

func TestMarshalAppend(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.reg != nil {
				t.Skip() // test requires custom registry
			}
			dst := make([]byte, 0, 1024)
			got, err := MarshalAppend(dst, tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", got, tc.want)
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}

func TestMarshalExtJSONAppendWithContext(t *testing.T) {
	t.Run("MarshalExtJSONAppendWithContext", func(t *testing.T) {
		dst := make([]byte, 0, 1024)
		type teststruct struct{ Foo int }
		val := teststruct{1}
		ec := bsoncodec.EncodeContext{Registry: DefaultRegistry}
		got, err := MarshalExtJSONAppendWithContext(ec, dst, val, true, false)
		noerr(t, err)
		want := []byte(`{"foo":{"$numberInt":"1"}}`)
		if !bytes.Equal(got, want) {
			t.Errorf("Bytes are not equal. got %v; want %v", got, want)
			t.Errorf("Bytes:\n%s\n%s", got, want)
		}
	})
}

func TestMarshalExtJSONWithContext(t *testing.T) {
	t.Run("MarshalExtJSONWithContext", func(t *testing.T) {
		type teststruct struct{ Foo int }
		val := teststruct{1}
		ec := bsoncodec.EncodeContext{Registry: DefaultRegistry}
		got, err := MarshalExtJSONWithContext(ec, val, true, false)
		noerr(t, err)
		want := []byte(`{"foo":{"$numberInt":"1"}}`)
		if !bytes.Equal(got, want) {
			t.Errorf("Bytes are not equal. got %v; want %v", got, want)
			t.Errorf("Bytes:\n%s\n%s", got, want)
		}
	})
}

func TestMarshal_roundtripFromBytes(t *testing.T) {
	before := []byte{
		// length
		0x1c, 0x0, 0x0, 0x0,

		// --- begin array ---

		// type - document
		0x3,
		// key - "foo"
		0x66, 0x6f, 0x6f, 0x0,

		// length
		0x12, 0x0, 0x0, 0x0,
		// type - string
		0x2,
		// key - "bar"
		0x62, 0x61, 0x72, 0x0,
		// value - string length
		0x4, 0x0, 0x0, 0x0,
		// value - "baz"
		0x62, 0x61, 0x7a, 0x0,

		// null terminator
		0x0,

		// --- end array ---

		// null terminator
		0x0,
	}

	var doc D
	require.NoError(t, Unmarshal(before, &doc))

	after, err := Marshal(doc)
	require.NoError(t, err)

	require.True(t, bytes.Equal(before, after))
}

func TestMarshal_roundtripFromDoc(t *testing.T) {
	before := D{
		{"foo", "bar"},
		{"baz", int64(-27)},
		{"bing", A{nil, primitive.Regex{Pattern: "word", Options: "i"}}},
	}

	b, err := Marshal(before)
	require.NoError(t, err)

	var after D
	require.NoError(t, Unmarshal(b, &after))

	if !cmp.Equal(after, before) {
		t.Errorf("Documents to not match. got %v; want %v", after, before)
	}
}

func TestCachingEncodersNotSharedAcrossRegistries(t *testing.T) {
	// Encoders that have caches for recursive encoder lookup should not be shared across Registry instances. Otherwise,
	// the first EncodeValue call would cache an encoder and a subsequent call would see that encoder even if a
	// different Registry is used.

	// Create a custom Registry that negates int32 values when encoding.
	var encodeInt32 bsoncodec.ValueEncoderFunc = func(_ bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
		if val.Kind() != reflect.Int32 {
			return fmt.Errorf("expected kind to be int32, got %v", val.Kind())
		}

		return vw.WriteInt32(int32(val.Int()) * -1)
	}
	customReg := NewRegistryBuilder().
		RegisterTypeEncoder(tInt32, encodeInt32).
		Build()

	// Helper function to run the test and make assertions. The provided original value should result in the document
	// {"x": {$numberInt: 1}} when marshalled with the default registry.
	verifyResults := func(t *testing.T, original interface{}) {
		// Marshal using the default and custom registries. Assert that the result is {x: 1} and {x: -1}, respectively.

		first, err := Marshal(original)
		assert.Nil(t, err, "Marshal error: %v", err)
		expectedFirst := Raw(bsoncore.BuildDocumentFromElements(
			nil,
			bsoncore.AppendInt32Element(nil, "x", 1),
		))
		assert.Equal(t, expectedFirst, Raw(first), "expected document %v, got %v", expectedFirst, Raw(first))

		second, err := MarshalWithRegistry(customReg, original)
		assert.Nil(t, err, "Marshal error: %v", err)
		expectedSecond := Raw(bsoncore.BuildDocumentFromElements(
			nil,
			bsoncore.AppendInt32Element(nil, "x", -1),
		))
		assert.Equal(t, expectedSecond, Raw(second), "expected document %v, got %v", expectedSecond, Raw(second))
	}

	t.Run("struct", func(t *testing.T) {
		type Struct struct {
			X int32
		}
		verifyResults(t, Struct{
			X: 1,
		})
	})
	t.Run("pointer", func(t *testing.T) {
		i32 := int32(1)
		verifyResults(t, M{
			"x": &i32,
		})
	})
}

func TestNullBytes(t *testing.T) {
	t.Run("element keys", func(t *testing.T) {
		doc := D{{"a\x00", "foobar"}}
		res, err := Marshal(doc)
		want := errors.New("BSON element key cannot contain null bytes")
		assert.Equal(t, want, err, "expected Marshal error %v, got error %v with result %q", want, err, Raw(res))
	})

	t.Run("regex values", func(t *testing.T) {
		wantErr := errors.New("BSON regex values cannot contain null bytes")

		testCases := []struct {
			name    string
			pattern string
			options string
		}{
			{"null bytes in pattern", "a\x00", "i"},
			{"null bytes in options", "pattern", "i\x00"},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				regex := primitive.Regex{
					Pattern: tc.pattern,
					Options: tc.options,
				}
				res, err := Marshal(D{{"foo", regex}})
				assert.Equal(t, wantErr, err, "expected Marshal error %v, got error %v with result %q", wantErr, err, Raw(res))
			})
		}
	})

	t.Run("sub document field name", func(t *testing.T) {
		doc := D{{"foo", D{{"foobar", D{{"a\x00", "foobar"}}}}}}
		res, err := Marshal(doc)
		wantErr := errors.New("BSON element key cannot contain null bytes")
		assert.Equal(t, wantErr, err, "expected Marshal error %v, got error %v with result %q", wantErr, err, Raw(res))
	})
}

func TestMarshalExtJSONIndent(t *testing.T) {
	type indentTestCase struct {
		name            string
		val             interface{}
		expectedExtJSON string
	}

	// expectedExtJSON must be written as below because single-quoted
	// literal strings capture undesired code formatting tabs
	testCases := []indentTestCase{
		{
			"empty val",
			struct{}{},
			`{}`,
		},
		{
			"embedded struct",
			struct {
				Embedded interface{} `json:"embedded"`
				Foo      string      `json:"foo"`
			}{
				Embedded: struct {
					Name string `json:"name"`
					Word string `json:"word"`
				}{
					Name: "test",
					Word: "word",
				},
				Foo: "bar",
			},
			"{\n\t\"embedded\": {\n\t\t\"name\": \"test\",\n\t\t\"word\": \"word\"\n\t},\n\t\"foo\": \"bar\"\n}",
		},
		{
			"date struct",
			struct {
				Foo  string    `json:"foo"`
				Date time.Time `json:"date"`
			}{
				Foo:  "bar",
				Date: time.Date(2000, time.January, 1, 12, 0, 0, 0, time.UTC),
			},
			"{\n\t\"foo\": \"bar\",\n\t\"date\": {\n\t\t\"$date\": {\n\t\t\t\"$numberLong\": \"946728000000\"\n\t\t}\n\t}\n}",
		},
		{
			"float struct",
			struct {
				Foo   string  `json:"foo"`
				Float float32 `json:"float"`
			}{
				Foo:   "bar",
				Float: 3.14,
			},
			"{\n\t\"foo\": \"bar\",\n\t\"float\": {\n\t\t\"$numberDouble\": \"3.140000104904175\"\n\t}\n}",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			extJSONBytes, err := MarshalExtJSONIndent(tc.val, true, false, "", "\t")
			assert.Nil(t, err, "Marshal indent error: %v", err)

			expectedExtJSONBytes := []byte(tc.expectedExtJSON)

			assert.Equal(t, expectedExtJSONBytes, extJSONBytes, "expected:\n%s\ngot:\n%s", expectedExtJSONBytes, extJSONBytes)
		})
	}
}
