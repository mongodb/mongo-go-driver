// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

func TestArray(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		t.Run("TooShort", func(t *testing.T) {
			want := NewInsufficientBytesError(nil, nil)
			got := Array{'\x00', '\x00'}.Validate()
			if !compareErrors(got, want) {
				t.Errorf("Did not get expected error. got %v; want %v", got, want)
			}
		})
		t.Run("InvalidLength", func(t *testing.T) {
			want := NewArrayLengthError(200, 5)
			r := make(Array, 5)
			binary.LittleEndian.PutUint32(r[0:4], 200)
			got := r.Validate()
			if !compareErrors(got, want) {
				t.Errorf("Did not get expected error. got %v; want %v", got, want)
			}
		})
		t.Run("Invalid Element", func(t *testing.T) {
			want := NewInsufficientBytesError(nil, nil)
			r := make(Array, 7)
			binary.LittleEndian.PutUint32(r[0:4], 7)
			r[4], r[5], r[6] = 0x02, 'f', 0x00
			got := r.Validate()
			if !compareErrors(got, want) {
				t.Errorf("Did not get expected error. got %v; want %v", got, want)
			}
		})
		t.Run("Missing Null Terminator", func(t *testing.T) {
			want := ErrMissingNull
			r := make(Array, 6)
			binary.LittleEndian.PutUint32(r[0:4], 6)
			r[4], r[5] = 0x0A, '0'
			got := r.Validate()
			if !compareErrors(got, want) {
				t.Errorf("Did not get expected error. got %v; want %v", got, want)
			}
		})
		testCases := []struct {
			name string
			r    Array
			want error
		}{
			{"array null", Array{'\x08', '\x00', '\x00', '\x00', '\x0A', '0', '\x00', '\x00'}, nil},
			{"array",
				Array{
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
			{"subarray",
				Array{
					'\x13', '\x00', '\x00', '\x00',
					'\x04',
					'0', '\x00',
					'\x0B', '\x00', '\x00', '\x00', '\x0A', '0', '\x00',
					'\x0A', '1', '\x00', '\x00', '\x00',
				},
				nil,
			},
			{"invalid key order",
				Array{
					'\x0B', '\x00', '\x00', '\x00', '\x0A', '2', '\x00',
					'\x0A', '0', '\x00', '\x00', '\x00',
				},
				errors.New(`array key "2" is out of order or invalid`),
			},
			{"invalid key type",
				Array{
					'\x0B', '\x00', '\x00', '\x00', '\x0A', 'p', '\x00',
					'\x0A', 'q', '\x00', '\x00', '\x00',
				},
				errors.New(`array key "p" is out of order or invalid`),
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				got := tc.r.Validate()
				if !compareErrors(got, tc.want) {
					t.Errorf("Returned error does not match. got %v; want %v", got, tc.want)
				}
			})
		}
	})
	t.Run("Index", func(t *testing.T) {
		t.Run("Out of bounds", func(t *testing.T) {
			rdr := Array{0xe, 0x0, 0x0, 0x0, 0xa, '0', 0x0, 0xa, '1', 0x0, 0xa, 0x7a, 0x0, 0x0}
			_, err := rdr.IndexErr(3)
			if err != ErrOutOfBounds {
				t.Errorf("Out of bounds should be returned when accessing element beyond end of Array. got %v; want %v", err, ErrOutOfBounds)
			}
		})
		t.Run("Validation Error", func(t *testing.T) {
			rdr := Array{0x07, 0x00, 0x00, 0x00, 0x00}
			_, got := rdr.IndexErr(1)
			want := NewInsufficientBytesError(nil, nil)
			if !compareErrors(got, want) {
				t.Errorf("Did not receive expected error. got %v; want %v", got, want)
			}
		})
		testArray := Array{
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
			want  Value
		}{
			{"first",
				0,
				Value{
					Type: bsontype.String,
					Data: []byte{0x04, 0x00, 0x00, 0x00, 0x62, 0x61, 0x72, 0x00},
				},
			},
			{"second",
				1,
				Value{
					Type: bsontype.String,
					Data: []byte{0x04, 0x00, 0x00, 0x00, 0x62, 0x61, 0x7a, 0x00},
				},
			},
			{"third",
				2,
				Value{
					Type: bsontype.String,
					Data: []byte{0x04, 0x00, 0x00, 0x00, 0x71, 0x75, 0x78, 0x00},
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Run("IndexErr", func(t *testing.T) {
					got, err := testArray.IndexErr(tc.index)
					if err != nil {
						t.Errorf("Unexpected error from IndexErr: %s", err)
					}
					if diff := cmp.Diff(got, tc.want); diff != "" {
						t.Errorf("Arrays differ: (-got +want)\n%s", diff)
					}
				})
				t.Run("Index", func(t *testing.T) {
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
	})
	t.Run("NewArrayFromReader", func(t *testing.T) {
		testCases := []struct {
			name     string
			ioReader io.Reader
			arr      Array
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
			t.Run(tc.name, func(t *testing.T) {
				arr, err := NewArrayFromReader(tc.ioReader)
				if !compareErrors(err, tc.err) {
					t.Errorf("errors do not match. got %v; want %v", err, tc.err)
				}
				if !bytes.Equal(tc.arr, arr) {
					t.Errorf("Arrays differ. got %v; want %v", tc.arr, arr)
				}
			})
		}
	})
	t.Run("DebugString", func(t *testing.T) {
		testCases := []struct {
			name             string
			arr              Array
			arrayString      string
			arrayDebugString string
		}{
			{
				"array",
				Array{
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
				Array{
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
				Array{
					'\x04', '\x00', '\x00', '\x00',
					'\x00',
				},
				``,
				`Array(4)[]`,
			},
			{
				"malformed--length too large",
				Array{
					'\x13', '\x00', '\x00', '\x00',
					'\x00',
				},
				``,
				`Array(19)[<malformed (15)>]`,
			},
			{
				"malformed--missing null byte",
				Array{
					'\x06', '\x00', '\x00', '\x00',
					'\x02', '0',
				},
				``,
				`Array(6)[<malformed (2)>]`,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
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
	})
}
