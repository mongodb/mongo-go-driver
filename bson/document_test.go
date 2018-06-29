// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestDocument(t *testing.T) {
	t.Run("NewDocument", func(t *testing.T) {
		t.Run("TooShort", func(t *testing.T) {
			want := NewErrTooSmall()
			_, got := ReadDocument([]byte{'\x00', '\x00'})
			if !want.Equals(got) {
				t.Errorf("Did not get expected error. got %#v; want %#v", got, want)
			}
		})
		t.Run("InvalidLength", func(t *testing.T) {
			want := ErrInvalidLength
			b := make([]byte, 5)
			binary.LittleEndian.PutUint32(b[0:4], 200)
			_, got := ReadDocument(b)
			if got != want {
				t.Errorf("Did not get expected error. got %#v; want %#v", got, want)
			}
		})
		t.Run("keyLength-error", func(t *testing.T) {
			want := ErrInvalidKey
			b := make([]byte, 8)
			binary.LittleEndian.PutUint32(b[0:4], 8)
			b[4], b[5], b[6], b[7] = '\x02', 'f', 'o', 'o'
			_, got := ReadDocument(b)
			if got != want {
				t.Errorf("Did not get expected error. got %#v; want %#v", got, want)
			}
		})
		t.Run("Missing-Null-Terminator", func(t *testing.T) {
			want := ErrInvalidReadOnlyDocument
			b := make([]byte, 9)
			binary.LittleEndian.PutUint32(b[0:4], 9)
			b[4], b[5], b[6], b[7], b[8] = '\x0A', 'f', 'o', 'o', '\x00'
			_, got := ReadDocument(b)
			if got != want {
				t.Errorf("Did not get expected error. got %#v; want %#v", got, want)
			}
		})
		t.Run("validateValue-error", func(t *testing.T) {
			want := NewErrTooSmall()
			b := make([]byte, 11)
			binary.LittleEndian.PutUint32(b[0:4], 11)
			b[4], b[5], b[6], b[7], b[8], b[9], b[10] = '\x01', 'f', 'o', 'o', '\x00', '\x01', '\x02'
			_, got := ReadDocument(b)
			if !want.Equals(got) {
				t.Errorf("Did not get expected error. got %#v; want %#v", got, want)
			}
		})
		testCases := []struct {
			name string
			b    []byte
			want *Document
			err  error
		}{
			{"empty document", []byte{'\x05', '\x00', '\x00', '\x00', '\x00'}, &Document{}, nil},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := ReadDocument(tc.b)
				if err != tc.err {
					t.Errorf("Did not get expected error. got %#v; want %#v", err, tc.err)
				}
				if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(Document{})); diff != "" {
					t.Errorf("Documents differ: (-got +want)\n%s", diff)
				}
			})
		}
	})
	t.Run("Keys", testDocumentKeys)
	t.Run("Append", func(t *testing.T) {
		t.Run("Nil Insert", func(t *testing.T) {
			func() {
				defer func() {
					r := recover()
					if r != ErrNilElement {
						t.Errorf("Did not received expected error from panic. got %#v; want %#v", r, ErrNilElement)
					}
				}()
				d := NewDocument()
				d.Append(nil)
			}()
		})
		t.Run("Ignore Nil Insert", func(t *testing.T) {
			func() {
				defer func() {
					r := recover()
					if r != nil {
						t.Errorf("Received unexpected panic from nil insert. got %#v; want %#v", r, nil)
					}
				}()
				want := NewDocument()
				want.IgnoreNilInsert = true

				got := NewDocument()
				got.IgnoreNilInsert = true
				got.Append(nil)
				if diff := cmp.Diff(got, want, cmp.AllowUnexported(Document{})); diff != "" {
					t.Errorf("Documents differ: (-got +want)\n%s", diff)
				}
			}()
		})
		testCases := []struct {
			name  string
			elems [][]*Element
			want  []byte
		}{
			{"one-one", tpag.oneOne(), tpag.oneOneAppendBytes()},
			{"two-one", tpag.twoOne(), tpag.twoOneAppendBytes()},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				d := NewDocument()
				for _, elems := range tc.elems {
					d.Append(elems...)
				}
				got, err := d.MarshalBSON()
				if err != nil {
					t.Errorf("Received an unexpected error while marhsaling BSON: %s", err)
				}
				if !bytes.Equal(got, tc.want) {
					t.Errorf("Output from Append is not correct. got %#v; want %#v", got, tc.want)
				}
			})
		}
	})
	t.Run("Prepend", func(t *testing.T) {
		t.Run("Nil Insert", func(t *testing.T) {
			testCases := []struct {
				name  string
				elems []*Element
				want  *Document
			}{
				{"first element nil", []*Element{nil}, &Document{elems: make([]*Element, 0), index: make([]uint32, 0)}},
			}

			for _, tc := range testCases {
				var got *Document
				func() {
					defer func() {
						r := recover()
						if r != ErrNilElement {
							t.Errorf("Did not received expected error from panic. got %#v; want %#v", r, ErrNilElement)
						}
						if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(Document{})); diff != "" {
							t.Errorf("Documents differ: (-got +want)\n%s", diff)
						}
					}()
					got = NewDocument()
					got.Prepend(tc.elems...)
				}()
			}
		})
		t.Run("Ignore Nil Insert", func(t *testing.T) {
			testCases := []struct {
				name  string
				elems []*Element
				want  *Document
			}{
				{"first element nil", []*Element{nil},
					&Document{
						IgnoreNilInsert: true,
						elems:           make([]*Element, 0), index: make([]uint32, 0)},
				},
			}

			for _, tc := range testCases {
				var got *Document
				func() {
					defer func() {
						r := recover()
						if r != nil {
							t.Errorf("Did not received expected error from panic. got %#v; want %#v", r, nil)
						}
						if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(Document{})); diff != "" {
							t.Errorf("Documents differ: (-got +want)\n%s", diff)
						}
					}()
					got = NewDocument()
					got.IgnoreNilInsert = true
					got.Prepend(tc.elems...)
				}()
			}
		})
		t.Run("Update Index Properly", func(t *testing.T) {
			a, b, c, d, e := EC.Null("a"), EC.Null("b"), EC.Null("c"), EC.Null("d"), EC.Null("e")
			testCases := []struct {
				name  string
				elems [][]*Element
				index [][]uint32
			}{
				{
					"two",
					[][]*Element{{a}, {b}},
					[][]uint32{{0}, {1, 0}},
				},
				{
					"three",
					[][]*Element{{a}, {b}, {c}},
					[][]uint32{{0}, {1, 0}, {2, 1, 0}},
				},
				{
					"four",
					[][]*Element{{a}, {d}, {b}, {c}},
					[][]uint32{{0}, {1, 0}, {2, 0, 1}, {3, 1, 0, 2}},
				},
				{
					"five",
					[][]*Element{{a}, {d}, {e}, {b}, {c}},
					[][]uint32{{0}, {1, 0}, {2, 1, 0}, {3, 0, 2, 1}, {4, 1, 0, 3, 2}},
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					d := NewDocument()
					for idx := range tc.elems {
						d.Prepend(tc.elems[idx]...)
						index := d.index
						for jdx := range tc.index[idx] {
							if tc.index[idx][jdx] != index[jdx] {
								t.Errorf(
									"Indexes do not match at %d: got %v; want %v",
									idx, index, tc.index[idx],
								)
								break
							}
						}
					}
				})
			}
		})
		testCases := []struct {
			name  string
			elems [][]*Element
			want  []byte
		}{
			{"one-one", tpag.oneOne(), tpag.oneOnePrependBytes()},
			{"two-one", tpag.twoOne(), tpag.twoOnePrependBytes()},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				d := NewDocument()
				for _, elems := range tc.elems {
					d.Prepend(elems...)
				}
				got, err := d.MarshalBSON()
				if err != nil {
					t.Errorf("Received an unexpected error while marhsaling BSON: %s", err)
				}
				if !bytes.Equal(got, tc.want) {
					t.Errorf("Output from Prepend is not correct. got %#v; want %#v", got, tc.want)
				}
			})
		}
	})
	t.Run("Set", func(t *testing.T) {
		t.Run("Nil Insert", func(t *testing.T) {
			testCases := []struct {
				name string
				elem *Element
				want *Document
			}{
				{
					"first element nil",
					nil,
					&Document{elems: make([]*Element, 0),
						index: make([]uint32, 0)}},
			}

			for _, tc := range testCases {
				var got *Document
				func() {
					defer func() {
						r := recover()
						if r != ErrNilElement {
							t.Errorf("Did not receive expected error from panic. got %#v; want %#v", r, ErrNilElement)
						}
						if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(Document{})); diff != "" {
							t.Errorf("Documents differ: (-got +want)\n%s", diff)
						}
					}()
					got = NewDocument()
					got.Set(tc.elem)
				}()
			}
		})
		t.Run("Ignore Nil Insert", func(t *testing.T) {
			testCases := []struct {
				name string
				elem *Element
				want *Document
			}{
				{"first element nil", nil,
					&Document{
						IgnoreNilInsert: true,
						elems:           make([]*Element, 0), index: make([]uint32, 0)},
				},
			}

			for _, tc := range testCases {
				var got *Document
				func() {
					defer func() {
						r := recover()
						if r != nil {
							t.Errorf("Did not received expected error from panic. got %#v; want %#v", r, nil)
						}
						if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(Document{})); diff != "" {
							t.Errorf("Documents differ: (-got +want)\n%s", diff)
						}
					}()
					got = NewDocument()
					got.IgnoreNilInsert = true
					got.Set(tc.elem)
				}()
			}
		})
		testCases := []struct {
			name string
			d    *Document
			elem *Element
			want *Document
		}{
			{
				"first",
				(&Document{}).Append(EC.Double("x", 3.14)),
				EC.Double("x", 3.14159),
				(&Document{}).Append(EC.Double("x", 3.14159)),
			},
			{"second", (&Document{}).Append(EC.Double("x", 3.14159), EC.String("y", "z")),
				EC.Double("y", 1.2345),
				(&Document{}).Append(EC.Double("x", 3.14159), EC.Double("y", 1.2345)),
			},
			{"concat", (&Document{}).Append(EC.Null("x")),
				EC.Null("y"),
				(&Document{}).Append(EC.Null("x"), EC.Null("y")),
			},
			{"concat-in-middle", (&Document{}).Append(EC.Null("w"), EC.Null("y"), EC.Null("z")),
				EC.Null("x"),
				(&Document{}).Append(EC.Null("w"), EC.Null("y"), EC.Null("z"), EC.Null("x")),
			},
			{
				"update-element-not-lexicographically-sorted",
				NewDocument(EC.Int32("b", 1), EC.Int32("a", 2), EC.Int32("d", 3), EC.Int32("c", 4)),
				EC.Int32("d", 5),
				NewDocument(EC.Int32("b", 1), EC.Int32("a", 2), EC.Int32("d", 5), EC.Int32("c", 4)),
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				got := tc.d.Set(tc.elem)
				if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(Document{}, Element{}, Value{})); diff != "" {
					t.Errorf("Documents differ: (-got +want)\n%s", diff)
				}
			})
		}
	})
	t.Run("Lookup", func(t *testing.T) {
		t.Run("empty key", func(t *testing.T) {
			d := NewDocument()
			_, err := d.LookupErr()
			if err != ErrEmptyKey {
				t.Errorf("Empty key lookup did not return expected result. got %#v; want %#v", err, ErrEmptyKey)
			}
		})
		testCases := []struct {
			name string
			d    *Document
			key  []string
			want *Element
			err  error
		}{
			{"first", (&Document{}).Append(EC.Null("x")), []string{"x"},
				&Element{&Value{start: 0, offset: 3}}, nil,
			},
			{"depth-one", (&Document{}).Append(EC.SubDocumentFromElements("x", EC.Null("y"))),
				[]string{"x", "y"},
				&Element{&Value{start: 0, offset: 3}}, nil,
			},
			{"invalid-depth-traversal", (&Document{}).Append(EC.Null("x")),
				[]string{"x", "y"},
				nil, ErrInvalidDepthTraversal,
			},
			{"not-found", (&Document{}).Append(EC.Null("x")),
				[]string{"y"},
				nil, ErrElementNotFound,
			},
			{"subarray",
				NewDocument(
					EC.ArrayFromElements("foo",
						VC.DocumentFromElements(
							EC.Int32("bar", 12),
						),
					),
				),
				[]string{"foo", "0", "bar"},
				EC.Int32("bar", 12),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := tc.d.LookupElementErr(tc.key...)
				if err != tc.err {
					t.Errorf("Returned error does not match. got %#v; want %#v", err, tc.err)
				}
				if !elementEqual(got, tc.want) {
					t.Errorf("Returned element does not match expected element. got %#v; want %#v", got, tc.want)
				}
			})
		}
	})
	t.Run("Delete", func(t *testing.T) {
		t.Run("empty key", func(t *testing.T) {
			d := NewDocument()
			var want *Element
			got := d.Delete()
			if got != want {
				t.Errorf("Delete should return nil element when deleting with empty key. got %#v; want %#v", got, want)
			}
		})
		d, c, b, a := EC.Null("d"), EC.Null("c"), EC.Null("b"), EC.Null("a")
		testCases := []struct {
			name string
			d    *Document
			keys [][]string
			want []*Element
		}{
			{"first", (&Document{}).Append(EC.Null("x")), [][]string{{"x"}},
				[]*Element{{&Value{start: 0, offset: 3}}},
			},
			{"depth-one", (&Document{}).Append(EC.SubDocumentFromElements("x", EC.Null("y"))),
				[][]string{{"x", "y"}},
				[]*Element{{&Value{start: 0, offset: 3}}},
			},
			{"invalid-depth-traversal", (&Document{}).Append(EC.Null("x")),
				[][]string{{"x", "y"}},
				[]*Element{nil},
			},
			{"not-found", (&Document{}).Append(EC.Null("x")),
				[][]string{{"y"}},
				[]*Element{nil},
			},
			{
				"delete twice",
				NewDocument(d, c, b, a),
				[][]string{{"d"}, {"c"}},
				[]*Element{d, c},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				for idx := range tc.keys {
					got := tc.d.Delete(tc.keys[idx]...)
					if !elementEqual(got, tc.want[idx]) {
						t.Errorf("Returned element does not match expected element. got %#v; want %#v", got, tc.want[idx])
					}
				}
			})
		}
	})
	t.Run("ElementAtOK", func(t *testing.T) {
		t.Run("Out of bounds", func(t *testing.T) {
			d := NewDocument(EC.Null("x"), EC.Null("y"), EC.Null("z"))
			_, ok := d.ElementAtOK(3)
			if ok {
				t.Errorf("ok=false should be returned when accessing element beyond end of document. got %#v; want %#v", ok, false)
			}
		})
		testCases := []struct {
			name  string
			elems []*Element
			index uint
			want  *Element
		}{
			{"first", []*Element{EC.Null("x"), EC.Null("y"), EC.Null("z")}, 0, EC.Null("x")},
			{"second", []*Element{EC.Null("x"), EC.Null("y"), EC.Null("z")}, 1, EC.Null("y")},
			{"third", []*Element{EC.Null("x"), EC.Null("y"), EC.Null("z")}, 2, EC.Null("z")},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				d := NewDocument(tc.elems...)
				got, ok := d.ElementAtOK(tc.index)
				if !ok {
					t.Errorf("ElementAtOK returned ok=false when true was expected")
				}
				if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(Element{}, Value{})); diff != "" {
					t.Errorf("Documents differ: (-got +want)\n%s", diff)
				}
			})
		}
	})
	t.Run("Iterator", func(t *testing.T) {
		elems := []*Element{EC.String("foo", "bar"), EC.Int32("baz", 1), EC.Null("bing")}
		d := NewDocument(elems...)

		iter := d.Iterator()

		for _, elem := range elems {
			require.True(t, iter.Next())
			require.NoError(t, iter.Err())
			requireElementsEqual(t, elem, iter.Element())
		}

		require.False(t, iter.Next())
		require.NoError(t, iter.Err())
	})
	t.Run("Concat", func(t *testing.T) {
		testCases := []struct {
			name     string
			doc      *Document
			concat   []interface{}
			expected *Document
			err      error
		}{
			{
				"nil",
				NewDocument(),
				[]interface{}{
					nil,
				},
				nil,
				ErrNilDocument,
			},
			{
				"nil document",
				NewDocument(),
				[]interface{}{
					(*Document)(nil),
				},
				nil,
				ErrNilDocument,
			},
			{
				"concat single doc",
				NewDocument(),
				[]interface{}{
					NewDocument(EC.String("foo", "bar")),
				},
				NewDocument(EC.String("foo", "bar")),
				nil,
			},
			{
				"concat multiple docs",
				NewDocument(),
				[]interface{}{
					NewDocument(EC.String("foo", "bar")),
					NewDocument(EC.Int32("baz", 3), EC.Null("bang")),
				},
				NewDocument(EC.String("foo", "bar"), EC.Int32("baz", 3), EC.Null("bang")),
				nil,
			},
			{
				"concat single byte slice",
				NewDocument(),
				[]interface{}{
					[]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - string
						0x2,
						// key - "foo"
						0x66, 0x6f, 0x6f, 0x0,
						// value - string length
						0x4, 0x0, 0x0, 0x0,
						// value - string "bar"
						0x62, 0x61, 0x72, 0x0,

						// null terminator
						0x0,
					},
				},
				NewDocument(EC.String("foo", "bar")),
				nil,
			},
			{
				"concat multiple byte slices",
				NewDocument(),
				[]interface{}{
					[]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - string
						0x2,
						// key - "foo"
						0x66, 0x6f, 0x6f, 0x0,
						// value - string length
						0x4, 0x0, 0x0, 0x0,
						// value - string "bar"
						0x62, 0x61, 0x72, 0x0,

						// null terminator
						0x0,
					},
					[]byte{
						// length
						0x14, 0x0, 0x0, 0x0,

						// type - string
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(3)
						0x3, 0x0, 0x0, 0x0,

						// type - null
						0xa,
						// key - "bang"
						0x62, 0x61, 0x6e, 0x67, 0x0,

						// null terminator
						0x0,
					},
				},
				NewDocument(EC.String("foo", "bar"), EC.Int32("baz", 3), EC.Null("bang")),
				nil,
			},
			{
				"concat single reader",
				NewDocument(),
				[]interface{}{
					Reader([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - string
						0x2,
						// key - "foo"
						0x66, 0x6f, 0x6f, 0x0,
						// value - string length
						0x4, 0x0, 0x0, 0x0,
						// value - string "bar"
						0x62, 0x61, 0x72, 0x0,

						// null terminator
						0x0,
					}),
				},
				NewDocument(EC.String("foo", "bar")),
				nil,
			},
			{
				"concat multiple readers",
				NewDocument(),
				[]interface{}{
					Reader([]byte{
						// length
						0x12, 0x0, 0x0, 0x0,

						// type - string
						0x2,
						// key - "foo"
						0x66, 0x6f, 0x6f, 0x0,
						// value - string length
						0x4, 0x0, 0x0, 0x0,
						// value - string "bar"
						0x62, 0x61, 0x72, 0x0,

						// null terminator
						0x0,
					}),
					Reader([]byte{
						// length
						0x14, 0x0, 0x0, 0x0,

						// type - string
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(3)
						0x3, 0x0, 0x0, 0x0,

						// type - null
						0xa,
						// key - "bang"
						0x62, 0x61, 0x6e, 0x67, 0x0,

						// null terminator
						0x0,
					}),
				},
				NewDocument(EC.String("foo", "bar"), EC.Int32("baz", 3), EC.Null("bang")),
				nil,
			},
			{
				"concat mixed",
				NewDocument(),
				[]interface{}{
					NewDocument(EC.String("foo", "bar")),
					[]byte{
						// length
						0xe, 0x0, 0x0, 0x0,

						// type - string
						0x10,
						// key - "baz"
						0x62, 0x61, 0x7a, 0x0,
						// value - int32(3)
						0x3, 0x0, 0x0, 0x0,

						// null terminator
						0x0,
					},
					Reader([]byte{
						// length
						0xb, 0x0, 0x0, 0x0,

						// type - null
						0xa,
						// key - "bang"
						0x62, 0x61, 0x6e, 0x67, 0x0,

						// null terminator
						0x0,
					}),
				},
				NewDocument(EC.String("foo", "bar"), EC.Int32("baz", 3), EC.Null("bang")),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.doc.Concat(tc.concat...)
				require.Equal(t, tc.err, err)
				if err != nil {
					return
				}

				require.True(t, documentComparer(tc.expected, tc.doc))
			})
		}

	})
	t.Run("Reset", func(t *testing.T) {
		d := NewDocument(EC.Null("a"), EC.Null("b"), EC.Null("c"), EC.Null("a"), EC.Null("e"))
		gotSlc := d.elems
		d.Reset()
		wantSlc := make([]*Element, 5)
		if diff := cmp.Diff(gotSlc, wantSlc, cmp.AllowUnexported(Element{})); diff != "" {
			t.Error("Pointers to elements should be cleared on Reset.")
			t.Errorf("Element slices differ: (-got +want)\n%s", diff)
		}
		if len(d.elems) != 0 {
			t.Errorf("Expected length of elements slice to be 0. got %d; want %d", len(d.elems), 0)
		}
		if len(d.index) != 0 {
			t.Errorf("Expected length of index slice to be 0. got %d; want %d", len(d.elems), 0)
		}
	})
	t.Run("WriteTo", func(t *testing.T) {
		testCases := []struct {
			name string
			d    *Document
			want []byte
			n    int64
			err  error
		}{
			{"empty-document", NewDocument(), []byte{'\x05', '\x00', '\x00', '\x00', '\x00'}, 5, nil},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var buf bytes.Buffer
				n, err := tc.d.WriteTo(&buf)
				if n != tc.n {
					t.Errorf("Number of bytes written does not match. got %d; want %d", n, tc.n)
				}
				if err != tc.err {
					t.Errorf("Returned error does not match expected error. got %s; want %s", err, tc.err)
				}
				if diff := cmp.Diff(buf.Bytes(), tc.want); diff != "" {
					t.Errorf("Written bytes differ: (-got +want)\n%s", diff)
				}
			})
		}
	})
	t.Run("WriteDocument", func(t *testing.T) {
		t.Run("invalid-document", func(t *testing.T) {
			d := NewDocument(EC.Double("", 3.14159))
			d.elems[0].value.data = d.elems[0].value.data[:3]
			b := make([]byte, 15)
			_, err := d.WriteDocument(0, b)
			if !NewErrTooSmall().Equals(err) {
				t.Errorf("Expected error not returned. got %s; want %s", err, NewErrTooSmall())
			}
		})
		t.Run("[]byte-too-small", func(t *testing.T) {
			d := NewDocument(EC.Double("", 3.14159))
			b := make([]byte, 5)
			_, err := d.WriteDocument(0, b)
			if !NewErrTooSmall().Equals(err) {
				t.Errorf("Expected error not returned. got %s; want %s", err, NewErrTooSmall())
			}
		})
		t.Run("invalid-writer", func(t *testing.T) {
			d := NewDocument(EC.Double("", 3.14159))
			var buf bytes.Buffer
			_, err := d.WriteDocument(0, buf)
			if err != ErrInvalidWriter {
				t.Errorf("Expected error not returned. got %s; want %s", err, NewErrTooSmall())
			}
		})

		testCases := []struct {
			name  string
			d     *Document
			start uint
			want  []byte
			n     int64
			err   error
		}{
			{"empty-document", NewDocument(), 0, []byte{'\x05', '\x00', '\x00', '\x00', '\x00'}, 5, nil},
		}

		for _, tc := range testCases {
			b := make([]byte, tc.n)
			n, err := tc.d.WriteDocument(tc.start, b)
			if n != tc.n {
				t.Errorf("Number of bytes written does not match. got %d; want %d", n, tc.n)
			}
			if err != tc.err {
				t.Errorf("Returned error does not match expected error. got %s; want %s", err, tc.err)
			}
			if diff := cmp.Diff(b, tc.want); diff != "" {
				t.Errorf("Written bytes differ: (-got +want)\n%s", diff)
			}
		}
	})
	t.Run("MarshalBSON", func(t *testing.T) {})
	t.Run("writeByteSlice", func(t *testing.T) {})
	t.Run("UnmarshalBSON", func(t *testing.T) {
		testCases := []struct {
			name string
			b    []byte
			want *Document
			err  error
		}{
			{"four",
				[]byte{
					'\x11', '\x00', '\x00', '\x00',
					'\x0A', 'x', '\x00', '\x0A', 'y', '\x00', '\x0A', 'z', '\x00', '\x0A', 'w', '\x00',
					'\x00',
				},
				NewDocument(EC.Null("x"), EC.Null("y"), EC.Null("z"), EC.Null("w")),
				nil,
			},
		}

		for _, tc := range testCases {
			d := NewDocument()
			err := d.UnmarshalBSON(tc.b)
			if err != tc.err {
				t.Errorf("Expected error not returned. got %s; want %s", err, tc.err)
			}
			if diff := cmp.Diff(d, tc.want, cmp.Comparer(documentComparer)); diff != "" {
				t.Errorf("Documents differ: (-got +want)\n%s", diff)
				t.Errorf("\n%#v\n%#v", d, tc.want)
			}

		}
	})
	t.Run("ReadFrom", func(t *testing.T) {
		t.Run("[]byte-too-small", func(t *testing.T) {
			var buf bytes.Buffer
			_, err := NewDocument().ReadFrom(&buf)
			if err != io.EOF {
				t.Errorf("Expected error not returned. got %s; want %s", err, io.EOF)
			}
		})
		t.Run("incorrect-length", func(t *testing.T) {
			var buf bytes.Buffer
			err := binary.Write(&buf, binary.LittleEndian, uint32(10))
			if err != nil {
				t.Errorf("Unexepected error while writing length: %s", err)
			}
			_, err = NewDocument().ReadFrom(&buf)
			if err != io.EOF {
				t.Errorf("Expected error not returned. got %s; want %s", err, io.EOF)
			}
		})
		t.Run("invalid-document", func(t *testing.T) {
			var buf bytes.Buffer
			_, err := (&buf).Write([]byte{'\x07', '\x00', '\x00', '\x00', '\x01', '\x00', '\x00'})
			if err != nil {
				t.Errorf("Unexpected error while writing document to buffer: %s", err)
			}
			_, err = NewDocument().ReadFrom(&buf)
			if !NewErrTooSmall().Equals(err) {
				t.Errorf("Expected error not returned. got %s; want %s", err, NewErrTooSmall())
			}
		})
		testCases := []struct {
			name string
			b    []byte
			want *Document
			n    int64
			err  error
		}{
			{"empty-document", []byte{'\x05', '\x00', '\x00', '\x00', '\x00'}, NewDocument(), 5, nil},
		}

		for _, tc := range testCases {
			var buf bytes.Buffer
			_, err := (&buf).Write(tc.b)
			if err != nil {
				t.Errorf("Unexpected error while writing document to buffer: %s", err)
			}
			d := NewDocument()
			n, err := d.ReadFrom(&buf)
			if n != tc.n {
				t.Errorf("Number of bytes written does not match. got %d; want %d", n, tc.n)
			}
			if err != tc.err {
				t.Errorf("Returned error does not match expected error. got %s; want %s", err, tc.err)
			}
			if diff := cmp.Diff(d, tc.want, cmp.AllowUnexported(Document{}, Element{})); diff != "" {
				t.Errorf("Written bytes differ: (-got +want)\n%s", diff)
			}
		}
	})
	t.Run("ToExtJSON", func(t *testing.T) {
		t.Run("Marshaling Error", func(t *testing.T) {
			var doc *Document
			_, err := doc.ToExtJSONErr(false)
			if err != ErrNilDocument {
				t.Errorf("Did not receive expected error. got %v; want %v", err, ErrNilDocument)
			}
		})
		t.Run("Canonical", func(t *testing.T) {
			doc := NewDocument(EC.String("hello", "world"), EC.Double("pi", 3.14159))
			want := `{"hello":"world","pi":{"$numberDouble":"3.14159"}}`
			got, err := doc.ToExtJSONErr(true)
			if err != nil {
				t.Errorf("Unexpected error while converting document to extended json: %v", err)
			}
			if got != want {
				t.Errorf("Did not receive expected result. got %s; want %s", got, want)
			}
		})
		t.Run("Relaxed", func(t *testing.T) {
			doc := NewDocument(EC.String("hello", "world"), EC.Double("pi", 3.14159))
			want := `{"hello":"world","pi":3.14159}`
			got, err := doc.ToExtJSONErr(false)
			if err != nil {
				t.Errorf("Unexpected error while converting document to extended json: %v", err)
			}
			if got != want {
				t.Errorf("Did not receive expected result. got %s; want %s", got, want)
			}
		})
	})
}

