// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/stretchr/testify/assert"
)

func TestEncoder(t *testing.T) {
	t.Run("Writer/Marshaler", func(t *testing.T) {
		testCases := []struct {
			name string
			m    Marshaler
			b    []byte
			err  error
		}{
			{
				"success",
				NewDocument(EC.Null("foo")),
				[]byte{
					0x0A, 0x00, 0x00, 0x00,
					0x0A, 'f', 'o', 'o', 0x00,
					0x00,
				},
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var buf bytes.Buffer
				enc := NewEncoder(&buf)
				err := enc.Encode(tc.m)
				if err != tc.err {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
				b := buf.Bytes()
				if diff := cmp.Diff(tc.b, b); diff != "" {
					t.Errorf("Bytes written differ: (-got +want)\n%s", diff)
				}
			})
		}
	})
	t.Run("Document/Document", func(t *testing.T) {
		testCases := []struct {
			name string
			d    *Document
			want *Document
			err  error
		}{
			{
				"success",
				NewDocument(EC.Null("foo")),
				NewDocument(EC.Null("foo")),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				enc := NewDocumentEncoder()
				got, err := enc.EncodeDocument(tc.d)
				if err != tc.err {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
				if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(Document{}, Element{}, Value{})); diff != "" {
					t.Errorf("Documents differ: (-got +want)\n%s", diff)
				}
			})
		}
	})
	t.Run("Writer/io.Reader", func(t *testing.T) {
		testCases := []struct {
			name string
			m    io.Reader
			b    []byte
			err  error
		}{
			{
				"success",
				bytes.NewReader([]byte{
					0x0A, 0x00, 0x00, 0x00,
					0x0A, 'f', 'o', 'o', 0x00,
					0x00,
				}),
				[]byte{
					0x0A, 0x00, 0x00, 0x00,
					0x0A, 'f', 'o', 'o', 0x00,
					0x00,
				},
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var buf bytes.Buffer
				enc := NewEncoder(&buf)
				err := enc.Encode(tc.m)
				if err != tc.err {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
				b := buf.Bytes()
				if diff := cmp.Diff(tc.b, b); diff != "" {
					t.Errorf("Bytes written differ: (-got +want)\n%s", diff)
				}
			})
		}
	})
	t.Run("Document/io.Reader", func(t *testing.T) {
		testCases := []struct {
			name string
			m    io.Reader
			want *Document
			err  error
		}{
			{
				"success",
				bytes.NewReader([]byte{
					0x0A, 0x00, 0x00, 0x00,
					0x0A, 'f', 'o', 'o', 0x00,
					0x00,
				}),
				NewDocument(EC.Null("foo")),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				enc := NewDocumentEncoder()
				got, err := enc.EncodeDocument(tc.m)
				if err != tc.err {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
				if !documentComparer(got, tc.want) {
					t.Errorf("Documents differ. got %v; want %v", got, tc.want)
				}
			})
		}
	})
	t.Run("Writer/[]byte", func(t *testing.T) {
		testCases := []struct {
			name string
			m    []byte
			b    []byte
			err  error
		}{
			{
				"success",
				[]byte{
					0x0A, 0x00, 0x00, 0x00,
					0x0A, 'f', 'o', 'o', 0x00,
					0x00,
				},
				[]byte{
					0x0A, 0x00, 0x00, 0x00,
					0x0A, 'f', 'o', 'o', 0x00,
					0x00,
				},
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var buf bytes.Buffer
				enc := NewEncoder(&buf)
				err := enc.Encode(tc.m)
				if err != tc.err {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
				b := buf.Bytes()
				if diff := cmp.Diff(tc.b, b); diff != "" {
					t.Errorf("Bytes written differ: (-got +want)\n%s", diff)
				}
			})
		}
	})
	t.Run("Document/[]byte", func(t *testing.T) {
		testCases := []struct {
			name string
			m    []byte
			want *Document
			err  error
		}{
			{
				"success",
				[]byte{
					0x0A, 0x00, 0x00, 0x00,
					0x0A, 'f', 'o', 'o', 0x00,
					0x00,
				},
				NewDocument(EC.Null("foo")),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				enc := NewDocumentEncoder()
				got, err := enc.EncodeDocument(tc.m)
				if err != tc.err {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
				if !documentComparer(got, tc.want) {
					t.Errorf("Documents differ. got %v; want %v", got, tc.want)
				}
			})
		}
	})
	t.Run("Writer/Reader", func(t *testing.T) {
		testCases := []struct {
			name string
			r    Reader
			b    []byte
			err  error
		}{
			{
				"success",
				[]byte{
					0x0A, 0x00, 0x00, 0x00,
					0x0A, 'f', 'o', 'o', 0x00,
					0x00,
				},
				[]byte{
					0x0A, 0x00, 0x00, 0x00,
					0x0A, 'f', 'o', 'o', 0x00,
					0x00,
				},
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var buf bytes.Buffer
				enc := NewEncoder(&buf)
				err := enc.Encode(tc.r)
				if err != tc.err {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
				b := buf.Bytes()
				if diff := cmp.Diff(tc.b, b); diff != "" {
					t.Errorf("Bytes written differ: (-got +want)\n%s", diff)
				}
			})
		}
	})
	t.Run("Document/Reader", func(t *testing.T) {
		testCases := []struct {
			name string
			r    Reader
			want *Document
			err  error
		}{
			{
				"success",
				[]byte{
					0x0A, 0x00, 0x00, 0x00,
					0x0A, 'f', 'o', 'o', 0x00,
					0x00,
				},
				NewDocument(EC.Null("foo")),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				enc := NewDocumentEncoder()
				got, err := enc.EncodeDocument(tc.r)
				if err != tc.err {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
				if !documentComparer(got, tc.want) {
					t.Errorf("Documents differ. got %v; want %v", got, tc.want)
				}
			})
		}
	})
	t.Run("Document/Marshaler", func(t *testing.T) {
		testCases := []struct {
			name string
			r    Marshaler
			want *Document
			err  error
		}{
			{
				"success",
				byteMarshaler([]byte{
					0x0A, 0x00, 0x00, 0x00,
					0x0A, 'f', 'o', 'o', 0x00,
					0x00,
				}),
				NewDocument(EC.Null("foo")),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				enc := NewDocumentEncoder()
				got, err := enc.EncodeDocument(tc.r)
				if err != tc.err {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
				if !documentComparer(got, tc.want) {
					t.Errorf("Documents differ. got %v; want %v", got, tc.want)
				}
			})
		}
	})
	t.Run("Writer/Reflection", reflectionEncoderTest)
	t.Run("Document/Reflection", func(t *testing.T) {
		testCases := []struct {
			name  string
			value interface{}
			want  *Document
			err   error
		}{
			{
				"struct",
				struct {
					A string
				}{
					A: "foo",
				},
				NewDocument(EC.String("a", "foo")),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				enc := NewDocumentEncoder()
				got, err := enc.EncodeDocument(tc.value)
				if err != tc.err {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
				if !documentComparer(got, tc.want) {
					t.Errorf("Documents differ. got %v; want %v", got, tc.want)
				}
			})
		}
	})
}

