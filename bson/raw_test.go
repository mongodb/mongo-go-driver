// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

func ExampleRaw_Validate() {
	rdr := make(Raw, 500)
	rdr[250], rdr[251], rdr[252], rdr[253], rdr[254] = '\x05', '\x00', '\x00', '\x00', '\x00'
	err := rdr[250:].Validate()
	fmt.Println(err)

	// Output: <nil>
}

func BenchmarkRawValidate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rdr := make(Raw, 500)
		rdr[250], rdr[251], rdr[252], rdr[253], rdr[254] = '\x05', '\x00', '\x00', '\x00', '\x00'
		_ = rdr[250:].Validate()
	}

}

func TestRaw(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		t.Run("TooShort", func(t *testing.T) {
			want := bsoncore.NewInsufficientBytesError(nil, nil)
			got := Raw{'\x00', '\x00'}.Validate()
			if !assert.CompareErrors(got, want) {
				t.Errorf("Did not get expected error. got %v; want %v", got, want)
			}
		})
		t.Run("InvalidLength", func(t *testing.T) {
			want := bsoncore.ValidationError("document length exceeds available bytes. length=200 remainingBytes=5")
			r := make(Raw, 5)
			binary.LittleEndian.PutUint32(r[0:4], 200)
			got := r.Validate()
			if !errors.Is(got, want) {
				t.Errorf("Did not get expected error. got %v; want %v", got, want)
			}
		})
		t.Run("keyLength-error", func(t *testing.T) {
			want := bsoncore.ErrMissingNull
			r := make(Raw, 8)
			binary.LittleEndian.PutUint32(r[0:4], 8)
			r[4], r[5], r[6], r[7] = '\x02', 'f', 'o', 'o'
			got := r.Validate()
			if !errors.Is(got, want) {
				t.Errorf("Did not get expected error. got %v; want %v", got, want)
			}
		})
		t.Run("Missing-Null-Terminator", func(t *testing.T) {
			want := bsoncore.ErrMissingNull
			r := make(Raw, 9)
			binary.LittleEndian.PutUint32(r[0:4], 9)
			r[4], r[5], r[6], r[7], r[8] = '\x0A', 'f', 'o', 'o', '\x00'
			got := r.Validate()
			if !errors.Is(got, want) {
				t.Errorf("Did not get expected error. got %v; want %v", got, want)
			}
		})
		t.Run("validateValue-error", func(t *testing.T) {
			want := bsoncore.ErrMissingNull
			r := make(Raw, 11)
			binary.LittleEndian.PutUint32(r[0:4], 11)
			r[4], r[5], r[6], r[7], r[8], r[9], r[10] = '\x01', 'f', 'o', 'o', '\x00', '\x01', '\x02'
			got := r.Validate()
			if !assert.CompareErrors(got, want) {
				t.Errorf("Did not get expected error. got %v; want %v", got, want)
			}
		})
		testCases := []struct {
			name string
			r    Raw
			err  error
		}{
			{"null", Raw{'\x08', '\x00', '\x00', '\x00', '\x0A', 'x', '\x00', '\x00'}, nil},
			{"subdocument",
				Raw{
					'\x15', '\x00', '\x00', '\x00',
					'\x03',
					'f', 'o', 'o', '\x00',
					'\x0B', '\x00', '\x00', '\x00', '\x0A', 'a', '\x00',
					'\x0A', 'b', '\x00', '\x00', '\x00',
				},
				nil,
			},
			{"array",
				Raw{
					'\x15', '\x00', '\x00', '\x00',
					'\x04',
					'f', 'o', 'o', '\x00',
					'\x0B', '\x00', '\x00', '\x00', '\x0A', '1', '\x00',
					'\x0A', '2', '\x00', '\x00', '\x00',
				},
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.r.Validate()
				if !errors.Is(err, tc.err) {
					t.Errorf("Returned error does not match. got %v; want %v", err, tc.err)
				}
			})
		}
	})
	t.Run("Lookup", func(t *testing.T) {
		t.Run("empty-key", func(t *testing.T) {
			rdr := Raw{'\x05', '\x00', '\x00', '\x00', '\x00'}
			_, err := rdr.LookupErr()
			if !errors.Is(err, bsoncore.ErrEmptyKey) {
				t.Errorf("Empty key lookup did not return expected result. got %v; want %v", err, bsoncore.ErrEmptyKey)
			}
		})
		t.Run("corrupted-subdocument", func(t *testing.T) {
			rdr := Raw{
				'\x0D', '\x00', '\x00', '\x00',
				'\x03', 'x', '\x00',
				'\x06', '\x00', '\x00', '\x00',
				'\x01',
				'\x00',
				'\x00',
			}
			_, err := rdr.LookupErr("x", "y")
			want := bsoncore.NewInsufficientBytesError(nil, nil)
			if !assert.CompareErrors(err, want) {
				t.Errorf("Empty key lookup did not return expected result. got %v; want %v", err, want)
			}
		})
		t.Run("corrupted-array", func(t *testing.T) {
			rdr := Raw{
				'\x0D', '\x00', '\x00', '\x00',
				'\x04', 'x', '\x00',
				'\x06', '\x00', '\x00', '\x00',
				'\x01',
				'\x00',
				'\x00',
			}
			_, err := rdr.LookupErr("x", "y")
			want := bsoncore.NewInsufficientBytesError(nil, nil)
			if !assert.CompareErrors(err, want) {
				t.Errorf("Empty key lookup did not return expected result. got %v; want %v", err, want)
			}
		})
		t.Run("invalid-traversal", func(t *testing.T) {
			rdr := Raw{'\x08', '\x00', '\x00', '\x00', '\x0A', 'x', '\x00', '\x00'}
			_, err := rdr.LookupErr("x", "y")
			want := bsoncore.InvalidDepthTraversalError{Key: "x", Type: bsoncore.TypeNull}
			if !assert.CompareErrors(err, want) {
				t.Errorf("Empty key lookup did not return expected result. got %v; want %v", err, want)
			}
		})
		testCases := []struct {
			name string
			r    Raw
			key  []string
			want RawValue
			err  error
		}{
			{"first",
				Raw{
					'\x08', '\x00', '\x00', '\x00', '\x0A', 'x', '\x00', '\x00',
				},
				[]string{"x"},
				RawValue{Type: TypeNull}, nil,
			},
			{"first-second",
				Raw{
					'\x15', '\x00', '\x00', '\x00',
					'\x03',
					'f', 'o', 'o', '\x00',
					'\x0B', '\x00', '\x00', '\x00', '\x0A', 'a', '\x00',
					'\x0A', 'b', '\x00', '\x00', '\x00',
				},
				[]string{"foo", "b"},
				RawValue{Type: TypeNull}, nil,
			},
			{"first-second-array",
				Raw{
					'\x15', '\x00', '\x00', '\x00',
					'\x04',
					'f', 'o', 'o', '\x00',
					'\x0B', '\x00', '\x00', '\x00', '\x0A', '1', '\x00',
					'\x0A', '2', '\x00', '\x00', '\x00',
				},
				[]string{"foo", "2"},
				RawValue{Type: TypeNull}, nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := tc.r.LookupErr(tc.key...)
				if !errors.Is(err, tc.err) {
					t.Errorf("Returned error does not match. got %v; want %v", err, tc.err)
				}
				if !cmp.Equal(got, tc.want) {
					t.Errorf("Returned element does not match expected element. got %v; want %v", got, tc.want)
				}
			})
		}
	})
	t.Run("ElementAt", func(t *testing.T) {
		t.Run("Out of bounds", func(t *testing.T) {
			rdr := Raw{0xe, 0x0, 0x0, 0x0, 0xa, 0x78, 0x0, 0xa, 0x79, 0x0, 0xa, 0x7a, 0x0, 0x0}
			_, err := rdr.IndexErr(3)
			if !errors.Is(err, bsoncore.ErrOutOfBounds) {
				t.Errorf("Out of bounds should be returned when accessing element beyond end of document. got %v; want %v", err, bsoncore.ErrOutOfBounds)
			}
		})
		t.Run("Validation Error", func(t *testing.T) {
			rdr := Raw{0x07, 0x00, 0x00, 0x00, 0x00}
			_, err := rdr.IndexErr(1)
			want := bsoncore.NewInsufficientBytesError(nil, nil)
			if !assert.CompareErrors(err, want) {
				t.Errorf("Did not receive expected error. got %v; want %v", err, want)
			}
		})
		testCases := []struct {
			name  string
			rdr   Raw
			index uint
			want  RawElement
		}{
			{"first",
				Raw{0xe, 0x0, 0x0, 0x0, 0xa, 0x78, 0x0, 0xa, 0x79, 0x0, 0xa, 0x7a, 0x0, 0x0},
				0, bsoncore.AppendNullElement(nil, "x")},
			{"second",
				Raw{0xe, 0x0, 0x0, 0x0, 0xa, 0x78, 0x0, 0xa, 0x79, 0x0, 0xa, 0x7a, 0x0, 0x0},
				1, bsoncore.AppendNullElement(nil, "y")},
			{"third",
				Raw{0xe, 0x0, 0x0, 0x0, 0xa, 0x78, 0x0, 0xa, 0x79, 0x0, 0xa, 0x7a, 0x0, 0x0},
				2, bsoncore.AppendNullElement(nil, "z")},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := tc.rdr.IndexErr(tc.index)
				if err != nil {
					t.Errorf("Unexpected error from ElementAt: %s", err)
				}
				if diff := cmp.Diff(got, tc.want); diff != "" {
					t.Errorf("Documents differ: (-got +want)\n%s", diff)
				}
			})
		}
	})
	t.Run("ReadDocument", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			name       string
			ioReader   io.Reader
			bsonReader Raw
			err        error
		}{
			{
				"nil reader",
				nil,
				nil,
				ErrNilReader,
			},
			{
				"premature end of reader",
				bytes.NewBuffer([]byte{}),
				nil,
				io.EOF,
			},
			{
				"empty document",
				bytes.NewBuffer([]byte{5, 0, 0, 0, 0}),
				[]byte{5, 0, 0, 0, 0},
				nil,
			},
			{
				"non-empty document",
				bytes.NewBuffer([]byte{
					// length
					0x17, 0x0, 0x0, 0x0,

					// type - string
					0x2,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - string "bar"
					0x62, 0x61, 0x72, 0x0,

					// type - null
					0xa,
					// key - "baz"
					0x62, 0x61, 0x7a, 0x0,

					// null terminator
					0x0,
				}),
				[]byte{
					// length
					0x17, 0x0, 0x0, 0x0,

					// type - string
					0x2,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - string "bar"
					0x62, 0x61, 0x72, 0x0,

					// type - null
					0xa,
					// key - "baz"
					0x62, 0x61, 0x7a, 0x0,

					// null terminator
					0x0,
				},
				nil,
			},
		}

		for _, tc := range testCases {
			tc := tc // Capture range variable.
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				reader, err := ReadDocument(tc.ioReader)
				require.Equal(t, err, tc.err)
				require.True(t, bytes.Equal(tc.bsonReader, reader))
			})
		}
	})
}