func testDocumentKeys(t *testing.T) {
	testCases := []struct {
		name      string
		d         *Document
		want      Keys
		err       error
		recursive bool
	}{
		{"one", (&Document{}).Append(EC.String("foo", "")), Keys{{Name: "foo"}}, nil, false},
		{"two", (&Document{}).Append(EC.Null("x"), EC.Null("y")), Keys{{Name: "x"}, {Name: "y"}}, nil, false},
		{"one-flat", (&Document{}).Append(EC.SubDocumentFromElements("foo", EC.Null("a"), EC.Null("b"))),
			Keys{{Name: "foo"}}, nil, false,
		},
		{"one-recursive", (&Document{}).Append(EC.SubDocumentFromElements("foo", EC.Null("a"), EC.Null("b"))),
			Keys{{Name: "foo"}, {Prefix: []string{"foo"}, Name: "a"}, {Prefix: []string{"foo"}, Name: "b"}}, nil, true,
		},
		// {"one-array-recursive", (&Document{}).Append(c.ArrayFromElements("foo", VC.Null(())),
		// 	Keys{{Name: "foo"}, {Prefix: []string{"foo"}, Name: "1"}, {Prefix: []string{"foo"}, Name: "2"}}, nil, true,
		// },
		// {"invalid-subdocument",
		// 	Reader{
		// 		'\x15', '\x00', '\x00', '\x00',
		// 		'\x03',
		// 		'f', 'o', 'o', '\x00',
		// 		'\x0B', '\x00', '\x00', '\x00', '\x01', '1', '\x00',
		// 		'\x0A', '2', '\x00', '\x00', '\x00',
		// 	},
		// 	nil, NewErrTooSmall(), true,
		// },
		// {"invalid-array",
		// 	Reader{
		// 		'\x15', '\x00', '\x00', '\x00',
		// 		'\x04',
		// 		'f', 'o', 'o', '\x00',
		// 		'\x0B', '\x00', '\x00', '\x00', '\x01', '1', '\x00',
		// 		'\x0A', '2', '\x00', '\x00', '\x00',
		// 	},
		// 	nil, NewErrTooSmall(), true,
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.d.Keys(tc.recursive)
			if err != tc.err {
				t.Errorf("Returned error does not match. got %#v; want %#v", err, tc.err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Returned keys do not match expected keys. got %#v; want %#v", got, tc.want)
			}
		})
	}
}

