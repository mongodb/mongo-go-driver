// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
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
			want := lengthError("array", 200, 5)
			r := make(Array, 5)
			binary.LittleEndian.PutUint32(r[0:4], 200)
			got := r.Validate()
			if !compareErrors(got, want) {
				t.Errorf("Did not get expected error. got %v; want %v", got, want)
			}
		})
		t.Run("Invalid Element", func(t *testing.T) {
			want := NewInsufficientBytesError(nil, nil)
			r := make(Array, 9)
			binary.LittleEndian.PutUint32(r[0:4], 9)
			r[4], r[5], r[6], r[7], r[8] = 0x02, 'f', 'o', 'o', 0x00
			got := r.Validate()
			if !compareErrors(got, want) {
				t.Errorf("Did not get expected error. got %v; want %v", got, want)
			}
		})
		t.Run("Missing Null Terminator", func(t *testing.T) {
			want := ErrMissingNull
			r := make(Array, 8)
			binary.LittleEndian.PutUint32(r[0:4], 8)
			r[4], r[5], r[6], r[7] = 0x0A, 'f', 'o', 'o'
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
			{"null", Array{'\x08', '\x00', '\x00', '\x00', '\x0A', 'x', '\x00', '\x00'}, nil},
			{"subarray",
				Array{
					'\x15', '\x00', '\x00', '\x00',
					'\x03',
					'f', 'o', 'o', '\x00',
					'\x0B', '\x00', '\x00', '\x00', '\x0A', 'a', '\x00',
					'\x0A', 'b', '\x00', '\x00', '\x00',
				},
				nil,
			},
			{"array",
				Array{
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
				got := tc.r.Validate()
				if !compareErrors(got, tc.want) {
					t.Errorf("Returned error does not match. got %v; want %v", got, tc.want)
				}
			})
		}
	})
	t.Run("Index", func(t *testing.T) {
		t.Run("Out of bounds", func(t *testing.T) {
			rdr := Array{0xe, 0x0, 0x0, 0x0, 0xa, 0x78, 0x0, 0xa, 0x79, 0x0, 0xa, 0x7a, 0x0, 0x0}
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
		testCases := []struct {
			name  string
			rdr   Array
			index uint
			want  Element
		}{
			{"first",
				Array{0xe, 0x0, 0x0, 0x0, 0xa, 0x78, 0x0, 0xa, 0x79, 0x0, 0xa, 0x7a, 0x0, 0x0},
				0, Element{0x0a, 0x78, 0x00},
			},
			{"second",
				Array{0xe, 0x0, 0x0, 0x0, 0xa, 0x78, 0x0, 0xa, 0x79, 0x0, 0xa, 0x7a, 0x0, 0x0},
				1, Element{0x0a, 0x79, 0x00},
			},
			{"third",
				Array{0xe, 0x0, 0x0, 0x0, 0xa, 0x78, 0x0, 0xa, 0x79, 0x0, 0xa, 0x7a, 0x0, 0x0},
				2, Element{0x0a, 0x7a, 0x00},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Run("IndexErr", func(t *testing.T) {
					got, err := tc.rdr.IndexErr(tc.index)
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
					got := tc.rdr.Index(tc.index)
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
}
