// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

func TestReadArray(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		ioReader io.Reader
		arr      RawArray
		err      error
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
			"empty Array",
			bytes.NewBuffer([]byte{5, 0, 0, 0, 0}),
			[]byte{5, 0, 0, 0, 0},
			nil,
		},
		{
			"non-empty Array",
			bytes.NewBuffer([]byte{
				'\x1B', '\x00', '\x00', '\x00',
				'\x02',
				'0', '\x00',
				'\x04', '\x00', '\x00', '\x00',
				'\x62', '\x61', '\x72', '\x00',
				'\x02',
				'1', '\x00',
				'\x04', '\x00', '\x00', '\x00',
				'\x62', '\x61', '\x7a', '\x00',
				'\x00',
			}),
			[]byte{
				'\x1B', '\x00', '\x00', '\x00',
				'\x02',
				'0', '\x00',
				'\x04', '\x00', '\x00', '\x00',
				'\x62', '\x61', '\x72', '\x00',
				'\x02',
				'1', '\x00',
				'\x04', '\x00', '\x00', '\x00',
				'\x62', '\x61', '\x7a', '\x00',
				'\x00',
			},
			nil,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			arr, err := ReadArray(tc.ioReader)
			if !assert.CompareErrors(err, tc.err) {
				t.Errorf("errors do not match. got %v; want %v", err, tc.err)
			}
			if !bytes.Equal(tc.arr, arr) {
				t.Errorf("Arrays differ. got %v; want %v", tc.arr, arr)
			}
		})
	}
}