var tpag testPrependAppendGenerator

type testPrependAppendGenerator struct{}

func (testPrependAppendGenerator) oneOne() [][]*Element {
	return [][]*Element{
		{EC.Double("foobar", 3.14159)},
	}
}

func (testPrependAppendGenerator) oneOneAppendBytes() []byte {
	return []byte{
		// size
		0x15, 0x0, 0x0, 0x0,
		// type
		0x1,
		// key
		0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72, 0x0,
		// value
		0x6e, 0x86, 0x1b, 0xf0, 0xf9, 0x21, 0x9, 0x40,
		// null terminator
		0x0,
	}
}

func (testPrependAppendGenerator) oneOnePrependBytes() []byte {
	return []byte{
		// size
		0x15, 0x0, 0x0, 0x0,
		// type
		0x1,
		// key
		0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72, 0x0,
		// value
		0x6e, 0x86, 0x1b, 0xf0, 0xf9, 0x21, 0x9, 0x40,
		// null terminator
		0x0,
	}
}

func (testPrependAppendGenerator) twoOne() [][]*Element {
	return [][]*Element{
		{EC.Double("foo", 1.234)},
		{EC.Double("foo", 5.678)},
	}
}

func (testPrependAppendGenerator) twoOneAppendBytes() []byte {
	return []byte{
		// size
		0x1f, 0x0, 0x0, 0x0,
		//type - key - value
		0x1, 0x66, 0x6f, 0x6f, 0x0, 0x58, 0x39, 0xb4, 0xc8, 0x76, 0xbe, 0xf3, 0x3f,
		// type - key - value
		0x1, 0x66, 0x6f, 0x6f, 0x0, 0x83, 0xc0, 0xca, 0xa1, 0x45, 0xb6, 0x16, 0x40,
		// null terminator
		0x0,
	}
}