func reflectionEncoderTest(t *testing.T) {
	oid := objectid.New()
	oids := []objectid.ObjectID{objectid.New(), objectid.New(), objectid.New()}
	var str = new(string)
	*str = "bar"
	now := time.Now()
	murl, err := url.Parse("https://mongodb.com/random-url?hello=world")
	if err != nil {
		t.Errorf("Error parsing URL: %v", err)
		t.FailNow()
	}
	decimal128, err := decimal.ParseDecimal128("1.5e10")
	if err != nil {
		t.Errorf("Error parsing decimal128: %v", err)
		t.FailNow()
	}

	testCases := []struct {
		name  string
		value interface{}
		b     []byte
		err   error
	}{
		{
			"map[bool]int",
			map[bool]int32{false: 1},
			[]byte{
				0x10, 0x00, 0x00, 0x00,
				0x10, 'f', 'a', 'l', 's', 'e', 0x00,
				0x01, 0x00, 0x00, 0x00,
				0x00,
			},
			nil,
		},
		{
			"map[int]int",
			map[int]int32{1: 1},
			[]byte{
				0x0C, 0x00, 0x00, 0x00,
				0x10, '1', 0x00,
				0x01, 0x00, 0x00, 0x00,
				0x00,
			},
			nil,
		},
		{
			"map[uint]int",
			map[uint]int32{1: 1},
			[]byte{
				0x0C, 0x00, 0x00, 0x00,
				0x10, '1', 0x00,
				0x01, 0x00, 0x00, 0x00,
				0x00,
			},
			nil,
		},
		{
			"map[float32]int",
			map[float32]int32{3.14: 1},
			[]byte{
				0x0F, 0x00, 0x00, 0x00,
				0x10, '3', '.', '1', '4', 0x00,
				0x01, 0x00, 0x00, 0x00,
				0x00,
			},
			nil,
		},
		{
			"map[float64]int",
			map[float64]int32{3.14: 1},
			[]byte{
				0x0F, 0x00, 0x00, 0x00,
				0x10, '3', '.', '1', '4', 0x00,
				0x01, 0x00, 0x00, 0x00,
				0x00,
			},
			nil,
		},
		{
			"map[string]int",
			map[string]int32{"foo": 1},
			[]byte{
				0x0E, 0x00, 0x00, 0x00,
				0x10, 'f', 'o', 'o', 0x00,
				0x01, 0x00, 0x00, 0x00,
				0x00,
			},
			nil,
		},
		{
			"map[string]objectid.ObjectID",
			map[string]objectid.ObjectID{"foo": oid},
			docToBytes(NewDocument(EC.ObjectID("foo", oid))),
			nil,
		},
		{
			"map[objectid.ObjectID]string",
			map[objectid.ObjectID]string{oid: "foo"},
			docToBytes(NewDocument(EC.String(oid.String(), "foo"))),
			nil,
		},
		{
			"map[string]*string",
			map[string]*string{"foo": str},
			docToBytes(NewDocument(EC.String("foo", "bar"))),
			nil,
		},
		{
			"map[string]*string with nil",
			map[string]*string{"baz": nil},
			docToBytes(NewDocument(EC.Null("baz"))),
			nil,
		},
		{
			"map[string]_Interface",
			map[string]_Interface{"foo": _impl{Foo: "bar"}},
			docToBytes(NewDocument(EC.SubDocumentFromElements("foo", EC.String("foo", "bar")))),
			nil,
		},
		{
			"map[string]_Interface with nil",
			map[string]_Interface{"baz": (*_impl)(nil)},
			docToBytes(NewDocument(EC.Null("baz"))),
			nil,
		},
		{
			"map[json.Number]json.Number(int64)",
			map[json.Number]json.Number{
				json.Number("5"): json.Number("10"),
			},
			docToBytes(NewDocument(EC.Int64("5", 10))),
			nil,
		},
		{
			"map[json.Number]json.Number(float64)",
			map[json.Number]json.Number{
				json.Number("5.0"): json.Number("10.1"),
			},
			docToBytes(NewDocument(EC.Double("5.0", 10.1))),
			nil,
		},
		{
			"map[*url.URL]*url.URL",
			map[*url.URL]*url.URL{
				murl: murl,
			},
			docToBytes(NewDocument(EC.String(murl.String(), murl.String()))),
			nil,
		},
		{
			"map[decimal.Decimal128]decimal.Decimal128",
			map[decimal.Decimal128]decimal.Decimal128{
				decimal128: decimal128,
			},
			docToBytes(NewDocument(EC.Decimal128(decimal128.String(), decimal128))),
			nil,
		},
		{
			"[]string",
			[]string{"foo", "bar", "baz"},
			[]byte{
				0x26, 0x00, 0x00, 0x00,
				0x02, '0', 0x00,
				0x04, 0x00, 0x00, 0x00,
				'f', 'o', 'o', 0x00,
				0x02, '1', 0x00,
				0x04, 0x00, 0x00, 0x00,
				'b', 'a', 'r', 0x00,
				0x02, '2', 0x00,
				0x04, 0x00, 0x00, 0x00,
				'b', 'a', 'z', 0x00,
				0x00,
			},
			nil,
		},
		{
			"[]*Element",
			[]*Element{EC.Null("A"), EC.Null("B"), EC.Null("C")},
			[]byte{
				0x0E, 0x00, 0x00, 0x00,
				0x0A, 'A', 0x00,
				0x0A, 'B', 0x00,
				0x0A, 'C', 0x00,
				0x00,
			},
			nil,
		},
		{
			"[]*Document",
			[]*Document{NewDocument(EC.Null("A"))},
			docToBytes(NewDocument(
				EC.SubDocumentFromElements("0", (EC.Null("A"))),
			)),
			nil,
		},
		{
			"[]Reader",
			[]Reader{{0x05, 0x00, 0x00, 0x00, 0x00}},
			docToBytes(NewDocument(
				EC.SubDocumentFromElements("0"),
			)),
			nil,
		},
		{
			"[]objectid.ObjectID",
			oids,
			arrToBytes(NewArray(
				VC.ObjectID(oids[0]),
				VC.ObjectID(oids[1]),
				VC.ObjectID(oids[2]),
			)),
			nil,
		},
		{
			"[]*string with nil",
			[]*string{str, nil},
			arrToBytes(NewArray(
				VC.String(*str),
				VC.Null(),
			)),
			nil,
		},
		{
			"[]_Interface with nil",
			[]_Interface{_impl{Foo: "bar"}, (*_impl)(nil), nil},
			arrToBytes(NewArray(
				VC.DocumentFromElements(EC.String("foo", "bar")),
				VC.Null(),
				VC.Null(),
			)),
			nil,
		},
		{
			"[]json.Number",
			[]json.Number{"5", "10.1"},
			arrToBytes(NewArray(
				VC.Int64(5),
				VC.Double(10.1),
			)),
			nil,
		},
		{
			"[]*url.URL",
			[]*url.URL{murl},
			arrToBytes(NewArray(
				VC.String(murl.String()),
			)),
			nil,
		},
		{
			"[]decimal.Decimal128",
			[]decimal.Decimal128{decimal128},
			arrToBytes(NewArray(
				VC.Decimal128(decimal128),
			)),
			nil,
		},
		{
			"map[string][]*Element",
			map[string][]*Element{"Z": {EC.Int32("A", 1), EC.Int32("B", 2), EC.Int32("EC", 3)}},
			docToBytes(NewDocument(
				EC.ArrayFromElements("Z", VC.Int32(1), VC.Int32(2), VC.Int32(3)),
			)),
			nil,
		},
		{
			"map[string][]*Value",
			map[string][]*Value{"Z": {VC.Int32(1), VC.Int32(2), VC.Int32(3)}},
			docToBytes(NewDocument(
				EC.ArrayFromElements("Z", VC.Int32(1), VC.Int32(2), VC.Int32(3)),
			)),
			nil,
		},
		{
			"map[string]*Element",
			map[string]*Element{"Z": EC.Int32("foo", 12345)},
			docToBytes(NewDocument(
				EC.Int32("foo", 12345),
			)),
			nil,
		},
		{
			"map[string]*Document",
			map[string]*Document{"Z": NewDocument(EC.Null("foo"))},
			docToBytes(NewDocument(
				EC.SubDocumentFromElements("Z", EC.Null("foo")),
			)),
			nil,
		},
		{
			"map[string]Reader",
			map[string]Reader{"Z": {0x05, 0x00, 0x00, 0x00, 0x00}},
			docToBytes(NewDocument(
				EC.SubDocumentFromReader("Z", Reader{0x05, 0x00, 0x00, 0x00, 0x00}),
			)),
			nil,
		},
		{
			"map[string][]int32",
			map[string][]int32{"Z": {1, 2, 3}},
			docToBytes(NewDocument(
				EC.ArrayFromElements("Z", VC.Int32(1), VC.Int32(2), VC.Int32(3)),
			)),
			nil,
		},
		{
			"map[string][]objectid.ObjectID",
			map[string][]objectid.ObjectID{"Z": oids},
			docToBytes(NewDocument(
				EC.ArrayFromElements("Z", VC.ObjectID(oids[0]), VC.ObjectID(oids[1]), VC.ObjectID(oids[2])),
			)),
			nil,
		},
		{
			"map[string][]*string with nil",
			map[string][]*string{"Z": {str, nil}},
			docToBytes(NewDocument(
				EC.ArrayFromElements("Z", VC.String("bar"), VC.Null()),
			)),
			nil,
		},
		{
			"map[string][]_Interface with nil",
			map[string][]_Interface{"Z": {_impl{Foo: "bar"}, (*_impl)(nil), nil}},
			docToBytes(NewDocument(
				EC.ArrayFromElements("Z", VC.DocumentFromElements(EC.String("foo", "bar")), VC.Null(), VC.Null()),
			)),
			nil,
		},
		{
			"map[string][]json.Number(int64)",
			map[string][]json.Number{"Z": {json.Number("5"), json.Number("10")}},
			docToBytes(NewDocument(
				EC.ArrayFromElements("Z", VC.Int64(5), VC.Int64(10)),
			)),
			nil,
		},
		{
			"map[string][]json.Number(float64)",
			map[string][]json.Number{"Z": {json.Number("5.0"), json.Number("10.10")}},
			docToBytes(NewDocument(
				EC.ArrayFromElements("Z", VC.Double(5.0), VC.Double(10.10)),
			)),
			nil,
		},
		{
			"map[string][]*url.URL",
			map[string][]*url.URL{"Z": {murl}},
			docToBytes(NewDocument(
				EC.ArrayFromElements("Z", VC.String(murl.String())),
			)),
			nil,
		},
		{
			"map[string][]decimal.Decimal128",
			map[string][]decimal.Decimal128{"Z": {decimal128}},
			docToBytes(NewDocument(
				EC.ArrayFromElements("Z", VC.Decimal128(decimal128)),
			)),
			nil,
		},
		{
			"[2]*Element",
			[2]*Element{EC.Int32("A", 1), EC.Int32("B", 2)},
			docToBytes(NewDocument(
				EC.Int32("A", 1), EC.Int32("B", 2),
			)),
			nil,
		},
		{
			"-",
			struct {
				A string `bson:"-"`
			}{
				A: "",
			},
			docToBytes(NewDocument()),
			nil,
		},
		{
			"omitempty",
			struct {
				A string `bson:",omitempty"`
			}{
				A: "",
			},
			docToBytes(NewDocument()),
			nil,
		},
		{
			"omitempty, empty time",
			struct {
				A time.Time `bson:",omitempty"`
			}{
				A: time.Time{},
			},
			docToBytes(NewDocument()),
			nil,
		},
		{
			"no private fields",
			struct {
				a string
			}{
				a: "should be empty",
			},
			docToBytes(NewDocument()),
			nil,
		},
		{
			"minsize",
			struct {
				A int64 `bson:",minsize"`
			}{
				A: 12345,
			},
			docToBytes(NewDocument(EC.Int32("a", 12345))),
			nil,
		},
		{
			"inline",
			struct {
				Foo struct {
					A int64 `bson:",minsize"`
				} `bson:",inline"`
			}{
				Foo: struct {
					A int64 `bson:",minsize"`
				}{
					A: 12345,
				},
			},
			docToBytes(NewDocument(EC.Int32("a", 12345))),
			nil,
		},
		{
			"inline map",
			struct {
				Foo map[string]string `bson:",inline"`
			}{
				Foo: map[string]string{"foo": "bar"},
			},
			docToBytes(NewDocument(EC.String("foo", "bar"))),
			nil,
		},
		{
			"alternate name bson:name",
			struct {
				A string `bson:"foo"`
			}{
				A: "bar",
			},
			docToBytes(NewDocument(EC.String("foo", "bar"))),
			nil,
		},
		{
			"alternate name",
			struct {
				A string `foo`
			}{
				A: "bar",
			},
			docToBytes(NewDocument(EC.String("foo", "bar"))),
			nil,
		},
		{
			"inline, omitempty",
			struct {
				A   string
				Foo zeroTest `bson:"omitempty,inline"`
			}{
				A:   "bar",
				Foo: zeroTest{true},
			},
			docToBytes(NewDocument(EC.String("a", "bar"))),
			nil,
		},
		{
			"struct{}",
			struct {
				A bool
				B int32
				C int64
				D uint16
				E uint64
				F float64
				G string
				H map[string]string
				I []byte
				J [4]byte
				K [2]string
				L struct {
					M string
				}
				N  *Element
				O  *Document
				P  Reader
				Q  objectid.ObjectID
				R  *string
				S  map[struct{}]struct{}
				T  []struct{}
				U  _Interface
				V  _Interface
				W  map[struct{}]struct{}
				X  map[struct{}]struct{}
				Y  json.Number
				Z  time.Time
				AA json.Number
				AB *url.URL
				AC decimal.Decimal128
				AD *time.Time
			}{
				A: true,
				B: 123,
				C: 456,
				D: 789,
				E: 101112,
				F: 3.14159,
				G: "Hello, world",
				H: map[string]string{"foo": "bar"},
				I: []byte{0x01, 0x02, 0x03},
				J: [4]byte{0x04, 0x05, 0x06, 0x07},
				K: [2]string{"baz", "qux"},
				L: struct {
					M string
				}{
					M: "foobar",
				},
				N:  EC.Null("N"),
				O:  NewDocument(EC.Int64("countdown", 9876543210)),
				P:  Reader{0x05, 0x00, 0x00, 0x00, 0x00},
				Q:  oid,
				R:  nil,
				S:  nil,
				T:  nil,
				U:  nil,
				V:  _Interface((*_impl)(nil)), // typed nil
				W:  map[struct{}]struct{}{},
				X:  nil,
				Y:  json.Number("5"),
				Z:  now,
				AA: json.Number("10.10"),
				AB: murl,
				AC: decimal128,
				AD: &now,
			},
			docToBytes(NewDocument(
				EC.Boolean("a", true),
				EC.Int32("b", 123),
				EC.Int64("c", 456),
				EC.Int32("d", 789),
				EC.Int64("e", 101112),
				EC.Double("f", 3.14159),
				EC.String("g", "Hello, world"),
				EC.SubDocumentFromElements("h", EC.String("foo", "bar")),
				EC.Binary("i", []byte{0x01, 0x02, 0x03}),
				EC.Binary("j", []byte{0x04, 0x05, 0x06, 0x07}),
				EC.ArrayFromElements("k", VC.String("baz"), VC.String("qux")),
				EC.SubDocumentFromElements("l", EC.String("m", "foobar")),
				EC.Null("N"),
				EC.SubDocumentFromElements("o", EC.Int64("countdown", 9876543210)),
				EC.SubDocumentFromElements("p"),
				EC.ObjectID("q", oid),
				EC.Null("r"),
				EC.Null("s"),
				EC.Null("t"),
				EC.Null("u"),
				EC.Null("v"),
				EC.SubDocument("w", NewDocument()),
				EC.Null("x"),
				EC.Int64("y", 5),
				EC.DateTime("z", now.UnixNano()/int64(time.Millisecond)),
				EC.Double("aa", 10.10),
				EC.String("ab", murl.String()),
				EC.Decimal128("ac", decimal128),
				EC.DateTime("ad", now.UnixNano()/int64(time.Millisecond)),
			)),
			nil,
		},
		{
			"struct{[]interface{}}",
			struct {
				A []bool
				B []int32
				C []int64
				D []uint16
				E []uint64
				F []float64
				G []string
				H []map[string]string
				I [][]byte
				J [1][4]byte
				K [1][2]string
				L []struct {
					M string
				}
				N  [][]string
				O  []*Element
				P  []*Document
				Q  []Reader
				R  []objectid.ObjectID
				S  []*string
				T  []struct{}
				U  []_Interface
				V  []_Interface
				W  []map[struct{}]struct{}
				X  []map[struct{}]struct{}
				Y  []map[struct{}]struct{}
				Z  []time.Time
				AA []json.Number
				AB []*url.URL
				AC []decimal.Decimal128
				AD []*time.Time
			}{
				A: []bool{true},
				B: []int32{123},
				C: []int64{456},
				D: []uint16{789},
				E: []uint64{101112},
				F: []float64{3.14159},
				G: []string{"Hello, world"},
				H: []map[string]string{{"foo": "bar"}},
				I: [][]byte{{0x01, 0x02, 0x03}},
				J: [1][4]byte{{0x04, 0x05, 0x06, 0x07}},
				K: [1][2]string{{"baz", "qux"}},
				L: []struct {
					M string
				}{
					{
						M: "foobar",
					},
				},
				N:  [][]string{{"foo", "bar"}},
				O:  []*Element{EC.Null("N")},
				P:  []*Document{NewDocument(EC.Int64("countdown", 9876543210))},
				Q:  []Reader{{0x05, 0x00, 0x00, 0x00, 0x00}},
				R:  oids,
				S:  []*string{str, nil},
				T:  nil,
				U:  nil,
				V:  []_Interface{_impl{Foo: "bar"}, nil, (*_impl)(nil)},
				W:  nil,
				X:  []map[struct{}]struct{}{},   // Should be empty BSON Array
				Y:  []map[struct{}]struct{}{{}}, // Should be BSON array with one element, an empty BSON SubDocument
				Z:  []time.Time{now, now},
				AA: []json.Number{json.Number("5"), json.Number("10.10")},
				AB: []*url.URL{murl},
				AC: []decimal.Decimal128{decimal128},
				AD: []*time.Time{&now, &now},
			},
			docToBytes(NewDocument(
				EC.ArrayFromElements("a", VC.Boolean(true)),
				EC.ArrayFromElements("b", VC.Int32(123)),
				EC.ArrayFromElements("c", VC.Int64(456)),
				EC.ArrayFromElements("d", VC.Int32(789)),
				EC.ArrayFromElements("e", VC.Int64(101112)),
				EC.ArrayFromElements("f", VC.Double(3.14159)),
				EC.ArrayFromElements("g", VC.String("Hello, world")),
				EC.ArrayFromElements("h", VC.DocumentFromElements(EC.String("foo", "bar"))),
				EC.ArrayFromElements("i", VC.Binary([]byte{0x01, 0x02, 0x03})),
				EC.ArrayFromElements("j", VC.Binary([]byte{0x04, 0x05, 0x06, 0x07})),
				EC.ArrayFromElements("k", VC.ArrayFromValues(VC.String("baz"), VC.String("qux"))),
				EC.ArrayFromElements("l", VC.DocumentFromElements(EC.String("m", "foobar"))),
				EC.ArrayFromElements("n", VC.ArrayFromValues(VC.String("foo"), VC.String("bar"))),
				EC.ArrayFromElements("o", VC.Null()),
				EC.ArrayFromElements("p", VC.DocumentFromElements(EC.Int64("countdown", 9876543210))),
				EC.ArrayFromElements("q", VC.DocumentFromElements()),
				EC.ArrayFromElements("r", VC.ObjectID(oids[0]), VC.ObjectID(oids[1]), VC.ObjectID(oids[2])),
				EC.ArrayFromElements("s", VC.String("bar"), VC.Null()),
				EC.Null("t"),
				EC.Null("u"),
				EC.ArrayFromElements("v", VC.DocumentFromElements(EC.String("foo", "bar")), VC.Null(), VC.Null()),
				EC.Null("w"),
				EC.Array("x", NewArray()),
				EC.ArrayFromElements("y", VC.Document(NewDocument())),
				EC.ArrayFromElements("z", VC.DateTime(now.UnixNano()/int64(time.Millisecond)), VC.DateTime(now.UnixNano()/int64(time.Millisecond))),
				EC.ArrayFromElements("aa", VC.Int64(5), VC.Double(10.10)),
				EC.ArrayFromElements("ab", VC.String(murl.String())),
				EC.ArrayFromElements("ac", VC.Decimal128(decimal128)),
				EC.ArrayFromElements("ad", VC.DateTime(now.UnixNano()/int64(time.Millisecond)), VC.DateTime(now.UnixNano()/int64(time.Millisecond))),
			)),
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := NewEncoder(&buf)
			err := enc.Encode(tc.value)
			if err != tc.err {
				t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
			}
			b := buf.Bytes()
			if diff := cmp.Diff(b, tc.b); diff != "" {
				t.Errorf("Bytes written differ: (-got +want)\n%s", diff)
				t.Errorf("Bytes\ngot: %v\nwant:%v\n", b, tc.b)
				t.Errorf("Readers\ngot: %v\nwant:%v\n", Reader(b), Reader(tc.b))
			}
		})
	}
}

