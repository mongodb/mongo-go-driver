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
	// corpus represent the initial seed data for the fuzzer and should only include minimal cases that expand the
	// fuzzer's search space (i.e. code coverage).
	corpus := []struct{ buf []byte }{
		{
			buf: []byte(""),
		},
		{
			buf: []byte("\x05\xf0\xff\x00\x7f"),
		},
		{
			buf: []byte("0\x00\x00\x00\x0f\x00000\x8a00000000000000000000000000000000000000"),
		},
		{
			buf: []byte("\x80\x00\x00\x00\x03000000\x00s\x00\x00\x00\x0300000\x00g\x00\x00\x00\x100z\x000000\x11\x00000\x150000\x020\x00\x02\x00\x00\x000\x12\x00\x050\x00\x01\x00\x00\x0000\x050\x00\x01\x00\x00\x0000\x040\x00200000\x00\x000\x02\x00\x10\x0000000\x110\x0000000000\x020\x00\x02\x00\x00\x000\x00\x050\x00\x01\x00\x00\x0000\x050\x00\x01\x00\x00\x0000\x00\x00\x00\x00"),
		},
		{
			//{
			//    double: 1.1,
			//    string: "hello",
			//    embedded: {
			//        array: [1, 2.0, "3", [4], {
			//            "5": {}
			//        }]
			//    },
			//    binary: Binary('AQID', 'valid_base64'),
			//    boolean: true,
			//    "null": null,
			//    regex: RegExp('hello', 'i'),
			//    js: "function() {}",
			//    scope: Code('function() {}'),
			//    int32: 32,
			//    timestamp: Timestamp(1, 2),
			//    int64: 64,
			//    minKey: MinKey(),
			//    maxKey: MaxKey()
			//}
			buf: []byte("\x59\x01\x00\x00\x01\x64\x6f\x75\x62\x6c\x65\x00\x9a\x99\x99\x99\x99\x99\xf1\x3f\x02\x73\x74\x72\x69\x6e\x67\x00\x06\x00\x00\x00\x68\x65\x6c\x6c\x6f\x00\x03\x65\x6d\x62\x65\x64\x64\x65\x64\x00\x4b\x00\x00\x00\x04\x61\x72\x72\x61\x79\x00\x3f\x00\x00\x00\x10\x30\x00\x01\x00\x00\x00\x01\x31\x00\x00\x00\x00\x00\x00\x00\x00\x40\x02\x32\x00\x02\x00\x00\x00\x33\x00\x04\x33\x00\x0c\x00\x00\x00\x10\x30\x00\x04\x00\x00\x00\x00\x03\x34\x00\x0d\x00\x00\x00\x03\x35\x00\x05\x00\x00\x00\x00\x00\x00\x00\x05\x62\x69\x6e\x61\x72\x79\x00\x03\x00\x00\x00\x00\x01\x02\x03\x07\x6f\x62\x6a\x65\x63\x74\x69\x64\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x08\x62\x6f\x6f\x6c\x65\x61\x6e\x00\x01\x09\x64\x61\x74\x65\x74\x69\x6d\x65\x00\x00\x00\x00\x00\x00\x00\x00\x00\x0a\x6e\x75\x6c\x6c\x00\x0b\x72\x65\x67\x65\x78\x00\x68\x65\x6c\x6c\x6f\x00\x69\x00\x0d\x6a\x73\x00\x0e\x00\x00\x00\x66\x75\x6e\x63\x74\x69\x6f\x6e\x28\x29\x20\x7b\x7d\x00\x0f\x73\x63\x6f\x70\x65\x00\x2c\x00\x00\x00\x0e\x00\x00\x00\x66\x75\x6e\x63\x74\x69\x6f\x6e\x28\x29\x20\x7b\x7d\x00\x16\x00\x00\x00\x02\x68\x65\x6c\x6c\x6f\x00\x06\x00\x00\x00\x77\x6f\x72\x6c\x64\x00\x00\x10\x69\x6e\x74\x33\x32\x00\x20\x00\x00\x00\x11\x74\x69\x6d\x65\x73\x74\x61\x6d\x70\x00\x02\x00\x00\x00\x01\x00\x00\x00\x12\x69\x6e\x74\x36\x34\x00\x40\x00\x00\x00\x00\x00\x00\x00\xff\x6d\x69\x6e\x6b\x65\x79\x00\x7f\x6d\x61\x78\x6b\x65\x79\x00\x00"),
		},
	}

	for _, tc := range corpus {
		f.Add(tc.buf)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		for _, typ := range []func() interface{}{
			func() interface{} { return new(interface{}) },
			func() interface{} { return make(map[string]interface{}) },
			func() interface{} { return new([]interface{}) },
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