func (testPrependAppendGenerator) twoOnePrependBytes() []byte {
	return []byte{
		// size
		0x1f, 0x0, 0x0, 0x0,
		// type - key - value
		0x1, 0x66, 0x6f, 0x6f, 0x0, 0x83, 0xc0, 0xca, 0xa1, 0x45, 0xb6, 0x16, 0x40,
		//type - key - value
		0x1, 0x66, 0x6f, 0x6f, 0x0, 0x58, 0x39, 0xb4, 0xc8, 0x76, 0xbe, 0xf3, 0x3f,
		// null terminator
		0x0,
	}
}

func ExampleDocument() {
	internalVersion := "1234567"

	f := func(appName string) *Document {
		doc := NewDocument(
			EC.SubDocumentFromElements("driver",
				EC.String("name", "mongo-go-driver"),
				EC.String("version", internalVersion),
			),
			EC.SubDocumentFromElements("os",
				EC.String("type", "darwin"),
				EC.String("architecture", "amd64"),
			),
			EC.String("platform", "go1.9.2"),
		)
		if appName != "" {
			doc.Append(EC.SubDocumentFromElements("application", EC.String("name", appName)))
		}

		return doc
	}
	buf, err := f("hello-world").MarshalBSON()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(buf)

	// Output: [177 0 0 0 3 100 114 105 118 101 114 0 52 0 0 0 2 110 97 109 101 0 16 0 0 0 109 111 110 103 111 45 103 111 45 100 114 105 118 101 114 0 2 118 101 114 115 105 111 110 0 8 0 0 0 49 50 51 52 53 54 55 0 0 3 111 115 0 46 0 0 0 2 116 121 112 101 0 7 0 0 0 100 97 114 119 105 110 0 2 97 114 99 104 105 116 101 99 116 117 114 101 0 6 0 0 0 97 109 100 54 52 0 0 2 112 108 97 116 102 111 114 109 0 8 0 0 0 103 111 49 46 57 46 50 0 3 97 112 112 108 105 99 97 116 105 111 110 0 27 0 0 0 2 110 97 109 101 0 12 0 0 0 104 101 108 108 111 45 119 111 114 108 100 0 0 0]
}

