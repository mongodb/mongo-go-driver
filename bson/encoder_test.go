// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsonrw/bsonrwtest"
)

func TestBasicEncode(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := make(bsonrw.SliceWriter, 0, 1024)
			vw, err := bsonrw.NewBSONValueWriter(&got)
			noerr(t, err)
			reg := DefaultRegistry
			encoder, err := reg.LookupEncoder(reflect.TypeOf(tc.val))
			noerr(t, err)
			err = encoder.EncodeValue(bsoncodec.EncodeContext{Registry: reg}, vw, reflect.ValueOf(tc.val))
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", got, tc.want)
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}

func TestEncoderEncode(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := make(bsonrw.SliceWriter, 0, 1024)
			vw, err := bsonrw.NewBSONValueWriter(&got)
			noerr(t, err)
			enc, err := NewEncoder(vw)
			noerr(t, err)
			err = enc.Encode(tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", got, tc.want)
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}

	t.Run("Marshaler", func(t *testing.T) {
		testCases := []struct {
			name    string
			buf     []byte
			err     error
			wanterr error
			vw      bsonrw.ValueWriter
		}{
			{
				"error",
				nil,
				errors.New("Marshaler error"),
				errors.New("Marshaler error"),
				&bsonrwtest.ValueReaderWriter{},
			},
			{
				"copy error",
				[]byte{0x05, 0x00, 0x00, 0x00, 0x00},
				nil,
				errors.New("copy error"),
				&bsonrwtest.ValueReaderWriter{Err: errors.New("copy error"), ErrAfter: bsonrwtest.WriteDocument},
			},
			{
				"success",
				[]byte{0x07, 0x00, 0x00, 0x00, 0x0A, 0x00, 0x00},
				nil,
				nil,
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				marshaler := testMarshaler{buf: tc.buf, err: tc.err}

				var vw bsonrw.ValueWriter
				var err error
				b := make(bsonrw.SliceWriter, 0, 100)
				compareVW := false
				if tc.vw != nil {
					vw = tc.vw
				} else {
					compareVW = true
					vw, err = bsonrw.NewBSONValueWriter(&b)
					noerr(t, err)
				}
				enc, err := NewEncoder(vw)
				noerr(t, err)
				got := enc.Encode(marshaler)
				want := tc.wanterr
				if !compareErrors(got, want) {
					t.Errorf("Did not receive expected error. got %v; want %v", got, want)
				}
				if compareVW {
					buf := b
					if !bytes.Equal(buf, tc.buf) {
						t.Errorf("Copied bytes do not match. got %v; want %v", buf, tc.buf)
					}
				}
			})
		}
	})
}

func FuzzDecode(f *testing.F) {
	seedCorpus := []struct{ buf []byte }{
		// Empty document.
		{[]byte{}},

		// Document with an embedded array document.
		// { "slice": [1,2.0,"3",0x3a,0x05,[ 6, 7.0, "8", 0x39, 0x0a ]] }
		{[]byte{0x80, 0x00, 0x00, 0x00, 0x03, 0x6f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x00, 0x73, 0x00, 0x00, 0x00, 0x04, 0x73, 0x6c, 0x69, 0x63, 0x65, 0x00, 0x67, 0x00, 0x00, 0x00, 0x10, 0x30, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x31, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x40, 0x02, 0x32, 0x00, 0x02, 0x00, 0x00, 0x00, 0x33, 0x00, 0x05, 0x33, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x34, 0x05, 0x34, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x05, 0x04, 0x35, 0x00, 0x32, 0x00, 0x00, 0x00, 0x10, 0x30, 0x00, 0x06, 0x00, 0x00, 0x00, 0x01, 0x31, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1c, 0x40, 0x02, 0x32, 0x00, 0x02, 0x00, 0x00, 0x00, 0x38, 0x00, 0x05, 0x33, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x39, 0x05, 0x34, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x0a, 0x00, 0x00, 0x00, 0x00}},
	}

	for _, tc := range seedCorpus {
		f.Add(tc.buf)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		for _, typ := range []func() interface{}{
			func() interface{} { return new(interface{}) },
		} {
			i := typ()
			if err := Unmarshal(data, i); err != nil {
				return
			}

			encoded, err := Marshal(i)
			if err != nil {
				t.Fatal("failed to marshal", err)
			}

			if err := Unmarshal(encoded, i); err != nil {
				t.Fatal("failed to unmarshal", err)
			}
		}
	})
}

type testMarshaler struct {
	buf []byte
	err error
}

func (tm testMarshaler) MarshalBSON() ([]byte, error) { return tm.buf, tm.err }

func docToBytes(d interface{}) []byte {
	b, err := Marshal(d)
	if err != nil {
		panic(err)
	}
	return b
}