type zeroTest struct {
	reportZero bool
}

func (z zeroTest) IsZero() bool { return z.reportZero }

type nonZeroer struct {
	value bool
}

func TestZeoerInterfaceUsedByDecoder(t *testing.T) {
	enc := &encoder{}

	// cases that are zero, because they are known types or pointers
	var st *nonZeroer
	assert.True(t, enc.isZero(reflect.ValueOf(st)))
	assert.True(t, enc.isZero(reflect.ValueOf(0)))
	assert.True(t, enc.isZero(reflect.ValueOf(false)))

	// cases that shouldn't be zero
	st = &nonZeroer{value: false}
	assert.False(t, enc.isZero(reflect.ValueOf(struct{ val bool }{val: true})))
	assert.False(t, enc.isZero(reflect.ValueOf(struct{ val bool }{val: false})))
	assert.False(t, enc.isZero(reflect.ValueOf(st)))
	st.value = true
	assert.False(t, enc.isZero(reflect.ValueOf(st)))

	// a test to see if the interface impacts the outcome
	z := zeroTest{}
	assert.False(t, enc.isZero(reflect.ValueOf(z)))

	z.reportZero = true
	assert.True(t, enc.isZero(reflect.ValueOf(z)))

}

type timePrtStruct struct{ TimePtrField *time.Time }

func TestRegressionNoDereferenceNilTimePtr(t *testing.T) {
	enc := &encoder{}

	assert.NotPanics(t, func() {
		res, err := enc.encodeStruct(reflect.ValueOf(timePrtStruct{}))
		assert.Len(t, res, 1)
		assert.Nil(t, err)
	})

	assert.NotPanics(t, func() {
		res, err := enc.encodeSliceAsArray(reflect.ValueOf([]*time.Time{nil, nil, nil}), false)
		assert.Len(t, res, 3)
		assert.Nil(t, err)
	})
}

func docToBytes(d *Document) []byte {
	b, err := d.MarshalBSON()
	if err != nil {
		panic(err)
	}
	return b
}

func arrToBytes(a *Array) []byte {
	b, err := a.MarshalBSON()
	if err != nil {
		panic(err)
	}
	return b
}

type byteMarshaler []byte

func (bm byteMarshaler) MarshalBSON() ([]byte, error) { return bm, nil }

type _Interface interface {
	method()
}

type _impl struct {
	Foo string
}

func (_impl) method() {}
