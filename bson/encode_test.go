// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
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
				N *Element
				O *Document
				P Reader
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
				N: EC.Null("N"),
				O: NewDocument(EC.Int64("countdown", 9876543210)),
				P: Reader{0x05, 0x00, 0x00, 0x00, 0x00},
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
				N [][]string
				O []*Element
				P []*Document
				Q []Reader
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
				N: [][]string{{"foo", "bar"}},
				O: []*Element{EC.Null("N")},
				P: []*Document{NewDocument(EC.Int64("countdown", 9876543210))},
				Q: []Reader{{0x05, 0x00, 0x00, 0x00, 0x00}},
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
			}
		})
	}
}

func docToBytes(d *Document) []byte {
	b, err := d.MarshalBSON()
	if err != nil {
		panic(err)
	}
	return b
}

type byteMarshaler []byte

func (bm byteMarshaler) MarshalBSON() ([]byte, error) { return bm, nil }
