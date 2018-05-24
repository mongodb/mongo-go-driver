// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"encoding/json"
	"net/url"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/stretchr/testify/require"
)

func requireErrEqual(t *testing.T, err1 error, err2 error) {
	switch e := err1.(type) {
	case ErrTooSmall:
		require.True(t, e.Equals(err2))

		return
	}

	switch e := err2.(type) {
	case ErrTooSmall:
		require.True(t, e.Equals(e))

		return
	}

	require.Equal(t, err1, err2)
}

func TestDecoder(t *testing.T) {
	t.Run("byte slice", func(t *testing.T) {
		testCases := []struct {
			name     string
			reader   *bytes.Buffer
			expected []byte
			actual   []byte
			err      error
		}{
			{
				"nil",
				bytes.NewBuffer([]byte{0x5, 0x0, 0x0, 0x0, 0x0}),
				nil,
				nil,
				NewErrTooSmall(),
			},
			{
				"empty slice",
				bytes.NewBuffer([]byte{0x5, 0x0, 0x0, 0x0}),
				nil,
				[]byte{},
				NewErrTooSmall(),
			},
			{
				"too small",
				bytes.NewBuffer([]byte{
					0x5, 0x0, 0x0, 0x0, 0x0,
				}),
				nil,
				make([]byte, 0x4),
				NewErrTooSmall(),
			},
			{
				"empty doc",
				bytes.NewBuffer([]byte{
					0x5, 0x0, 0x0, 0x0, 0x0,
				}),
				[]byte{0x5, 0x0, 0x0, 0x0, 0x0},
				make([]byte, 0x5),
				nil,
			},
			{
				"non-empty doc",
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
				make([]byte, 0x17),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				d := NewDecoder(tc.reader)

				err := d.Decode(tc.actual)
				requireErrEqual(t, tc.err, err)
				if err != nil {
					return
				}

				require.True(t, bytes.Equal(tc.expected, tc.actual))
			})
		}
	})

	t.Run("Reader", func(t *testing.T) {
		testCases := []struct {
			name     string
			reader   *bytes.Buffer
			expected Reader
			actual   Reader
			err      error
		}{
			{
				"nil",
				bytes.NewBuffer([]byte{0x5, 0x0, 0x0, 0x0, 0x0}),
				nil,
				nil,
				NewErrTooSmall(),
			},
			{
				"empty slice",
				bytes.NewBuffer([]byte{0x5, 0x0, 0x0, 0x0}),
				nil,
				[]byte{},
				NewErrTooSmall(),
			},
			{
				"too small",
				bytes.NewBuffer([]byte{
					0x5, 0x0, 0x0, 0x0, 0x0,
				}),
				nil,
				make([]byte, 0x4),
				NewErrTooSmall(),
			},
			{
				"empty doc",
				bytes.NewBuffer([]byte{
					0x5, 0x0, 0x0, 0x0, 0x0,
				}),
				[]byte{0x5, 0x0, 0x0, 0x0, 0x0},
				make([]byte, 0x5),
				nil,
			},
			{
				"non-empty doc",
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
				make([]byte, 0x17),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				d := NewDecoder(tc.reader)

				err := d.Decode(tc.actual)
				requireErrEqual(t, tc.err, err)
				if err != nil {
					return
				}

				require.True(t, bytes.Equal(tc.expected, tc.actual))
			})
		}
	})

	t.Run("io.Writer", func(t *testing.T) {
		testCases := []struct {
			name     string
			reader   *bytes.Buffer
			expected *bytes.Buffer
			actual   *bytes.Buffer
			err      error
		}{
			{
				"empty doc",
				bytes.NewBuffer([]byte{
					0x5, 0x0, 0x0, 0x0, 0x0,
				}),
				bytes.NewBuffer([]byte{
					0x5, 0x0, 0x0, 0x0, 0x0,
				}),
				bytes.NewBuffer([]byte{}),
				nil,
			},
			{
				"non-empty doc",
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
				bytes.NewBuffer([]byte{}),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				d := NewDecoder(tc.reader)

				err := d.Decode(tc.actual)
				requireErrEqual(t, tc.err, err)
				if err != nil {
					return
				}

				require.Equal(t, tc.expected, tc.actual)
			})
		}
	})

	t.Run("Unmarshaler", func(t *testing.T) {
		testCases := []struct {
			name     string
			reader   *bytes.Buffer
			expected *Document
			actual   *Document
			err      error
		}{
			{
				"empty doc",
				bytes.NewBuffer([]byte{
					0x5, 0x0, 0x0, 0x0, 0x0,
				}),
				NewDocument(),
				NewDocument(),
				nil,
			},
			{
				"non-empty doc",
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
				NewDocument(
					EC.String("foo", "bar"),
					EC.Null("baz"),
				),
				NewDocument(),
				nil,
			},
			{
				"nested doc",
				bytes.NewBuffer([]byte{
					// length
					0x26, 0x0, 0x0, 0x0,

					// type - string
					0x2,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - string "bar"
					0x62, 0x61, 0x72, 0x0,

					// type - document
					0x3,
					// key - "baz"
					0x62, 0x61, 0x7a, 0x0,

					// -- begin subdocument --

					// length
					0xf, 0x0, 0x0, 0x0,

					// type - int32
					0x10,
					// key - "bang"
					0x62, 0x61, 0x6e, 0x67, 0x0,
					// value - int32(12)
					0xc, 0x0, 0x0, 0x0,

					// null terminator
					0x0,

					// -- end subdocument

					// null terminator
					0x0,
				}),
				NewDocument(
					EC.String("foo", "bar"),
					EC.SubDocumentFromElements("baz",
						EC.Int32("bang", 12),
					),
				),
				NewDocument(),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				d := NewDecoder(tc.reader)

				err := d.Decode(tc.actual)
				requireErrEqual(t, tc.err, err)
				if err != nil {
					return
				}

				require.True(t, documentComparer(tc.expected, tc.actual))
			})
		}
	})

	t.Run("map", func(t *testing.T) {
		testCases := []struct {
			name     string
			reader   *bytes.Buffer
			expected map[string]interface{}
			actual   map[string]interface{}
			err      error
		}{
			{
				"empty doc",
				bytes.NewBuffer([]byte{
					0x5, 0x0, 0x0, 0x0, 0x0,
				}),
				make(map[string]interface{}),
				make(map[string]interface{}),
				nil,
			},
			{
				"non-empty doc",
				bytes.NewBuffer([]byte{
					// length
					0x1b, 0x0, 0x0, 0x0,

					// type - string
					0x2,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - string "bar"
					0x62, 0x61, 0x72, 0x0,

					// type - int32
					0x10,
					// key - "baz"
					0x62, 0x61, 0x7a, 0x0,
					// value - int32(32)
					0x20, 0x0, 0x0, 0x0,

					// null terminator
					0x0,
				}),
				map[string]interface{}{
					"foo": "bar",
					"baz": int32(32),
				},
				make(map[string]interface{}),
				nil,
			},
			{
				"containing array",
				bytes.NewBuffer(
					[]byte{
						// length
						0x1a, 0x0, 0x0, 0x0,

						// type - array
						0x4,
						// key - "foo"
						0x66, 0x6f, 0x6f, 0x0,

						// ----- begin array -----

						// length
						0x10, 0x0, 0x0, 0x0,

						// type string
						0x2,
						// key - "0"
						0x30, 0x0,
						// value - string length
						0x4, 0x0, 0x0, 0x0,
						// value - string "bar"
						0x62, 0x61, 0x72, 0x0,

						// null terminator
						0x0,

						// ----- end array -----

						// null terminator
						0x0,
					},
				),
				map[string]interface{}{
					"foo": []interface{}{"bar"},
				},
				make(map[string]interface{}),
				nil,
			},
			{
				"nested doc",
				bytes.NewBuffer([]byte{
					// length
					0x26, 0x0, 0x0, 0x0,

					// type - string
					0x2,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - string "bar"
					0x62, 0x61, 0x72, 0x0,

					// type - document
					0x3,
					// key - "baz"
					0x62, 0x61, 0x7a, 0x0,

					// -- begin subdocument --

					// length
					0xf, 0x0, 0x0, 0x0,

					// type - int32
					0x10,
					// key - "bang"
					0x62, 0x61, 0x6e, 0x67, 0x0,
					// value - int32(12)
					0xc, 0x0, 0x0, 0x0,

					// null terminator
					0x0,

					// -- end subdocument

					// null terminator
					0x0,
				}),
				map[string]interface{}{
					"foo": "bar",
					"baz": map[string]interface{}{
						"bang": int32(12),
					},
				},
				make(map[string]interface{}),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				d := NewDecoder(tc.reader)

				err := d.Decode(tc.actual)
				requireErrEqual(t, tc.err, err)
				if err != nil {
					return
				}

				require.True(t, cmp.Equal(tc.expected, tc.actual))
			})
		}
	})

	t.Run("element slice", func(t *testing.T) {
		testCases := []struct {
			name     string
			reader   *bytes.Buffer
			expected []*Element
			actual   []*Element
			err      error
		}{
			{
				"empty doc",
				bytes.NewBuffer([]byte{
					0x5, 0x0, 0x0, 0x0, 0x0,
				}),
				[]*Element{},
				[]*Element{},
				nil,
			},
			{
				"non-empty doc",
				bytes.NewBuffer([]byte{
					// length
					0x1b, 0x0, 0x0, 0x0,

					// type - string
					0x2,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - string "bar"
					0x62, 0x61, 0x72, 0x0,

					// type - int32
					0x10,
					// key - "baz"
					0x62, 0x61, 0x7a, 0x0,
					// value - int32(32)
					0x20, 0x0, 0x0, 0x0,

					// null terminator
					0x0,
				}),
				[]*Element{
					EC.String("foo", "bar"),
					EC.Int32("baz", 32),
				},
				make([]*Element, 2),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				d := NewDecoder(tc.reader)

				err := d.Decode(tc.actual)
				requireErrEqual(t, tc.err, err)
				if err != nil {
					return
				}

				elementSliceEqual(t, tc.expected, tc.actual)
			})
		}
	})

	t.Run("struct", func(t *testing.T) {
		stringValue := "bar"
		int32Value := int32(32)
		int32Value12 := int32(12)
		testCases := []struct {
			name     string
			reader   *bytes.Buffer
			expected interface{}
			actual   interface{}
			err      error
		}{
			{
				"empty doc",
				bytes.NewBuffer([]byte{
					0x5, 0x0, 0x0, 0x0, 0x0,
				}),
				&struct{}{},
				&struct{}{},
				nil,
			},
			{
				"non-empty doc",
				bytes.NewBuffer([]byte{
					// length
					0x25, 0x0, 0x0, 0x0,

					// type - string
					0x2,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - string "bar"
					0x62, 0x61, 0x72, 0x0,

					// type - int32
					0x10,
					// key - "baz"
					0x62, 0x61, 0x7a, 0x0,
					// value - int32(32)
					0x20, 0x0, 0x0, 0x0,

					// type - regex
					0xb,
					// key - "r"
					0x72, 0x0,
					// value - pattern("WoRd")
					0x57, 0x6f, 0x52, 0x64, 0x0,
					// value - options("i")
					0x69, 0x0,

					// null terminator
					0x0,
				}),
				&struct {
					Foo string
					Baz int32
					R   Regex
				}{
					"bar",
					32,
					Regex{Pattern: "WoRd", Options: "i"},
				},
				&struct {
					Foo string
					Baz int32
					R   Regex
				}{},
				nil,
			},
			{
				"non-empty doc pointers",
				bytes.NewBuffer([]byte{
					// length
					0x25, 0x0, 0x0, 0x0,

					// type - string
					0x2,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - string "bar"
					0x62, 0x61, 0x72, 0x0,

					// type - int32
					0x10,
					// key - "baz"
					0x62, 0x61, 0x7a, 0x0,
					// value - int32(32)
					0x20, 0x0, 0x0, 0x0,

					// type - regex
					0xb,
					// key - "r"
					0x72, 0x0,
					// value - pattern("WoRd")
					0x57, 0x6f, 0x52, 0x64, 0x0,
					// value - options("i")
					0x69, 0x0,

					// null terminator
					0x0,
				}),
				&struct {
					Foo *string
					Baz *int32
					R   *Regex
				}{
					&stringValue,
					&int32Value,
					&Regex{Pattern: "WoRd", Options: "i"},
				},
				&struct {
					Foo *string
					Baz *int32
					R   *Regex
				}{},
				nil,
			},
			{
				"empty interface field",
				bytes.NewBuffer([]byte{
					// length
					0xe, 0x0, 0x0, 0x0,

					// type - int32
					0x10,
					// key - "baz"
					0x62, 0x61, 0x7a, 0x0,
					// value - int32(32)
					0x20, 0x0, 0x0, 0x0,

					// null terminator
					0x0,
				}),
				&struct {
					Baz interface{}
				}{
					int32(32),
				},
				&struct {
					Baz interface{}
				}{},
				nil,
			},
			{
				"containing array",
				bytes.NewBuffer(
					[]byte{
						// length
						0x1a, 0x0, 0x0, 0x0,

						// type - array
						0x4,
						// key - "foo"
						0x66, 0x6f, 0x6f, 0x0,

						// ----- begin array -----

						// length
						0x10, 0x0, 0x0, 0x0,

						// type string
						0x2,
						// key - "0"
						0x30, 0x0,
						// value - string length
						0x4, 0x0, 0x0, 0x0,
						// value - string "bar"
						0x62, 0x61, 0x72, 0x0,

						// null terminator
						0x0,

						// ----- end array -----

						// null terminator
						0x0,
					},
				),
				&struct{ Foo interface{} }{[]interface{}{"bar"}},
				&struct{ Foo interface{} }{},
				nil,
			},
			{
				"nested doc",
				bytes.NewBuffer([]byte{
					// length
					0x26, 0x0, 0x0, 0x0,

					// type - string
					0x2,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - string "bar"
					0x62, 0x61, 0x72, 0x0,

					// type - document
					0x3,
					// key - "baz"
					0x62, 0x61, 0x7a, 0x0,

					// -- begin subdocument --

					// length
					0xf, 0x0, 0x0, 0x0,

					// type - int32
					0x10,
					// key - "bang"
					0x62, 0x61, 0x6e, 0x67, 0x0,
					// value - int32(12)
					0xc, 0x0, 0x0, 0x0,

					// null terminator
					0x0,

					// -- end subdocument

					// null terminator
					0x0,
				}),
				&struct {
					Foo string
					Baz struct {
						Bang int32
					}
				}{
					"bar",
					struct{ Bang int32 }{12},
				},
				&struct {
					Foo string
					Baz struct {
						Bang int32
					}
				}{},
				nil,
			},
			{
				"nested doc pointer",
				bytes.NewBuffer([]byte{
					// length
					0x26, 0x0, 0x0, 0x0,

					// type - string
					0x2,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - string "bar"
					0x62, 0x61, 0x72, 0x0,

					// type - document
					0x3,
					// key - "baz"
					0x62, 0x61, 0x7a, 0x0,

					// -- begin subdocument --

					// length
					0xf, 0x0, 0x0, 0x0,

					// type - int32
					0x10,
					// key - "bang"
					0x62, 0x61, 0x6e, 0x67, 0x0,
					// value - int32(12)
					0xc, 0x0, 0x0, 0x0,

					// null terminator
					0x0,

					// -- end subdocument

					// null terminator
					0x0,
				}),
				&struct {
					Foo *string
					Baz *struct {
						Bang *int32
					}
				}{
					&stringValue,
					&struct{ Bang *int32 }{&int32Value12},
				},
				&struct {
					Foo *string
					Baz *struct {
						Bang *int32
					}
				}{},
				nil,
			},
			{
				"struct tags",
				bytes.NewBuffer([]byte{
					// length
					0x1b, 0x0, 0x0, 0x0,

					// type - string
					0x2,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - string "bar"
					0x62, 0x61, 0x72, 0x0,

					// type - int32
					0x10,
					// key - "baz"
					0x62, 0x61, 0x7a, 0x0,
					// value - int32(32)
					0x20, 0x0, 0x0, 0x0,

					// null terminator
					0x0,
				}),
				&struct {
					A string `bson:"foo"`
					B int32  `bson:"baz,omitempty"`
				}{
					"bar",
					32,
				},
				&struct {
					A string `bson:"foo"`
					B int32  `bson:"baz,omitempty"`
				}{},
				nil,
			},
			{
				"struct with pointers tags",
				bytes.NewBuffer([]byte{
					// length
					0x1b, 0x0, 0x0, 0x0,

					// type - string
					0x2,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - string "bar"
					0x62, 0x61, 0x72, 0x0,

					// type - int32
					0x10,
					// key - "baz"
					0x62, 0x61, 0x7a, 0x0,
					// value - int32(32)
					0x20, 0x0, 0x0, 0x0,

					// null terminator
					0x0,
				}),
				&struct {
					A *string `bson:"foo"`
					B *int32  `bson:"baz,omitempty"`
				}{
					&stringValue,
					&int32Value,
				},
				&struct {
					A *string `bson:"foo"`
					B *int32  `bson:"baz,omitempty"`
				}{},
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				d := NewDecoder(tc.reader)

				err := d.Decode(tc.actual)
				requireErrEqual(t, tc.err, err)
				if err != nil {
					return
				}

				require.True(t, cmp.Equal(tc.expected, tc.actual))
			})
		}
	})

	t.Run("numbers", func(t *testing.T) {
		t.Run("decode int32", func(t *testing.T) {
			uint8Val := uint8(1)
			uint16Val := uint16(2)
			uint32Val := uint32(3)
			uint64Val := uint64(4)
			uintVal := uint(5)
			int8Val := int8(6)
			int16Val := int16(7)
			int32Val := int32(8)
			int64Val := int64(9)
			intVal := int(10)
			float32Val := float32(11.0)
			float64Val := float64(12.0)
			testCases := []struct {
				name     string
				reader   *bytes.Buffer
				expected interface{}
				actual   interface{}
				err      error
			}{
				{
					"negative into uint8",
					bytes.NewBuffer([]byte{
						// length
						0xe, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(-27)
						0xe5, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz uint8
					}{
						0,
					},
					&struct {
						Baz uint8
					}{},
					nil,
				},
				{
					"negative into uint8 pointer",
					bytes.NewBuffer([]byte{
						// length
						0xe, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(-27)
						0xe5, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz *uint8
					}{
						nil,
					},
					&struct {
						Baz *uint8
					}{},
					nil,
				},
				{
					"negative into uint16",
					bytes.NewBuffer([]byte{
						// length
						0xe, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(-27)
						0xe5, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz uint16
					}{
						0,
					},
					&struct {
						Baz uint16
					}{},
					nil,
				},
				{
					"negative into uint16 pointer",
					bytes.NewBuffer([]byte{
						// length
						0xe, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(-27)
						0xe5, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz *uint16
					}{
						nil,
					},
					&struct {
						Baz *uint16
					}{},
					nil,
				},
				{
					"negative into uint32",
					bytes.NewBuffer([]byte{
						// length
						0xe, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(-27)
						0xe5, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz uint32
					}{
						0,
					},
					&struct {
						Baz uint32
					}{},
					nil,
				},
				{
					"negative into uint32 pointer",
					bytes.NewBuffer([]byte{
						// length
						0xe, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(-27)
						0xe5, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz *uint32
					}{
						nil,
					},
					&struct {
						Baz *uint32
					}{},
					nil,
				},
				{
					"negative into uint64",
					bytes.NewBuffer([]byte{
						// length
						0xe, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(-27)
						0xe5, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz uint64
					}{
						0,
					},
					&struct {
						Baz uint64
					}{},
					nil,
				},
				{
					"negative into uint64 pointer",
					bytes.NewBuffer([]byte{
						// length
						0xe, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(-27)
						0xe5, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz *uint64
					}{
						nil,
					},
					&struct {
						Baz *uint64
					}{},
					nil,
				},
				{
					"negative into uint",
					bytes.NewBuffer([]byte{
						// length
						0xe, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(-27)
						0xe5, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz uint
					}{
						0,
					},
					&struct {
						Baz uint
					}{},
					nil,
				},
				{
					"negative into uint pointer",
					bytes.NewBuffer([]byte{
						// length
						0xe, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(-27)
						0xe5, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz *uint
					}{
						nil,
					},
					&struct {
						Baz *uint
					}{},
					nil,
				},
				{
					"too high for int8",
					bytes.NewBuffer([]byte{
						// length
						0xe, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(2^24)
						0x0, 0x0, 0x0, 0x1,

						// null terminator
						0x0,
					}),
					&struct {
						Baz int8
					}{
						0,
					},
					&struct {
						Baz int8
					}{},
					nil,
				},
				{
					"too high for int8 pointer",
					bytes.NewBuffer([]byte{
						// length
						0xe, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(2^24)
						0x0, 0x0, 0x0, 0x1,

						// null terminator
						0x0,
					}),
					&struct {
						Baz *int8
					}{
						nil,
					},
					&struct {
						Baz *int8
					}{},
					nil,
				},
				{
					"too high for int16",
					bytes.NewBuffer([]byte{
						// length
						0xe, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(2^24)
						0x0, 0x0, 0x0, 0x1,

						// null terminator
						0x0,
					}),
					&struct {
						Baz int16
					}{
						0,
					},
					&struct {
						Baz int16
					}{},
					nil,
				},
				{
					"too high for int16 pointer",
					bytes.NewBuffer([]byte{
						// length
						0xe, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(2^24)
						0x0, 0x0, 0x0, 0x1,

						// null terminator
						0x0,
					}),
					&struct {
						Baz *int16
					}{
						nil,
					},
					&struct {
						Baz *int16
					}{},
					nil,
				},
				{
					"success",
					bytes.NewBuffer([]byte{
						// length
						0x59, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "a"
						0x61, 0x0,
						// value - int32(1)
						0x1, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "b"
						0x62, 0x0,
						// value - int32(2)
						0x2, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "c"
						0x63, 0x0,
						// value - int32(3)
						0x3, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "d"
						0x64, 0x0,
						// value - int32(4)
						0x4, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "e"
						0x65, 0x0,
						// value - int32(5)
						0x5, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "f"
						0x66, 0x0,
						// value - int32(6)
						0x6, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "g"
						0x67, 0x0,
						// value - int32(7)
						0x7, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "h"
						0x68, 0x0,
						// value - int32(8)
						0x8, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "i"
						0x69, 0x0,
						// value - int32(9)
						0x9, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "j"
						0x6a, 0x0,
						// value - int32(10)
						0xa, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "k"
						0x6b, 0x0,
						// value - int32(11)
						0xb, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "l"
						0x6c, 0x0,
						// value - int32(12)
						0xc, 0x0, 0x0, 0x0,

						// null terminator
						0x0,
					}),
					&struct {
						A uint8
						B uint16
						C uint32
						D uint64
						E uint
						F int8
						G int16
						H int32
						I int64
						J int
						K float32
						L float64
					}{
						1,
						2,
						3,
						4,
						5,
						6,
						7,
						8,
						9,
						10,
						11,
						12,
					},
					&struct {
						A uint8
						B uint16
						C uint32
						D uint64
						E uint
						F int8
						G int16
						H int32
						I int64
						J int
						K float32
						L float64
					}{},
					nil,
				},
				{
					"success pointer",
					bytes.NewBuffer([]byte{
						// length
						0x59, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "a"
						0x61, 0x0,
						// value - int32(1)
						0x1, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "b"
						0x62, 0x0,
						// value - int32(2)
						0x2, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "c"
						0x63, 0x0,
						// value - int32(3)
						0x3, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "d"
						0x64, 0x0,
						// value - int32(4)
						0x4, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "e"
						0x65, 0x0,
						// value - int32(5)
						0x5, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "f"
						0x66, 0x0,
						// value - int32(6)
						0x6, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "g"
						0x67, 0x0,
						// value - int32(7)
						0x7, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "h"
						0x68, 0x0,
						// value - int32(8)
						0x8, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "i"
						0x69, 0x0,
						// value - int32(9)
						0x9, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "j"
						0x6a, 0x0,
						// value - int32(10)
						0xa, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "k"
						0x6b, 0x0,
						// value - int32(11)
						0xb, 0x0, 0x0, 0x0,

						// type - int32
						0x10,
						// key - "l"
						0x6c, 0x0,
						// value - int32(12)
						0xc, 0x0, 0x0, 0x0,

						// null terminator
						0x0,
					}),
					&struct {
						A *uint8
						B *uint16
						C *uint32
						D *uint64
						E *uint
						F *int8
						G *int16
						H *int32
						I *int64
						J *int
						K *float32
						L *float64
					}{
						&uint8Val,
						&uint16Val,
						&uint32Val,
						&uint64Val,
						&uintVal,
						&int8Val,
						&int16Val,
						&int32Val,
						&int64Val,
						&intVal,
						&float32Val,
						&float64Val,
					},
					&struct {
						A *uint8
						B *uint16
						C *uint32
						D *uint64
						E *uint
						F *int8
						G *int16
						H *int32
						I *int64
						J *int
						K *float32
						L *float64
					}{},
					nil,
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					d := NewDecoder(tc.reader)

					err := d.Decode(tc.actual)
					requireErrEqual(t, tc.err, err)
					if err != nil {
						return
					}

					require.True(t, reflect.DeepEqual(tc.expected, tc.actual))
				})
			}
		})

		t.Run("decode int64", func(t *testing.T) {
			uint8Val := uint8(1)
			uint16Val := uint16(2)
			uint32Val := uint32(3)
			uint64Val := uint64(4)
			uintVal := uint(5)
			int8Val := int8(6)
			int16Val := int16(7)
			int32Val := int32(8)
			int64Val := int64(9)
			intVal := int(10)
			float32Val := float32(11.0)
			float64Val := float64(12.0)
			testCases := []struct {
				name     string
				reader   *bytes.Buffer
				expected interface{}
				actual   interface{}
				err      error
			}{
				{
					"negative into uint8",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int64(-27)
						0xe5, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz uint8
					}{
						0,
					},
					&struct {
						Baz uint8
					}{},
					nil,
				},
				{
					"negative into uint8 pointer",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int64(-27)
						0xe5, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz *uint8
					}{
						nil,
					},
					&struct {
						Baz *uint8
					}{},
					nil,
				},
				{
					"negative into uint16",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int64(-27)
						0xe5, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz uint16
					}{
						0,
					},
					&struct {
						Baz uint16
					}{},
					nil,
				},
				{
					"negative into uint16 pointer",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int64(-27)
						0xe5, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz *uint16
					}{
						nil,
					},
					&struct {
						Baz *uint16
					}{},
					nil,
				},
				{
					"negative into uint32",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int64(-27)
						0xe5, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz uint32
					}{
						0,
					},
					&struct {
						Baz uint32
					}{},
					nil,
				},
				{
					"negative into uint32 pointer",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int64(-27)
						0xe5, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz *uint32
					}{
						nil,
					},
					&struct {
						Baz *uint32
					}{},
					nil,
				},
				{
					"negative into uint64",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int64(-27)
						0xe5, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz uint64
					}{
						0,
					},
					&struct {
						Baz uint64
					}{},
					nil,
				},
				{
					"negative into uint64 pointer",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int64(-27)
						0xe5, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz *uint64
					}{
						nil,
					},
					&struct {
						Baz *uint64
					}{},
					nil,
				},
				{
					"negative into uint",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int64(-27)
						0xe5, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz uint
					}{
						0,
					},
					&struct {
						Baz uint
					}{},
					nil,
				},
				{
					"negative into uint pointer",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int64(-27)
						0xe5, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,

						// null terminator
						0x0,
					}),
					&struct {
						Baz *uint
					}{
						nil,
					},
					&struct {
						Baz *uint
					}{},
					nil,
				},
				{
					"too high for int8",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int64(2^56)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,

						// null terminator
						0x0,
					}),
					&struct {
						Baz int8
					}{
						0,
					},
					&struct {
						Baz int8
					}{},
					nil,
				},
				{
					"too high for int8 pointer",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int64(2^56)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,

						// null terminator
						0x0,
					}),
					&struct {
						Baz *int8
					}{
						nil,
					},
					&struct {
						Baz *int8
					}{},
					nil,
				},
				{
					"too high for int16",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int64(2^56)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,

						// null terminator
						0x0,
					}),
					&struct {
						Baz int16
					}{
						0,
					},
					&struct {
						Baz int16
					}{},
					nil,
				},
				{
					"too high for int16 pointer",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int64(2^56)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,

						// null terminator
						0x0,
					}),
					&struct {
						Baz *int16
					}{
						nil,
					},
					&struct {
						Baz *int16
					}{},
					nil,
				},
				{
					"too high for int32",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int64(2^56)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,

						// null terminator
						0x0,
					}),
					&struct {
						Baz int32
					}{
						0,
					},
					&struct {
						Baz int32
					}{},
					nil,
				},
				{
					"too high for int32 pointer",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int64(2^56)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,

						// null terminator
						0x0,
					}),
					&struct {
						Baz *int32
					}{
						nil,
					},
					&struct {
						Baz *int32
					}{},
					nil,
				},
				{
					"success",
					bytes.NewBuffer([]byte{
						// length
						0x89, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "a"
						0x61, 0x0,
						// value - int64(1)
						0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "b"
						0x62, 0x0,
						// value - int64(2)
						0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "c"
						0x63, 0x0,
						// value - int64(3)
						0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "d"
						0x64, 0x0,
						// value - int64(4)
						0x4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "e"
						0x65, 0x0,
						// value - int64(5)
						0x5, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "f"
						0x66, 0x0,
						// value - int64(6)
						0x6, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "g"
						0x67, 0x0,
						// value - int64(7)
						0x7, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "h"
						0x68, 0x0,
						// value - int64(8)
						0x8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "i"
						0x69, 0x0,
						// value - int64(9)
						0x9, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "j"
						0x6a, 0x0,
						// value - int64(10)
						0xa, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "k"
						0x6b, 0x0,
						// value - int64(11)
						0xb, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "l"
						0x6c, 0x0,
						// value - int64(12)
						0xc, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// null terminator
						0x0,
					}),
					&struct {
						A uint8
						B uint16
						C uint32
						D uint64
						E uint
						F int8
						G int16
						H int32
						I int64
						J int
						K float32
						L float64
					}{
						1,
						2,
						3,
						4,
						5,
						6,
						7,
						8,
						9,
						10,
						11,
						12,
					},
					&struct {
						A uint8
						B uint16
						C uint32
						D uint64
						E uint
						F int8
						G int16
						H int32
						I int64
						J int
						K float32
						L float64
					}{},
					nil,
				},
				{
					"success pointers",
					bytes.NewBuffer([]byte{
						// length
						0x89, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "a"
						0x61, 0x0,
						// value - int64(1)
						0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "b"
						0x62, 0x0,
						// value - int64(2)
						0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "c"
						0x63, 0x0,
						// value - int64(3)
						0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "d"
						0x64, 0x0,
						// value - int64(4)
						0x4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "e"
						0x65, 0x0,
						// value - int64(5)
						0x5, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "f"
						0x66, 0x0,
						// value - int64(6)
						0x6, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "g"
						0x67, 0x0,
						// value - int64(7)
						0x7, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "h"
						0x68, 0x0,
						// value - int64(8)
						0x8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "i"
						0x69, 0x0,
						// value - int64(9)
						0x9, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "j"
						0x6a, 0x0,
						// value - int64(10)
						0xa, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "k"
						0x6b, 0x0,
						// value - int64(11)
						0xb, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// type - int64
						0x12,
						// key - "l"
						0x6c, 0x0,
						// value - int64(12)
						0xc, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,

						// null terminator
						0x0,
					}),
					&struct {
						A *uint8
						B *uint16
						C *uint32
						D *uint64
						E *uint
						F *int8
						G *int16
						H *int32
						I *int64
						J *int
						K *float32
						L *float64
					}{
						&uint8Val,
						&uint16Val,
						&uint32Val,
						&uint64Val,
						&uintVal,
						&int8Val,
						&int16Val,
						&int32Val,
						&int64Val,
						&intVal,
						&float32Val,
						&float64Val,
					},
					&struct {
						A *uint8
						B *uint16
						C *uint32
						D *uint64
						E *uint
						F *int8
						G *int16
						H *int32
						I *int64
						J *int
						K *float32
						L *float64
					}{},
					nil,
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					d := NewDecoder(tc.reader)

					err := d.Decode(tc.actual)
					requireErrEqual(t, tc.err, err)
					if err != nil {
						return
					}

					require.True(t, reflect.DeepEqual(tc.expected, tc.actual))
				})
			}
		})

		t.Run("decode double", func(t *testing.T) {
			/*var uint8Value uint8
			var uint16Value uint16*/
			dataToDecode := []byte{
				// length
				0x12, 0x0, 0x0, 0x0,

				// type - double
				0x1,
				// key - "baz"
				0x62, 0x61, 0x7a, 0x0,
				// value - double(0.5)
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xe0, 0x3f,

				// null terminator
				0x0,
			}

			uint8Val := uint8(1)
			uint16Val := uint16(2)
			uint32Val := uint32(3)
			uint64Val := uint64(4)
			uintVal := uint(5)
			int8Val := int8(6)
			int16Val := int16(7)
			int32Val := int32(8)
			int64Val := int64(9)
			intVal := int(10)
			float32Val := float32(11.0)
			float64Val := float64(12.0)

			testCases := []struct {
				name     string
				reader   *bytes.Buffer
				expected interface{}
				actual   interface{}
				err      error
			}{
				{
					"fraction into uint8",
					bytes.NewBuffer(dataToDecode),
					&struct {
						Baz uint8
					}{
						0,
					},
					&struct {
						Baz uint8
					}{},
					nil,
				},
				{
					"fraction into uint8 pointer",
					bytes.NewBuffer(dataToDecode),
					&struct {
						Baz *uint8
					}{
						nil,
					},
					&struct {
						Baz *uint8
					}{},
					nil,
				},
				{
					"fraction into uint16",
					bytes.NewBuffer(dataToDecode),
					&struct {
						Baz uint16
					}{
						0,
					},
					&struct {
						Baz uint16
					}{},
					nil,
				},
				{
					"fraction into uint16 pointer",
					bytes.NewBuffer(dataToDecode),
					&struct {
						Baz *uint16
					}{
						nil,
					},
					&struct {
						Baz *uint16
					}{},
					nil,
				},
				{
					"fraction into uint32",
					bytes.NewBuffer(dataToDecode),
					&struct {
						Baz uint32
					}{
						0,
					},
					&struct {
						Baz uint32
					}{},
					nil,
				},
				{
					"fraction into uint32 pointer",
					bytes.NewBuffer(dataToDecode),
					&struct {
						Baz *uint32
					}{
						nil,
					},
					&struct {
						Baz *uint32
					}{},
					nil,
				},
				{
					"fraction into uint64",
					bytes.NewBuffer(dataToDecode),
					&struct {
						Baz uint64
					}{
						0,
					},
					&struct {
						Baz uint64
					}{},
					nil,
				},
				{
					"fraction into uint64 pointer",
					bytes.NewBuffer(dataToDecode),
					&struct {
						Baz *uint64
					}{
						nil,
					},
					&struct {
						Baz *uint64
					}{},
					nil,
				},
				{
					"fraction into uint",
					bytes.NewBuffer(dataToDecode),
					&struct {
						Baz uint
					}{
						0,
					},
					&struct {
						Baz uint
					}{},
					nil,
				},
				{
					"fraction into uint pointer",
					bytes.NewBuffer(dataToDecode),
					&struct {
						Baz *uint
					}{
						nil,
					},
					&struct {
						Baz *uint
					}{},
					nil,
				},
				{
					"fraction into int32",
					bytes.NewBuffer(dataToDecode),
					&struct {
						Baz int32
					}{
						0,
					},
					&struct {
						Baz int32
					}{},
					nil,
				},
				{
					"fraction into int32 pointer",
					bytes.NewBuffer(dataToDecode),
					&struct {
						Baz *int32
					}{
						nil,
					},
					&struct {
						Baz *int32
					}{},
					nil,
				},
				{
					"fraction into int64",
					bytes.NewBuffer(dataToDecode),
					&struct {
						Baz int64
					}{
						0,
					},
					&struct {
						Baz int64
					}{},
					nil,
				},
				{
					"fraction into int64 pointer",
					bytes.NewBuffer(dataToDecode),
					&struct {
						Baz *int64
					}{
						nil,
					},
					&struct {
						Baz *int64
					}{},
					nil,
				},
				{
					"fraction into int",
					bytes.NewBuffer(dataToDecode),
					&struct {
						Baz int
					}{
						0,
					},
					&struct {
						Baz int
					}{},
					nil,
				},
				{
					"fraction into int pointer",
					bytes.NewBuffer(dataToDecode),
					&struct {
						Baz *int
					}{
						nil,
					},
					&struct {
						Baz *int
					}{},
					nil,
				},
				{
					"too precise for float32",
					bytes.NewBuffer([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - double
						0x1,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - double(3.00000000001)
						0xf6, 0x57, 0x0, 0x0, 0x0, 0x0, 0x8, 0x40,

						// null terminator
						0x0,
					}),
					&struct {
						Baz float32
					}{
						3,
					},
					&struct {
						Baz float32
					}{},
					nil,
				},
				{
					"success",
					bytes.NewBuffer([]byte{
						// length
						0x89, 0x0, 0x0, 0x0,

						// type - double
						0x1,
						// key - "a"
						0x61, 0x0,
						// value - double(1.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f,

						// type - double
						0x1,
						// key - "b"
						0x62, 0x0,
						// value - double(2.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x40,

						// type - double
						0x1,
						// key - "c"
						0x63, 0x0,
						// value - double(3.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x8, 0x40,

						// type - double
						0x1,
						// key - "d"
						0x64, 0x0,
						// value - double(4.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x10, 0x40,

						// type - double
						0x1,
						// key - "e"
						0x65, 0x0,
						// value - double(5.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x14, 0x40,

						// type - double
						0x1,
						// key - "f"
						0x66, 0x0,
						// value - double(6.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x18, 0x40,

						// type - double
						0x1,
						// key - "g"
						0x67, 0x0,
						// value - double(7.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1c, 0x40,

						// type - double
						0x1,
						// key - "h"
						0x68, 0x0,
						// value - double(8.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x20, 0x40,

						// type - double
						0x1,
						// key - "i"
						0x69, 0x0,
						// value - double(9.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x22, 0x40,

						// type - double
						0x1,
						// key - "j"
						0x6a, 0x0,
						// value - double(10.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x24, 0x40,

						// type - double
						0x1,
						// key - "k"
						0x6b, 0x0,
						// value - double(11.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x26, 0x40,

						// type - double
						0x1,
						// key - "j"
						0x6c, 0x0,
						// value - double(12.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x28, 0x40,

						// null terminator
						0x0,
					}),
					&struct {
						A uint8
						B uint16
						C uint32
						D uint64
						E uint
						F int8
						G int16
						H int32
						I int64
						J int
						K float32
						L float64
					}{
						1.0,
						2.0,
						3.0,
						4.0,
						5.0,
						6.0,
						7.0,
						8.0,
						9.0,
						10.0,
						11.0,
						12.0,
					},
					&struct {
						A uint8
						B uint16
						C uint32
						D uint64
						E uint
						F int8
						G int16
						H int32
						I int64
						J int
						K float32
						L float64
					}{},
					nil,
				},
				{
					"success pointers",
					bytes.NewBuffer([]byte{
						// length
						0x89, 0x0, 0x0, 0x0,

						// type - double
						0x1,
						// key - "a"
						0x61, 0x0,
						// value - double(1.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f,

						// type - double
						0x1,
						// key - "b"
						0x62, 0x0,
						// value - double(2.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x40,

						// type - double
						0x1,
						// key - "c"
						0x63, 0x0,
						// value - double(3.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x8, 0x40,

						// type - double
						0x1,
						// key - "d"
						0x64, 0x0,
						// value - double(4.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x10, 0x40,

						// type - double
						0x1,
						// key - "e"
						0x65, 0x0,
						// value - double(5.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x14, 0x40,

						// type - double
						0x1,
						// key - "f"
						0x66, 0x0,
						// value - double(6.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x18, 0x40,

						// type - double
						0x1,
						// key - "g"
						0x67, 0x0,
						// value - double(7.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1c, 0x40,

						// type - double
						0x1,
						// key - "h"
						0x68, 0x0,
						// value - double(8.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x20, 0x40,

						// type - double
						0x1,
						// key - "i"
						0x69, 0x0,
						// value - double(9.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x22, 0x40,

						// type - double
						0x1,
						// key - "j"
						0x6a, 0x0,
						// value - double(10.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x24, 0x40,

						// type - double
						0x1,
						// key - "k"
						0x6b, 0x0,
						// value - double(11.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x26, 0x40,

						// type - double
						0x1,
						// key - "j"
						0x6c, 0x0,
						// value - double(12.0)
						0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x28, 0x40,

						// null terminator
						0x0,
					}),
					&struct {
						A *uint8
						B *uint16
						C *uint32
						D *uint64
						E *uint
						F *int8
						G *int16
						H *int32
						I *int64
						J *int
						K *float32
						L *float64
					}{
						&uint8Val,
						&uint16Val,
						&uint32Val,
						&uint64Val,
						&uintVal,
						&int8Val,
						&int16Val,
						&int32Val,
						&int64Val,
						&intVal,
						&float32Val,
						&float64Val,
					},
					&struct {
						A *uint8
						B *uint16
						C *uint32
						D *uint64
						E *uint
						F *int8
						G *int16
						H *int32
						I *int64
						J *int
						K *float32
						L *float64
					}{},
					nil,
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					d := NewDecoder(tc.reader)

					err := d.Decode(tc.actual)
					requireErrEqual(t, tc.err, err)
					if err != nil {
						return
					}

					require.True(t, reflect.DeepEqual(tc.expected, tc.actual))
				})
			}
		})
		t.Run("decimal128", func(t *testing.T) {
			decimal128, err := decimal.ParseDecimal128("1.5e10")
			if err != nil {
				t.Errorf("Error parsing decimal128: %v", err)
				t.FailNow()
			}
			testCases := []struct {
				name     string
				reader   []byte
				expected interface{}
				actual   interface{}
				err      error
			}{
				{
					"decimal128",
					docToBytes(NewDocument(EC.Decimal128("a", decimal128))),
					&struct {
						A decimal.Decimal128
					}{
						A: decimal128,
					},
					&struct {
						A decimal.Decimal128
					}{},
					nil,
				},
				{
					"decimal128 pointer",
					docToBytes(NewDocument(EC.Decimal128("a", decimal128))),
					&struct {
						A *decimal.Decimal128
					}{
						A: &decimal128,
					},
					&struct {
						A *decimal.Decimal128
					}{},
					nil,
				},
			}
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					d := NewDecoder(bytes.NewBuffer(tc.reader))

					err := d.Decode(tc.actual)
					requireErrEqual(t, tc.err, err)
					if err != nil {
						return
					}

					require.True(t, reflect.DeepEqual(tc.expected, tc.actual))
				})
			}
		})
	})

	t.Run("mixed types", func(t *testing.T) {
		stringValue := "baz"
		testCases := []struct {
			name     string
			reader   *bytes.Buffer
			expected interface{}
			actual   interface{}
			err      error
		}{
			{
				"struct containing slice",
				bytes.NewBuffer([]byte{
					// length
					0x1a, 0x0, 0x0, 0x0,

					// --- begin array ---

					// type - array
					0x4,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,

					// length
					0x10, 0x0, 0x0, 0x0,
					// type - string
					0x2,
					// key - "0"
					0x30, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - string
					0x62, 0x61, 0x7a, 0x0,

					// null terminator
					0x0,

					// --- end array ---

					// null terminator
					0x0,
				}),
				&struct {
					Foo []string
				}{
					[]string{"baz"},
				},
				&struct {
					Foo []string
				}{},
				nil,
			},
			{
				"struct containing slice pointer",
				bytes.NewBuffer([]byte{
					// length
					0x1a, 0x0, 0x0, 0x0,

					// --- begin array ---

					// type - array
					0x4,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,

					// length
					0x10, 0x0, 0x0, 0x0,
					// type - string
					0x2,
					// key - "0"
					0x30, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - string
					0x62, 0x61, 0x7a, 0x0,

					// null terminator
					0x0,

					// --- end array ---

					// null terminator
					0x0,
				}),
				&struct {
					Foo *[]string
				}{
					&[]string{"baz"},
				},
				&struct {
					Foo *[]string
				}{},
				nil,
			},
			{
				"struct containing array",
				bytes.NewBuffer([]byte{
					// length
					0x1a, 0x0, 0x0, 0x0,

					// --- begin array ---

					// type - array
					0x4,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,

					// length
					0x10, 0x0, 0x0, 0x0,
					// type - string
					0x2,
					// key - "0"
					0x30, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - "baz"
					0x62, 0x61, 0x7a, 0x0,

					// null terminator
					0x0,

					// --- end array ---

					// null terminator
					0x0,
				}),
				&struct {
					Foo [1]string
				}{
					[...]string{"baz"},
				},
				&struct {
					Foo [1]string
				}{},
				nil,
			},
			{
				"struct containing array pointer",
				bytes.NewBuffer([]byte{
					// length
					0x1a, 0x0, 0x0, 0x0,

					// --- begin array ---

					// type - array
					0x4,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,

					// length
					0x10, 0x0, 0x0, 0x0,
					// type - string
					0x2,
					// key - "0"
					0x30, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - "baz"
					0x62, 0x61, 0x7a, 0x0,

					// null terminator
					0x0,

					// --- end array ---

					// null terminator
					0x0,
				}),
				&struct {
					Foo *[1]string
				}{
					&[...]string{"baz"},
				},
				&struct {
					Foo *[1]string
				}{},
				nil,
			},
			{
				"struct containing map",
				bytes.NewBuffer([]byte{
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
				}),
				&struct {
					Foo map[string]string
				}{
					map[string]string{
						"bar": "baz",
					},
				},
				&struct {
					Foo map[string]string
				}{},
				nil,
			},

			{
				"struct containing map of pointers",
				bytes.NewBuffer([]byte{
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
				}),
				&struct {
					Foo map[string]*string
				}{
					map[string]*string{
						"bar": &stringValue,
					},
				},
				&struct {
					Foo map[string]*string
				}{},
				nil,
			},

			{
				"struct containing document",
				bytes.NewBuffer([]byte{
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
				}),
				&struct {
					Foo *Document
				}{
					NewDocument(
						EC.String("bar", "baz"),
					),
				},
				&struct {
					Foo *Document
				}{},
				nil,
			},

			{
				"struct containing reader",
				bytes.NewBuffer([]byte{
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
				}),
				&struct {
					Foo Reader
				}{
					Reader{
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
					},
				},
				&struct {
					Foo Reader
				}{},
				nil,
			},
			{
				"struct containing reader pointer",
				bytes.NewBuffer([]byte{
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
				}),
				&struct {
					Foo *Reader
				}{
					&Reader{
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
					},
				},
				&struct {
					Foo *Reader
				}{},
				nil,
			},

			{
				"map containing slice",
				bytes.NewBuffer([]byte{
					// length
					0x1a, 0x0, 0x0, 0x0,

					// --- begin array ---

					// type - array
					0x4,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,

					// length
					0x10, 0x0, 0x0, 0x0,
					// type - string
					0x2,
					// key - "0"
					0x30, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - "baz"
					0x62, 0x61, 0x7a, 0x0,

					// null terminator
					0x0,

					// --- end array ---

					// null terminator
					0x0,
				}),
				map[string][]string{
					"foo": {"baz"},
				},
				make(map[string][]string),
				nil,
			},
			{
				"map containing slice pointer",
				bytes.NewBuffer([]byte{
					// length
					0x1a, 0x0, 0x0, 0x0,

					// --- begin array ---

					// type - array
					0x4,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,

					// length
					0x10, 0x0, 0x0, 0x0,
					// type - string
					0x2,
					// key - "0"
					0x30, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - "baz"
					0x62, 0x61, 0x7a, 0x0,

					// null terminator
					0x0,

					// --- end array ---

					// null terminator
					0x0,
				}),
				map[string]*[]string{
					"foo": {"baz"},
				},
				make(map[string]*[]string),
				nil,
			},
			{
				"map containing array",
				bytes.NewBuffer([]byte{
					// length
					0x1a, 0x0, 0x0, 0x0,

					// --- begin array ---

					// type - array
					0x4,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,

					// length
					0x10, 0x0, 0x0, 0x0,
					// type - string
					0x2,
					// key - "0"
					0x30, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - "baz"
					0x62, 0x61, 0x7a, 0x0,

					// null terminator
					0x0,

					// --- end array ---

					// null terminator
					0x0,
				}),
				map[string][1]string{
					"foo": {"baz"},
				},
				make(map[string][1]string),
				nil,
			},
			{
				"map containing array pointer",
				bytes.NewBuffer([]byte{
					// length
					0x1a, 0x0, 0x0, 0x0,

					// --- begin array ---

					// type - array
					0x4,
					// key - "foo"
					0x66, 0x6f, 0x6f, 0x0,

					// length
					0x10, 0x0, 0x0, 0x0,
					// type - string
					0x2,
					// key - "0"
					0x30, 0x0,
					// value - string length
					0x4, 0x0, 0x0, 0x0,
					// value - "baz"
					0x62, 0x61, 0x7a, 0x0,

					// null terminator
					0x0,

					// --- end array ---

					// null terminator
					0x0,
				}),
				map[string]*[1]string{
					"foo": {"baz"},
				},
				make(map[string]*[1]string),
				nil,
			},
			{
				"map containing struct",
				bytes.NewBuffer([]byte{
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
				}),
				map[string]struct{ Bar string }{
					"foo": {Bar: "baz"},
				},
				make(map[string]struct{ Bar string }),
				nil,
			},
			{
				"map containing pointer to struct",
				bytes.NewBuffer([]byte{
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
				}),
				map[string]*struct{ Bar string }{
					"foo": {Bar: "baz"},
				},
				make(map[string]*struct{ Bar string }),
				nil,
			},
			{
				"map containing struct with pointers",
				bytes.NewBuffer([]byte{
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
				}),
				map[string]struct{ Bar *string }{
					"foo": {Bar: &stringValue},
				},
				make(map[string]struct{ Bar *string }),
				nil,
			},
			{
				"map containing pointer to struct with pointers",
				bytes.NewBuffer([]byte{
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
				}),
				map[string]*struct{ Bar *string }{
					"foo": {Bar: &stringValue},
				},
				make(map[string]*struct{ Bar *string }),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				d := NewDecoder(tc.reader)

				err := d.Decode(tc.actual)
				requireErrEqual(t, tc.err, err)
				if err != nil {
					return
				}

				require.True(t, cmp.Equal(tc.actual, tc.expected))
			})
		}
	})
	t.Run("pluggable types", func(t *testing.T) {
		intJSONNumber := json.Number("5")
		floatJSONNumber := json.Number("10.1")
		murl, err := url.Parse("https://mongodb.com/random-url?hello=world")
		if err != nil {
			t.Errorf("Error parsing URL: %v", err)
			t.FailNow()
		}
		testCases := []struct {
			name     string
			reader   []byte
			expected interface{}
			actual   interface{}
			err      error
		}{
			{
				"*url.URL",
				docToBytes(NewDocument(EC.String("a", murl.String()))),
				&struct {
					A *url.URL
				}{
					A: murl,
				},
				&struct {
					A *url.URL
				}{},
				nil,
			},
			{
				"json.Number",
				docToBytes(NewDocument(EC.Int64("a", 5), EC.Double("b", 10.10))),
				&struct {
					A json.Number
					B json.Number
				}{
					A: json.Number("5"),
					B: json.Number("10.1"),
				},
				&struct {
					A json.Number
					B json.Number
				}{},
				nil,
			},
			{
				"json.Number pointer",
				docToBytes(NewDocument(EC.Int64("a", 5), EC.Double("b", 10.10))),
				&struct {
					A *json.Number
					B *json.Number
				}{
					A: &intJSONNumber,
					B: &floatJSONNumber,
				},
				&struct {
					A *json.Number
					B *json.Number
				}{},
				nil,
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				d := NewDecoder(bytes.NewBuffer(tc.reader))

				err := d.Decode(tc.actual)
				requireErrEqual(t, tc.err, err)
				if err != nil {
					return
				}

				require.True(t, reflect.DeepEqual(tc.expected, tc.actual))
			})
		}
	})
}

func elementSliceEqual(t *testing.T, e1 []*Element, e2 []*Element) {
	require.Equal(t, len(e1), len(e2))

	for i := range e1 {
		require.True(t, readerElementComparer(e1[i], e2[i]))
	}
}