func TestArray_Validate(t *testing.T) {
	t.Parallel()

	t.Run("TooShort", func(t *testing.T) {
		t.Parallel()

		want := bsoncore.NewInsufficientBytesError(nil, nil)
		got := RawArray{'\x00', '\x00'}.Validate()
		if !assert.CompareErrors(got, want) {
			t.Errorf("Did not get expected error. got %v; want %v", got, want)
		}
	})

	t.Run("InvalidLength", func(t *testing.T) {
		t.Parallel()

		want := bsoncore.NewArrayLengthError(200, 5)
		r := make(RawArray, 5)
		binary.LittleEndian.PutUint32(r[0:4], 200)
		got := r.Validate()
		if !assert.CompareErrors(got, want) {
			t.Errorf("Did not get expected error. got %v; want %v", got, want)
		}
	})

	t.Run("Invalid Element", func(t *testing.T) {
		t.Parallel()

		want := bsoncore.NewInsufficientBytesError(nil, nil)
		r := make(RawArray, 7)
		binary.LittleEndian.PutUint32(r[0:4], 7)
		r[4], r[5], r[6] = 0x02, 'f', 0x00
		got := r.Validate()
		if !assert.CompareErrors(got, want) {
			t.Errorf("Did not get expected error. got %v; want %v", got, want)
		}
	})

	t.Run("Missing Null Terminator", func(t *testing.T) {
		t.Parallel()

		want := bsoncore.ErrMissingNull
		r := make(RawArray, 6)
		binary.LittleEndian.PutUint32(r[0:4], 6)
		r[4], r[5] = 0x0A, '0'
		got := r.Validate()
		if !assert.CompareErrors(got, want) {
			t.Errorf("Did not get expected error. got %v; want %v", got, want)
		}
	})

	testCases := []struct {
		name string
		r    RawArray
		want error
	}{
		{"array null", RawArray{'\x08', '\x00', '\x00', '\x00', '\x0A', '0', '\x00', '\x00'}, nil},
		{
			"array",
			RawArray{
				'\x1B', '\x00', '\x00', '\x00',
				'\x02',
				'0', '\x00',
				'\x04', '\x00', '\x00', '\x00',
				'\x62', '\x61', '\x72', '\x00',
				'\x02',
				'1', '\x00',
				'\x04', '\x00', '\x00', '\x00',
				'\x62', '\x61', '\x7a', '\x00',
				'\x00',
			},
			nil,
		},
		{
			"subarray",
			RawArray{
				'\x13', '\x00', '\x00', '\x00',
				'\x04',
				'0', '\x00',
				'\x0B', '\x00', '\x00', '\x00', '\x0A', '0', '\x00',
				'\x0A', '1', '\x00', '\x00', '\x00',
			},
			nil,
		},
		{
			"invalid key order",
			RawArray{
				'\x0B', '\x00', '\x00', '\x00', '\x0A', '2', '\x00',
				'\x0A', '0', '\x00', '\x00', '\x00',
			},
			errors.New(`array key "2" is out of order or invalid`),
		},
		{
			"invalid key type",
			RawArray{
				'\x0B', '\x00', '\x00', '\x00', '\x0A', 'p', '\x00',
				'\x0A', 'q', '\x00', '\x00', '\x00',
			},
			errors.New(`array key "p" is out of order or invalid`),
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture the range variable

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.r.Validate()
			if !assert.CompareErrors(got, tc.want) {
				t.Errorf("Returned error does not match. got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestArray_Index(t *testing.T) {
	t.Parallel()

	t.Run("Out of bounds", func(t *testing.T) {
		t.Parallel()

		rdr := RawArray{0xe, 0x0, 0x0, 0x0, 0xa, '0', 0x0, 0xa, '1', 0x0, 0xa, 0x7a, 0x0, 0x0}
		_, err := rdr.IndexErr(3)
		if !errors.Is(err, bsoncore.ErrOutOfBounds) {
			t.Errorf("Out of bounds should be returned when accessing element beyond end of Array. got %v; want %v", err,
				bsoncore.ErrOutOfBounds)
		}
	})

	t.Run("Validation Error", func(t *testing.T) {
		t.Parallel()

		rdr := RawArray{0x07, 0x00, 0x00, 0x00, 0x00}
		_, got := rdr.IndexErr(1)
		want := bsoncore.NewInsufficientBytesError(nil, nil)
		if !assert.CompareErrors(got, want) {
			t.Errorf("Did not receive expected error. got %v; want %v", got, want)
		}
	})

	testArray := RawArray{
		'\x26', '\x00', '\x00', '\x00',
		'\x02',
		'0', '\x00',
		'\x04', '\x00', '\x00', '\x00',
		'\x62', '\x61', '\x72', '\x00',
		'\x02',
		'1', '\x00',
		'\x04', '\x00', '\x00', '\x00',
		'\x62', '\x61', '\x7a', '\x00',
		'\x02',
		'2', '\x00',
		'\x04', '\x00', '\x00', '\x00',
		'\x71', '\x75', '\x78', '\x00',
		'\x00',
	}
	testCases := []struct {
		name  string
		index uint
		want  RawValue
	}{
		{
			"first",
			0,
			RawValue{
				Type:  TypeString,
				Value: []byte{0x04, 0x00, 0x00, 0x00, 0x62, 0x61, 0x72, 0x00},
			},
		},
		{
			"second",
			1,
			RawValue{
				Type:  TypeString,
				Value: []byte{0x04, 0x00, 0x00, 0x00, 0x62, 0x61, 0x7a, 0x00},
			},
		},
		{
			"third",
			2,
			RawValue{
				Type:  TypeString,
				Value: []byte{0x04, 0x00, 0x00, 0x00, 0x71, 0x75, 0x78, 0x00},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture the range variable

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("IndexErr", func(t *testing.T) {
				t.Parallel()

				got, err := testArray.IndexErr(tc.index)
				if err != nil {
					t.Errorf("Unexpected error from IndexErr: %s", err)
				}
				if diff := cmp.Diff(got, tc.want); diff != "" {
					t.Errorf("Arrays differ: (-got +want)\n%s", diff)
				}
			})

			t.Run("Index", func(t *testing.T) {
				t.Parallel()

				defer func() {
					if err := recover(); err != nil {
						t.Errorf("Unexpected error: %v", err)
					}
				}()
				got := testArray.Index(tc.index)
				if diff := cmp.Diff(got, tc.want); diff != "" {
					t.Errorf("Arrays differ: (-got +want)\n%s", diff)
				}
			})
		})
	}
}

func TestRawArray_Stringt(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		arr              RawArray
		arrayString      string
		arrayDebugString string
	}{
		{
			"array",
			RawArray{
				'\x1B', '\x00', '\x00', '\x00',
				'\x02',
				'0', '\x00',
				'\x04', '\x00', '\x00', '\x00',
				'\x62', '\x61', '\x72', '\x00',
				'\x02',
				'1', '\x00',
				'\x04', '\x00', '\x00', '\x00',
				'\x62', '\x61', '\x7a', '\x00',
				'\x00',
			},
			`["bar","baz"]`,
			`Array(27)["bar","baz"]`,
		},
		{
			"subarray",
			RawArray{
				'\x13', '\x00', '\x00', '\x00',
				'\x04',
				'0', '\x00',
				'\x0B', '\x00', '\x00', '\x00',
				'\x0A', '0', '\x00',
				'\x0A', '1', '\x00',
				'\x00', '\x00',
			},
			`[[null,null]]`,
			`Array(19)[Array(11)[null,null]]`,
		},
		{
			"malformed--length too small",
			RawArray{
				'\x04', '\x00', '\x00', '\x00',
				'\x00',
			},
			``,
			`Array(4)[]`,
		},
		{
			"malformed--length too large",
			RawArray{
				'\x13', '\x00', '\x00', '\x00',
				'\x00',
			},
			``,
			`Array(19)[<malformed (15)>]`,
		},
		{
			"malformed--missing null byte",
			RawArray{
				'\x06', '\x00', '\x00', '\x00',
				'\x02', '0',
			},
			``,
			`Array(6)[<malformed (2)>]`,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture the range variable

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			arrayString := tc.arr.String()
			if arrayString != tc.arrayString {
				t.Errorf("array strings do not match. got %q; want %q",
					arrayString, tc.arrayString)
			}

			arrayDebugString := tc.arr.DebugString()
			if arrayDebugString != tc.arrayDebugString {
				t.Errorf("array debug strings do not match. got %q; want %q",
					arrayDebugString, tc.arrayDebugString)
			}
		})
	}
}

func TestRawArray_Values(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		arr  RawArray
		want []RawValue
	}{
		{
			name: "empty",
			arr:  []byte{0x05, 0x00, 0x00, 0x00, 0x00},
			want: []RawValue{},
		},
		{
			name: "null document",
			arr:  []byte{0x07, 0x00, 0x00, 0x00, 0x0A, 0x00, 0x00}, // [{}]
			want: []RawValue{
				{
					Type:  TypeNull,
					Value: []byte{},
				},
			},
		},
		{
			name: "same",
			arr: []byte{
				0x13, 0x00, 0x00, 0x00, 0x10, 0x30, 0x00, 0x01, 0x00, 0x00,
				0x00, 0x10, 0x31, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00,
			}, // [int32(1), int32(2)]
			want: []RawValue{
				{ // int32(1)
					Type:  TypeInt32,
					Value: []byte{0x01, 0x00, 0x00, 0x00},
				},
				{ // int32(2)
					Type:  TypeInt32,
					Value: []byte{0x02, 0x00, 0x00, 0x00},
				},
			},
		},
		{
			name: "mixed",
			arr: []byte{
				0x17, 0x00, 0x00, 0x00, 0x10, 0x30, 0x00, 0x01, 0x00, 0x00,
				0x00, 0x02, 0x31, 0x00, 0x04, 0x00, 0x00, 0x00, 0x66, 0x6F, 0x6F, 0x00, 0x00,
			}, // [int32(1), "foo"]
			want: []RawValue{
				{ // int32(1)
					Type:  TypeInt32,
					Value: []byte{0x01, 0x00, 0x00, 0x00},
				},
				{ // "foo"
					Type:  TypeString,
					Value: []byte{0x04, 0x00, 0x00, 0x00, 0x66, 0x6F, 0x6F, 0x00},
				},
			},
		},
	}

	for _, tcase := range testCases {
		tcase := tcase

		t.Run(tcase.name, func(t *testing.T) {
			t.Parallel()

			values, err := tcase.arr.Values()
			require.NoError(t, err, "failed to turn array into values")
			require.Len(t, values, len(tcase.want), "got len does not match want")

			for idx, want := range tcase.want {
				assert.True(t, want.Equal(values[idx]), "want: %v, got: %v", want, values[idx])
			}
		})
	}
}