func BenchmarkDocument(b *testing.B) {
	b.ReportAllocs()
	internalVersion := "1234567"
	for i := 0; i < b.N; i++ {
		doc := NewDocument(
			EC.SubDocumentFromElements("driver",
				EC.String("name", "mongo-go-driver"),
				EC.String("version", internalVersion),
			),
			EC.SubDocumentFromElements("os",
				EC.String("type", "darwin"),
				EC.String("architecture", "amd64"),
			),
			EC.String("platform", "go1.9.2"),
		)
		_, _ = doc.MarshalBSON()
	}
}

func valueEqual(v1, v2 *Value) bool {
	if v1 == nil && v2 == nil {
		return true
	}

	if v1 == nil || v2 == nil {
		return false
	}

	if v1.start != v2.start {
		return false
	}

	if v1.offset != v2.offset {
		return false
	}
	return true
}

func elementEqual(e1, e2 *Element) bool {
	if e1 == nil && e2 == nil {
		return true
	}
	if e1 == nil || e2 == nil {
		return false
	}

	return valueEqual(e1.value, e2.value)
}

func documentComparer(d1, d2 *Document) bool {
	if (len(d1.elems) != len(d2.elems)) || (len(d1.index) != len(d2.index)) {
		return false
	}
	for index := range d1.elems {
		b1, err := d1.elems[index].MarshalBSON()
		if err != nil {
			return false
		}
		b2, err := d2.elems[index].MarshalBSON()
		if err != nil {
			return false
		}

		if !bytes.Equal(b1, b2) {
			return false
		}

		if d1.index[index] != d2.index[index] {
			return false
		}
	}
	return true
}