func BenchmarkRawString(b *testing.B) {
	// Create 1KiB and 128B strings to exercise the string-heavy call paths in
	// the "Raw.String" method.
	var buf strings.Builder
	for i := 0; i < 16000000; i++ {
		buf.WriteString("abcdefgh")
	}
	str1k := buf.String()
	str128 := str1k[:128]

	cases := []struct {
		description string
		value       interface{}
	}{
		{
			description: "string",
			value:       D{{Key: "key", Value: str128}},
		},
		{
			description: "integer",
			value:       D{{Key: "key", Value: int64(1234567890)}},
		},
		{
			description: "float",
			value:       D{{Key: "key", Value: float64(1234567890.123456789)}},
		},
		{
			description: "nested document",
			value: D{{
				Key: "key",
				Value: D{{
					Key: "key",
					Value: D{{
						Key:   "key",
						Value: str128,
					}},
				}},
			}},
		},
		{
			description: "array of strings",
			value: D{{
				Key:   "key",
				Value: []string{str128, str128, str128, str128},
			}},
		},
		{
			description: "mixed struct",
			value: struct {
				Key1 struct {
					Nested string
				}
				Key2 string
				Key3 []string
				Key4 float64
			}{
				Key1: struct{ Nested string }{Nested: str1k},
				Key2: str1k,
				Key3: []string{str1k, str1k, str1k, str1k},
				Key4: 1234567890.123456789,
			},
		},
		// Very voluminous document with hundreds of thousands of keys
		{
			description: "very_voluminous_document",
			value:       createVoluminousDocument(100000),
		},
		// Document containing large strings and values
		{
			description: "large_strings_and_values",
			value:       createLargeStringsDocument(10),
		},
		// Document with massive arrays that are large
		{
			description: "massive_arrays",
			value:       createMassiveArraysDocument(100000),
		},
	}

	for _, tc := range cases {
		b.Run(tc.description, func(b *testing.B) {
			bs, err := Marshal(tc.value)
			require.NoError(b, err)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = Raw(bs).String()
			}
		})

		b.Run(tc.description+"_StringN", func(b *testing.B) {
			bs, err := Marshal(tc.value)
			require.NoError(b, err)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bsoncore.Document(bs).StringN(1024) // Assuming you want to limit to 1024 bytes for this benchmark
			}
		})
	}
}

