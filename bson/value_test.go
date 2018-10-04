package bson

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestValue(t *testing.T) {
	t.Run("panic", func(t *testing.T) {
		handle := func() {
			if got := recover(); got != ErrUninitializedElement {
				want := ErrUninitializedElement
				t.Errorf("Incorrect value for panic. got %s; want %s", got, want)
			}
		}
		t.Run("key", func(t *testing.T) {
			defer handle()
			(*Element)(nil).Key()
		})
		t.Run("type", func(t *testing.T) {
			defer handle()
			(*Value)(nil).Type()
		})
		t.Run("double", func(t *testing.T) {
			defer handle()
			(*Value)(nil).Double()
		})
		t.Run("string", func(t *testing.T) {
			defer handle()
			(*Value)(nil).StringValue()
		})
		t.Run("document", func(t *testing.T) {
			defer handle()
			(*Value)(nil).ReaderDocument()
		})
	})
	t.Run("key", func(t *testing.T) {
		buf := []byte{
			'\x00', '\x00', '\x00', '\x00',
			'\x02', 'f', 'o', 'o', '\x00',
			'\x00', '\x00', '\x00', '\x00', '\x00',
			'\x00'}
		e := &Element{&Value{start: 4, offset: 9, data: buf}}
		want := "foo"
		got := e.Key()
		if got != want {
			t.Errorf("Unexpected result. got %s; want %s", got, want)
		}
	})
	t.Run("type", func(t *testing.T) {
		buf := []byte{
			'\x00', '\x00', '\x00', '\x00',
			'\x02', 'f', 'o', 'o', '\x00',
			'\x00', '\x00', '\x00', '\x00', '\x00',
			'\x00',
		}
		e := &Element{&Value{start: 4, offset: 9, data: buf}}
		want := TypeString
		got := e.value.Type()
		if got != want {
			t.Errorf("Unexpected result. got %v; want %v", got, want)
		}
	})
	t.Run("double", func(t *testing.T) {
		buf := []byte{
			'\x00', '\x00', '\x00', '\x00',
			'\x01', 'f', 'o', 'o', '\x00',
			'\x00', '\x00', '\x00', '\x00',
			'\x00', '\x00', '\x00', '\x00',
			'\x00',
		}
		e := &Element{&Value{start: 4, offset: 9, data: buf}}
		binary.LittleEndian.PutUint64(buf[9:17], math.Float64bits(3.14159))
		want := 3.14159
		got := e.value.Double()
		if got != want {
			t.Errorf("Unexpected result. got %f; want %f", got, want)
		}
	})
	t.Run("string", func(t *testing.T) {
		buf := []byte{
			'\x00', '\x00', '\x00', '\x00',
			'\x02', 'f', 'o', 'o', '\x00',
			'\x00', '\x00', '\x00', '\x00',
			'b', 'a', 'r', '\x00',
			'\x00',
		}
		e := &Element{&Value{start: 4, offset: 9, data: buf}}
		binary.LittleEndian.PutUint32(buf[9:13], 4)
		want := "bar"
		got := e.value.StringValue()
		if got != want {
			t.Errorf("Unexpected result. got %s; want %s", got, want)
		}
	})
	t.Run("Equal", func(t *testing.T) {
		codewithscopeval := func() *Value {
			b, err := NewDocument(
				EC.CodeWithScope("cws", "var hello = 'world';",
					NewDocument(EC.Boolean("foo", true)),
				)).MarshalBSON()
			noerr(t, err)
			elem, err := Reader(b).Lookup("cws")
			noerr(t, err)
			return elem.Value()
		}
		testCases := []struct {
			name  string
			v1    *Value
			v2    *Value
			equal bool
		}{
			{"both nil", nil, nil, true},
			{"v1 nil", nil, VC.Boolean(true), false},
			{"v2 nil", VC.Boolean(true), nil, false},
			{"different types", VC.String("foo"), VC.Boolean(true), false},
			{
				"both with d field equal",
				VC.DocumentFromElements(EC.String("hello", "world")),
				VC.DocumentFromElements(EC.String("hello", "world")),
				true,
			},
			{
				"both with d field not equal",
				VC.DocumentFromElements(EC.String("hello", "world")),
				VC.DocumentFromElements(EC.Boolean("foo", true)),
				false,
			},
			{
				"v1 invalid d field",
				VC.Document(&Document{elems: []*Element{nil}}),
				VC.Boolean(true),
				false,
			},
			{
				"v2 invalid d field",
				VC.Boolean(true),
				VC.Document(&Document{elems: []*Element{nil}}),
				false,
			},
			{
				"equal bytes",
				VC.Boolean(true),
				VC.Boolean(true),
				true,
			},
			{
				"equal document",
				VC.DocumentFromReader(Reader{0x08, 0x00, 0x00, 0x00, 0x08, 0x00, 0x01, 0x00}),
				VC.DocumentFromElements(EC.Boolean("", true)),
				true,
			},
			{
				"equal array",
				VC.ArrayFromValues(VC.Boolean(true)),
				&Value{
					start: 0, offset: 2,
					data: []byte{0x04, 0x00, 0x09, 0x00, 0x00, 0x00, 0x08, 0x30, 0x00, 0x01, 0x00},
				},
				true,
			},
			{
				"equal code with scope",
				VC.CodeWithScope("var hello = 'world';", NewDocument(EC.Boolean("foo", true))),
				codewithscopeval(),
				true,
			},
			{
				"v1 code with scope invalid d",
				VC.CodeWithScope("var hello = 'world';", &Document{elems: []*Element{nil}}),
				codewithscopeval(),
				false,
			},
			{
				"v2 code with scope invalid d",
				codewithscopeval(),
				VC.CodeWithScope("var hello = 'world';", &Document{elems: []*Element{nil}}),
				false,
			},
			{
				"unused d field",
				VC.Boolean(true),
				&Value{start: 0, offset: 2, data: []byte{0x08, 0x00, 0x01}, d: NewDocument(EC.Undefined("val"))},
				true,
			},
			{
				"read JavaScript error",
				&Value{
					start: 0, offset: 2,
					data: []byte{0x0F, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0x00, 0x00, 0x00}, d: NewDocument(),
				},
				codewithscopeval(),
				false,
			},
		}

		for idx, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				equal := tc.v1.Equal(tc.v2)
				if equal != tc.equal {
					t.Errorf("test case #%d: Expected equality not satisfied. got=%t; want=%t", idx, equal, tc.equal)
					t.Errorf("\nv1: %#v\nv2: %#v", tc.v1, tc.v2)
					spew.Dump(tc.v1, tc.v2)
				}
			})
		}
	})
}