func TestComplexDocuments_StringN(t *testing.T) {
	testCases := []struct {
		description string
		n           int
		doc         any
	}{
		{"n>0, massive array documents", 1000, createMassiveArraysDocument(1000)},
		{"n>0, voluminous document with unique values", 1000, createUniqueVoluminousDocument(t, 1000)},
		{"n>0, large single document", 1000, createLargeSingleDoc(t)},
		{"n>0, voluminous document with arrays containing documents", 1000, createVoluminousArrayDocuments(t, 1000)},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			bson, _ := Marshal(tc.doc)
			bsonDoc := bsoncore.Document(bson)

			got := bsonDoc.StringN(tc.n)
			assert.Equal(t, tc.n, len(got))
		})
	}
}

// createVoluminousDocument generates a document with a specified number of keys, simulating a very large document in terms of the number of keys.
func createVoluminousDocument(numKeys int) D {
	d := make(D, numKeys)
	for i := 0; i < numKeys; i++ {
		d = append(d, E{Key: fmt.Sprintf("key%d", i), Value: "value"})
	}
	return d
}

// createLargeStringsDocument generates a document with large string values, simulating a document with very large data values.
func createLargeStringsDocument(sizeMB int) D {
	largeString := strings.Repeat("a", sizeMB*1024*1024)
	return D{
		{Key: "largeString1", Value: largeString},
		{Key: "largeString2", Value: largeString},
		{Key: "largeString3", Value: largeString},
		{Key: "largeString4", Value: largeString},
	}
}

// createMassiveArraysDocument generates a document with massive arrays, simulating a document that contains large arrays of data.
func createMassiveArraysDocument(arraySize int) D {
	massiveArray := make([]string, arraySize)
	for i := 0; i < arraySize; i++ {
		massiveArray[i] = "value"
	}

	return D{
		{Key: "massiveArray1", Value: massiveArray},
		{Key: "massiveArray2", Value: massiveArray},
		{Key: "massiveArray3", Value: massiveArray},
		{Key: "massiveArray4", Value: massiveArray},
	}
}

// createUniqueVoluminousDocument creates a BSON document with multiple key value pairs and unique value types.
func createUniqueVoluminousDocument(t *testing.T, size int) bsoncore.Document {
	t.Helper()

	docs := make(D, size)

	for i := 0; i < size; i++ {
		docs = append(docs, E{
			Key: "x", Value: NewObjectID(),
		})
		docs = append(docs, E{
			Key: "z", Value: "y",
		})
	}

	bsonData, err := Marshal(docs)
	assert.NoError(t, err)

	return bsoncore.Document(bsonData)
}

// createLargeSingleDoc creates a large single BSON document.
func createLargeSingleDoc(t *testing.T) bsoncore.Document {
	t.Helper()

	var b strings.Builder
	b.Grow(1048577)

	for i := 0; i < 1048577; i++ {
		b.WriteByte(0)
	}
	s := b.String()

	doc := D{
		{Key: "x", Value: s},
	}

	bsonData, err := Marshal(doc)
	assert.NoError(t, err)

	return bsoncore.Document(bsonData)
}

// createVoluminousArrayDocuments creates a volumninous BSON document with arrays containing documents.
func createVoluminousArrayDocuments(t *testing.T, size int) bsoncore.Document {
	t.Helper()

	docs := make(D, size)

	for i := 0; i < size; i++ {
		docs = append(docs, E{
			Key: "x", Value: NewObjectID(),
		})
		docs = append(docs, E{
			Key: "z", Value: A{D{{Key: "x", Value: "y"}}},
		})
	}

	bsonData, err := Marshal(docs)
	assert.NoError(t, err)

	return bsoncore.Document(bsonData)
}
