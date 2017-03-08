// BSON library for Go
//
// Copyright (c) 2010-2012 - Gustavo Niemeyer <gustavo@niemeyer.net>
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
// gobson - BSON library for Go.

package bson_test

import (
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/stretchr/testify/require"
)

// Wrap up the document elements contained in data, prepending the int32
// length of the data, and appending the '\x00' value closing the document.
func wrapInDoc(data string) string {
	result := make([]byte, len(data)+5)
	binary.LittleEndian.PutUint32(result, uint32(len(result)))
	copy(result[4:], []byte(data))
	return string(result)
}

func makeZeroDoc(value interface{}) (zero interface{}) {
	v := reflect.ValueOf(value)
	t := v.Type()
	switch t.Kind() {
	case reflect.Map:
		mv := reflect.MakeMap(t)
		zero = mv.Interface()
	case reflect.Ptr:
		pv := reflect.New(v.Type().Elem())
		zero = pv.Interface()
	case reflect.Slice, reflect.Int, reflect.Int64, reflect.Struct:
		zero = reflect.New(t).Interface()
	default:
		panic("unsupported doc type: " + t.Name())
	}
	return zero
}

func testUnmarshal(t *testing.T, data string, obj interface{}) {
	zero := makeZeroDoc(obj)
	err := bson.Unmarshal([]byte(data), zero)
	require.NoError(t, err)
	require.Equal(t, zero, obj)
}

type testItemType struct {
	obj  interface{}
	data string
}

// --------------------------------------------------------------------------
// Samples from bsonspec.org:

var sampleItems = []testItemType{
	{bson.M{"hello": "world"},
		"\x16\x00\x00\x00\x02hello\x00\x06\x00\x00\x00world\x00\x00"},
	{bson.M{"BSON": []interface{}{"awesome", float64(5.05), 1986}},
		"1\x00\x00\x00\x04BSON\x00&\x00\x00\x00\x020\x00\x08\x00\x00\x00" +
			"awesome\x00\x011\x00333333\x14@\x102\x00\xc2\x07\x00\x00\x00\x00"},
}

func TestMarshalSampleItems(t *testing.T) {
	for _, item := range sampleItems {
		t.Run(item.data, func(t *testing.T) {
			data, err := bson.Marshal(item.obj)
			require.NoError(t, err)
			require.Equal(t, string(data), item.data)
		})
	}
}

func TestUnmarshalSampleItems(t *testing.T) {
	for _, item := range sampleItems {
		t.Run(item.data, func(t *testing.T) {
			value := bson.M{}
			err := bson.Unmarshal([]byte(item.data), value)
			require.NoError(t, err)
			require.Equal(t, value, item.obj)
		})
	}
}

// --------------------------------------------------------------------------
// Every type, ordered by the type flag. These are not wrapped with the
// length and last \x00 from the document. wrapInDoc() computes them.
// Note that all of them should be supported as two-way conversions.

var allItems = []testItemType{
	{bson.M{},
		""},
	{bson.M{"_": float64(5.05)},
		"\x01_\x00333333\x14@"},
	{bson.M{"_": "yo"},
		"\x02_\x00\x03\x00\x00\x00yo\x00"},
	{bson.M{"_": bson.M{"a": true}},
		"\x03_\x00\x09\x00\x00\x00\x08a\x00\x01\x00"},
	{bson.M{"_": []interface{}{true, false}},
		"\x04_\x00\r\x00\x00\x00\x080\x00\x01\x081\x00\x00\x00"},
	{bson.M{"_": []byte("yo")},
		"\x05_\x00\x02\x00\x00\x00\x00yo"},
	{bson.M{"_": bson.Binary{0x80, []byte("udef")}},
		"\x05_\x00\x04\x00\x00\x00\x80udef"},
	{bson.M{"_": bson.Undefined}, // Obsolete, but still seen in the wild.
		"\x06_\x00"},
	{bson.M{"_": bson.ObjectId("0123456789ab")},
		"\x07_\x000123456789ab"},
	{bson.M{"_": bson.DBPointer{"testnamespace", bson.ObjectId("0123456789ab")}},
		"\x0C_\x00\x0e\x00\x00\x00testnamespace\x000123456789ab"},
	{bson.M{"_": false},
		"\x08_\x00\x00"},
	{bson.M{"_": true},
		"\x08_\x00\x01"},
	{bson.M{"_": time.Unix(0, 258e6)}, // Note the NS <=> MS conversion.
		"\x09_\x00\x02\x01\x00\x00\x00\x00\x00\x00"},
	{bson.M{"_": nil},
		"\x0A_\x00"},
	{bson.M{"_": bson.RegEx{"ab", "cd"}},
		"\x0B_\x00ab\x00cd\x00"},
	{bson.M{"_": bson.JavaScript{"code", nil}},
		"\x0D_\x00\x05\x00\x00\x00code\x00"},
	{bson.M{"_": bson.Symbol("sym")},
		"\x0E_\x00\x04\x00\x00\x00sym\x00"},
	{bson.M{"_": bson.JavaScript{"code", bson.M{"": nil}}},
		"\x0F_\x00\x14\x00\x00\x00\x05\x00\x00\x00code\x00" +
			"\x07\x00\x00\x00\x0A\x00\x00"},
	{bson.M{"_": 258},
		"\x10_\x00\x02\x01\x00\x00"},
	{bson.M{"_": bson.MongoTimestamp(258)},
		"\x11_\x00\x02\x01\x00\x00\x00\x00\x00\x00"},
	{bson.M{"_": int64(258)},
		"\x12_\x00\x02\x01\x00\x00\x00\x00\x00\x00"},
	{bson.M{"_": int64(258 << 32)},
		"\x12_\x00\x00\x00\x00\x00\x02\x01\x00\x00"},
	{bson.M{"_": bson.MaxKey},
		"\x7F_\x00"},
	{bson.M{"_": bson.MinKey},
		"\xFF_\x00"},
}

func TestMarshalAllItems(t *testing.T) {
	for _, item := range allItems {
		t.Run(item.data, func(t *testing.T) {
			data, err := bson.Marshal(item.obj)
			require.NoError(t, err)
			require.Equal(t, string(data), wrapInDoc(item.data))
		})
	}
}

func TestUnmarshalAllItems(t *testing.T) {
	for _, item := range allItems {
		t.Run(item.data, func(t *testing.T) {
			value := bson.M{}
			err := bson.Unmarshal([]byte(wrapInDoc(item.data)), value)
			require.NoError(t, err)
			require.Equal(t, value, item.obj)
		})
	}
}

func TestUnmarshalRawAllItems(t *testing.T) {
	for _, item := range allItems {
		if len(item.data) == 0 {
			continue
		}
		value := item.obj.(bson.M)["_"]
		if value == nil {
			continue
		}
		t.Run(item.data, func(t *testing.T) {
			pv := reflect.New(reflect.ValueOf(value).Type())
			raw := bson.Raw{item.data[0], []byte(item.data[3:])}
			err := raw.Unmarshal(pv.Interface())
			require.NoError(t, err)
			require.Equal(t, pv.Elem().Interface(), value)
		})
	}
}

func TestUnmarshalRawIncompatible(t *testing.T) {
	raw := bson.Raw{0x08, []byte{0x01}} // true
	err := raw.Unmarshal(&struct{}{})
	require.Error(t, err)
}

func TestUnmarshalZeroesStruct(t *testing.T) {
	data, err := bson.Marshal(bson.M{"b": 2})
	require.NoError(t, err)
	type T struct{ A, B int }
	v := T{A: 1}
	err = bson.Unmarshal(data, &v)
	require.NoError(t, err)
	require.Equal(t, v.A, 0)
	require.Equal(t, v.B, 2)
}

func TestUnmarshalZeroesMap(t *testing.T) {
	data, err := bson.Marshal(bson.M{"b": 2})
	require.NoError(t, err)
	m := bson.M{"a": 1}
	err = bson.Unmarshal(data, &m)
	require.NoError(t, err)
	require.Equal(t, m, bson.M{"b": 2})
}

func TestUnmarshalNonNilInterface(t *testing.T) {
	data, err := bson.Marshal(bson.M{"b": 2})
	require.NoError(t, err)
	m := bson.M{"a": 1}
	var i interface{}
	i = m
	err = bson.Unmarshal(data, &i)
	require.NoError(t, err)
	require.Equal(t, i, bson.M{"b": 2})
	require.Equal(t, m, bson.M{"a": 1})
}

// --------------------------------------------------------------------------
// Some one way marshaling operations which would unmarshal differently.

var oneWayMarshalItems = []testItemType{
	// These are being passed as pointers, and will unmarshal as values.
	{bson.M{"": &bson.Binary{0x02, []byte("old")}},
		"\x05\x00\x07\x00\x00\x00\x02\x03\x00\x00\x00old"},
	{bson.M{"": &bson.Binary{0x80, []byte("udef")}},
		"\x05\x00\x04\x00\x00\x00\x80udef"},
	{bson.M{"": &bson.RegEx{"ab", "cd"}},
		"\x0B\x00ab\x00cd\x00"},
	{bson.M{"": &bson.JavaScript{"code", nil}},
		"\x0D\x00\x05\x00\x00\x00code\x00"},
	{bson.M{"": &bson.JavaScript{"code", bson.M{"": nil}}},
		"\x0F\x00\x14\x00\x00\x00\x05\x00\x00\x00code\x00" +
			"\x07\x00\x00\x00\x0A\x00\x00"},

	// There's no float32 type in BSON.  Will encode as a float64.
	{bson.M{"": float32(5.05)},
		"\x01\x00\x00\x00\x00@33\x14@"},

	// The array will be unmarshaled as a slice instead.
	{bson.M{"": [2]bool{true, false}},
		"\x04\x00\r\x00\x00\x00\x080\x00\x01\x081\x00\x00\x00"},

	// The typed slice will be unmarshaled as []interface{}.
	{bson.M{"": []bool{true, false}},
		"\x04\x00\r\x00\x00\x00\x080\x00\x01\x081\x00\x00\x00"},

	// Will unmarshal as a []byte.
	{bson.M{"": bson.Binary{0x00, []byte("yo")}},
		"\x05\x00\x02\x00\x00\x00\x00yo"},
	{bson.M{"": bson.Binary{0x02, []byte("old")}},
		"\x05\x00\x07\x00\x00\x00\x02\x03\x00\x00\x00old"},

	// No way to preserve the type information here. We might encode as a zero
	// value, but this would mean that pointer values in structs wouldn't be
	// able to correctly distinguish between unset and set to the zero value.
	{bson.M{"": (*byte)(nil)},
		"\x0A\x00"},

	// No int types smaller than int32 in BSON. Could encode this as a char,
	// but it would still be ambiguous, take more, and be awkward in Go when
	// loaded without typing information.
	{bson.M{"": byte(8)},
		"\x10\x00\x08\x00\x00\x00"},

	// There are no unsigned types in BSON.  Will unmarshal as int32 or int64.
	{bson.M{"": uint32(258)},
		"\x10\x00\x02\x01\x00\x00"},
	{bson.M{"": uint64(258)},
		"\x12\x00\x02\x01\x00\x00\x00\x00\x00\x00"},
	{bson.M{"": uint64(258 << 32)},
		"\x12\x00\x00\x00\x00\x00\x02\x01\x00\x00"},

	// This will unmarshal as int.
	{bson.M{"": int32(258)},
		"\x10\x00\x02\x01\x00\x00"},

	// That's a special case. The unsigned value is too large for an int32,
	// so an int64 is used instead.
	{bson.M{"": uint32(1<<32 - 1)},
		"\x12\x00\xFF\xFF\xFF\xFF\x00\x00\x00\x00"},
	{bson.M{"": uint(1<<32 - 1)},
		"\x12\x00\xFF\xFF\xFF\xFF\x00\x00\x00\x00"},
}

func TestOneWayMarshalItems(t *testing.T) {
	for _, item := range oneWayMarshalItems {
		t.Run(item.data, func(t *testing.T) {
			data, err := bson.Marshal(item.obj)
			require.NoError(t, err)
			require.Equal(t, string(data), wrapInDoc(item.data))
		})
	}
}

// --------------------------------------------------------------------------
// Two-way tests for user-defined structures using the samples
// from bsonspec.org.

type specSample1 struct {
	Hello string
}

type specSample2 struct {
	BSON []interface{} `bson:"BSON"`
}

var structSampleItems = []testItemType{
	{&specSample1{"world"},
		"\x16\x00\x00\x00\x02hello\x00\x06\x00\x00\x00world\x00\x00"},
	{&specSample2{[]interface{}{"awesome", float64(5.05), 1986}},
		"1\x00\x00\x00\x04BSON\x00&\x00\x00\x00\x020\x00\x08\x00\x00\x00" +
			"awesome\x00\x011\x00333333\x14@\x102\x00\xc2\x07\x00\x00\x00\x00"},
}

func TestMarshalStructSampleItems(t *testing.T) {
	for _, item := range structSampleItems {
		t.Run(item.data, func(t *testing.T) {
			data, err := bson.Marshal(item.obj)
			require.NoError(t, err)
			require.Equal(t, string(data), item.data)
		})
	}
}

func TestUnmarshalStructSampleItems(t *testing.T) {
	for _, item := range structSampleItems {
		t.Run(item.data, func(t *testing.T) {
			testUnmarshal(t, item.data, item.obj)
		})
	}
}

func Test64bitInt(t *testing.T) {
	var i int64 = (1 << 31)
	if int(i) > 0 {
		data, err := bson.Marshal(bson.M{"i": int(i)})
		require.NoError(t, err)
		require.Equal(t, string(data), wrapInDoc("\x12i\x00\x00\x00\x00\x80\x00\x00\x00\x00"))

		var result struct{ I int }
		err = bson.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Equal(t, int64(result.I), i)
	}
}

// --------------------------------------------------------------------------
// Generic two-way struct marshaling tests.

var bytevar = byte(8)
var byteptr = &bytevar

var structItems = []testItemType{
	{&struct{ Ptr *byte }{nil},
		"\x0Aptr\x00"},
	{&struct{ Ptr *byte }{&bytevar},
		"\x10ptr\x00\x08\x00\x00\x00"},
	{&struct{ Ptr **byte }{&byteptr},
		"\x10ptr\x00\x08\x00\x00\x00"},
	{&struct{ Byte byte }{8},
		"\x10byte\x00\x08\x00\x00\x00"},
	{&struct{ Byte byte }{0},
		"\x10byte\x00\x00\x00\x00\x00"},
	{&struct {
		V byte `bson:"Tag"`
	}{8},
		"\x10Tag\x00\x08\x00\x00\x00"},
	{&struct {
		V *struct {
			Byte byte
		}
	}{&struct{ Byte byte }{8}},
		"\x03v\x00" + "\x0f\x00\x00\x00\x10byte\x00\b\x00\x00\x00\x00"},
	{&struct{ priv byte }{}, ""},

	// The order of the dumped fields should be the same in the struct.
	{&struct{ A, C, B, D, F, E *byte }{},
		"\x0Aa\x00\x0Ac\x00\x0Ab\x00\x0Ad\x00\x0Af\x00\x0Ae\x00"},

	{&struct{ V bson.Raw }{bson.Raw{0x03, []byte("\x0f\x00\x00\x00\x10byte\x00\b\x00\x00\x00\x00")}},
		"\x03v\x00" + "\x0f\x00\x00\x00\x10byte\x00\b\x00\x00\x00\x00"},
	{&struct{ V bson.Raw }{bson.Raw{0x10, []byte("\x00\x00\x00\x00")}},
		"\x10v\x00" + "\x00\x00\x00\x00"},

	// Byte arrays.
	{&struct{ V [2]byte }{[2]byte{'y', 'o'}},
		"\x05v\x00\x02\x00\x00\x00\x00yo"},
}

func TestMarshalStructItems(t *testing.T) {
	for _, item := range structItems {
		t.Run(item.data, func(t *testing.T) {
			data, err := bson.Marshal(item.obj)
			require.NoError(t, err)
			require.Equal(t, string(data), wrapInDoc(item.data))
		})
	}
}

func TestUnmarshalStructItems(t *testing.T) {
	for _, item := range structItems {
		t.Run(item.data, func(t *testing.T) {
			testUnmarshal(t, wrapInDoc(item.data), item.obj)
		})
	}
}

func TestUnmarshalRawStructItems(t *testing.T) {
	for _, item := range structItems {
		t.Run(item.data, func(t *testing.T) {
			raw := bson.Raw{0x03, []byte(wrapInDoc(item.data))}
			zero := makeZeroDoc(item.obj)
			err := raw.Unmarshal(zero)
			require.NoError(t, err)
			require.Equal(t, zero, item.obj)
		})
	}
}

func TestUnmarshalRawNil(t *testing.T) {
	// Regression test: shouldn't try to nil out the pointer itself,
	// as it's not settable.
	raw := bson.Raw{0x0A, []byte{}}
	err := raw.Unmarshal(&struct{}{})
	require.NoError(t, err)
}

// --------------------------------------------------------------------------
// One-way marshaling tests.

type dOnIface struct {
	D interface{}
}

type ignoreField struct {
	Before string
	Ignore string `bson:"-"`
	After  string
}

var marshalItems = []testItemType{
	// Ordered document dump.  Will unmarshal as a dictionary by default.
	{bson.D{{"a", nil}, {"c", nil}, {"b", nil}, {"d", nil}, {"f", nil}, {"e", true}},
		"\x0Aa\x00\x0Ac\x00\x0Ab\x00\x0Ad\x00\x0Af\x00\x08e\x00\x01"},
	{MyD{{"a", nil}, {"c", nil}, {"b", nil}, {"d", nil}, {"f", nil}, {"e", true}},
		"\x0Aa\x00\x0Ac\x00\x0Ab\x00\x0Ad\x00\x0Af\x00\x08e\x00\x01"},
	{&dOnIface{bson.D{{"a", nil}, {"c", nil}, {"b", nil}, {"d", true}}},
		"\x03d\x00" + wrapInDoc("\x0Aa\x00\x0Ac\x00\x0Ab\x00\x08d\x00\x01")},

	{bson.RawD{{"a", bson.Raw{0x0A, nil}}, {"c", bson.Raw{0x0A, nil}}, {"b", bson.Raw{0x08, []byte{0x01}}}},
		"\x0Aa\x00" + "\x0Ac\x00" + "\x08b\x00\x01"},
	{MyRawD{{"a", bson.Raw{0x0A, nil}}, {"c", bson.Raw{0x0A, nil}}, {"b", bson.Raw{0x08, []byte{0x01}}}},
		"\x0Aa\x00" + "\x0Ac\x00" + "\x08b\x00\x01"},
	{&dOnIface{bson.RawD{{"a", bson.Raw{0x0A, nil}}, {"c", bson.Raw{0x0A, nil}}, {"b", bson.Raw{0x08, []byte{0x01}}}}},
		"\x03d\x00" + wrapInDoc("\x0Aa\x00"+"\x0Ac\x00"+"\x08b\x00\x01")},

	{&ignoreField{"before", "ignore", "after"},
		"\x02before\x00\a\x00\x00\x00before\x00\x02after\x00\x06\x00\x00\x00after\x00"},

	// Marshalling a Raw document does nothing.
	{bson.Raw{0x03, []byte(wrapInDoc("anything"))},
		"anything"},
	{bson.Raw{Data: []byte(wrapInDoc("anything"))},
		"anything"},
}

func TestMarshalOneWayItems(t *testing.T) {
	for _, item := range marshalItems {
		t.Run(item.data, func(t *testing.T) {
			data, err := bson.Marshal(item.obj)
			require.NoError(t, err)
			require.Equal(t, string(data), wrapInDoc(item.data))
		})
	}
}

// --------------------------------------------------------------------------
// One-way unmarshaling tests.

var unmarshalItems = []testItemType{
	// Field is private.  Should not attempt to unmarshal it.
	{&struct{ priv byte }{},
		"\x10priv\x00\x08\x00\x00\x00"},

	// Wrong casing. Field names are lowercased.
	{&struct{ Byte byte }{},
		"\x10Byte\x00\x08\x00\x00\x00"},

	// Ignore non-existing field.
	{&struct{ Byte byte }{9},
		"\x10boot\x00\x08\x00\x00\x00" + "\x10byte\x00\x09\x00\x00\x00"},

	// Do not unmarshal on ignored field.
	{&ignoreField{"before", "", "after"},
		"\x02before\x00\a\x00\x00\x00before\x00" +
			"\x02-\x00\a\x00\x00\x00ignore\x00" +
			"\x02after\x00\x06\x00\x00\x00after\x00"},

	// Ignore unsuitable types silently.
	{map[string]string{"str": "s"},
		"\x02str\x00\x02\x00\x00\x00s\x00" + "\x10int\x00\x01\x00\x00\x00"},
	{map[string][]int{"array": []int{5, 9}},
		"\x04array\x00" + wrapInDoc("\x100\x00\x05\x00\x00\x00"+"\x021\x00\x02\x00\x00\x00s\x00"+"\x102\x00\x09\x00\x00\x00")},

	// Wrong type. Shouldn't init pointer.
	{&struct{ Str *byte }{},
		"\x02str\x00\x02\x00\x00\x00s\x00"},
	{&struct{ Str *struct{ Str string } }{},
		"\x02str\x00\x02\x00\x00\x00s\x00"},

	// Ordered document.
	{&struct{ bson.D }{bson.D{{"a", nil}, {"c", nil}, {"b", nil}, {"d", true}}},
		"\x03d\x00" + wrapInDoc("\x0Aa\x00\x0Ac\x00\x0Ab\x00\x08d\x00\x01")},

	// Raw document.
	{&bson.Raw{0x03, []byte(wrapInDoc("\x10byte\x00\x08\x00\x00\x00"))},
		"\x10byte\x00\x08\x00\x00\x00"},

	// RawD document.
	{&struct{ bson.RawD }{bson.RawD{{"a", bson.Raw{0x0A, []byte{}}}, {"c", bson.Raw{0x0A, []byte{}}}, {"b", bson.Raw{0x08, []byte{0x01}}}}},
		"\x03rawd\x00" + wrapInDoc("\x0Aa\x00\x0Ac\x00\x08b\x00\x01")},

	// Decode old binary.
	{bson.M{"_": []byte("old")},
		"\x05_\x00\x07\x00\x00\x00\x02\x03\x00\x00\x00old"},

	// Decode old binary without length. According to the spec, this shouldn't happen.
	{bson.M{"_": []byte("old")},
		"\x05_\x00\x03\x00\x00\x00\x02old"},

	// Decode a doc within a doc in to a slice within a doc; shouldn't error
	{&struct{ Foo []string }{},
		"\x03\x66\x6f\x6f\x00\x05\x00\x00\x00\x00"},
}

func TestUnmarshalOneWayItems(t *testing.T) {
	for _, item := range unmarshalItems {
		t.Run(item.data, func(t *testing.T) {
			testUnmarshal(t, wrapInDoc(item.data), item.obj)
		})
	}
}

func TestUnmarshalNilInStruct(t *testing.T) {
	// Nil is the default value, so we need to ensure it's indeed being set.
	b := byte(1)
	v := &struct{ Ptr *byte }{&b}
	err := bson.Unmarshal([]byte(wrapInDoc("\x0Aptr\x00")), v)
	require.NoError(t, err)
	require.Equal(t, v, &struct{ Ptr *byte }{nil})
}

// --------------------------------------------------------------------------
// Marshalling error cases.

type structWithDupKeys struct {
	Name  byte
	Other byte `bson:"name"` // Tag should precede.
}

var marshalErrorItems = []testItemType{
	{bson.M{"": uint64(1 << 63)},
		"BSON has no uint64 type, and value is too large to fit correctly in an int64"},
	{bson.M{"": bson.ObjectId("tooshort")},
		"ObjectIDs must be exactly 12 bytes long (got 8)"},
	{int64(123),
		"Can't marshal int64 as a BSON document"},
	{bson.M{"": 1i},
		"Can't marshal complex128 in a BSON document"},
	{&structWithDupKeys{},
		"Duplicated key 'name' in struct bson_test.structWithDupKeys"},
	{bson.Raw{0xA, []byte{}},
		"Attempted to marshal Raw kind 10 as a document"},
	{bson.Raw{0x3, []byte{}},
		"Attempted to marshal empty Raw document"},
	{bson.M{"w": bson.Raw{0x3, []byte{}}},
		"Attempted to marshal empty Raw document"},
	{&inlineCantPtr{&struct{ A, B int }{1, 2}},
		"Option ,inline needs a struct value or map field"},
	{&inlineDupName{1, struct{ A, B int }{2, 3}},
		"Duplicated key 'a' in struct bson_test.inlineDupName"},
	{&inlineDupMap{},
		"Multiple ,inline maps in struct bson_test.inlineDupMap"},
	{&inlineBadKeyMap{},
		"Option ,inline needs a map with string keys in struct bson_test.inlineBadKeyMap"},
	{&inlineMap{A: 1, M: map[string]interface{}{"a": 1}},
		`Can't have key "a" in inlined map; conflicts with struct field`},
}

func TestMarshalErrorItems(t *testing.T) {
	for _, item := range marshalErrorItems {
		t.Run(item.data, func(t *testing.T) {
			data, err := bson.Marshal(item.obj)
			require.EqualError(t, err, item.data)
			require.Nil(t, data)
		})

	}
}

// --------------------------------------------------------------------------
// Unmarshalling error cases.

type unmarshalErrorType struct {
	obj   interface{}
	data  string
	error string
}

var unmarshalErrorItems = []unmarshalErrorType{
	// Tag name conflicts with existing parameter.
	{&structWithDupKeys{},
		"\x10name\x00\x08\x00\x00\x00",
		"Duplicated key 'name' in struct bson_test.structWithDupKeys"},

	// Non-string map key.
	{map[int]interface{}{},
		"\x10name\x00\x08\x00\x00\x00",
		"BSON map must have string keys. Got: map[int]interface {}"},

	{nil,
		"\xEEname\x00",
		"Unknown element kind (0xEE)"},

	{struct{ Name bool }{},
		"\x10name\x00\x08\x00\x00\x00",
		"Unmarshal can't deal with struct values. Use a pointer."},

	{123,
		"\x10name\x00\x08\x00\x00\x00",
		"Unmarshal needs a map or a pointer to a struct."},

	{nil,
		"\x08\x62\x00\x02",
		"encoded boolean must be 1 or 0, found 2"},
}

func TestUnmarshalErrorItems(t *testing.T) {
	for _, item := range unmarshalErrorItems {
		t.Run(item.error, func(t *testing.T) {
			data := []byte(wrapInDoc(item.data))
			var value interface{}
			switch reflect.ValueOf(item.obj).Kind() {
			case reflect.Map, reflect.Ptr:
				value = makeZeroDoc(item.obj)
			case reflect.Invalid:
				value = bson.M{}
			default:
				value = item.obj
			}
			err := bson.Unmarshal(data, value)
			require.EqualError(t, err, item.error)
		})
	}
}

type unmarshalRawErrorType struct {
	obj   interface{}
	raw   bson.Raw
	error string
}

var unmarshalRawErrorItems = []unmarshalRawErrorType{
	// Tag name conflicts with existing parameter.
	{&structWithDupKeys{},
		bson.Raw{0x03, []byte("\x10byte\x00\x08\x00\x00\x00")},
		"Duplicated key 'name' in struct bson_test.structWithDupKeys"},

	{&struct{}{},
		bson.Raw{0xEE, []byte{}},
		"Unknown element kind (0xEE)"},

	{struct{ Name bool }{},
		bson.Raw{0x10, []byte("\x08\x00\x00\x00")},
		"Raw Unmarshal can't deal with struct values. Use a pointer."},

	{123,
		bson.Raw{0x10, []byte("\x08\x00\x00\x00")},
		"Raw Unmarshal needs a map or a valid pointer."},
}

func TestUnmarshalRawErrorItems(t *testing.T) {
	for _, item := range unmarshalRawErrorItems {
		t.Run(item.error, func(t *testing.T) {
			err := item.raw.Unmarshal(item.obj)
			require.EqualError(t, err, item.error)
		})
	}
}

var corruptedData = []string{
	"\x04\x00\x00\x00\x00",         // Document shorter than minimum
	"\x06\x00\x00\x00\x00",         // Not enough data
	"\x05\x00\x00",                 // Broken length
	"\x05\x00\x00\x00\xff",         // Corrupted termination
	"\x0A\x00\x00\x00\x0Aooop\x00", // Unfinished C string

	// Array end past end of string (s[2]=0x07 is correct)
	wrapInDoc("\x04\x00\x09\x00\x00\x00\x0A\x00\x00"),

	// Array end within string, but past acceptable.
	wrapInDoc("\x04\x00\x08\x00\x00\x00\x0A\x00\x00"),

	// Document end within string, but past acceptable.
	wrapInDoc("\x03\x00\x08\x00\x00\x00\x0A\x00\x00"),

	// String with corrupted end.
	wrapInDoc("\x02\x00\x03\x00\x00\x00yo\xFF"),

	// String with negative length (issue #116).
	"\x0c\x00\x00\x00\x02x\x00\xff\xff\xff\xff\x00",

	// String with zero length (must include trailing '\x00')
	"\x0c\x00\x00\x00\x02x\x00\x00\x00\x00\x00\x00",

	// Binary with negative length.
	"\r\x00\x00\x00\x05x\x00\xff\xff\xff\xff\x00\x00",
}

func TestUnmarshalMapDocumentTooShort(t *testing.T) {
	for _, data := range corruptedData {
		t.Run(data, func(t *testing.T) {
			err := bson.Unmarshal([]byte(data), bson.M{})
			require.EqualError(t, err, "Document is corrupted")
			err = bson.Unmarshal([]byte(data), &struct{}{})
			require.EqualError(t, err, "Document is corrupted")
		})
	}
}

// --------------------------------------------------------------------------
// Setter test cases.

var setterResult = map[string]error{}

type setterType struct {
	received interface{}
}

func (o *setterType) SetBSON(raw bson.Raw) error {
	err := raw.Unmarshal(&o.received)
	if err != nil {
		panic("The panic:" + err.Error())
	}
	if s, ok := o.received.(string); ok {
		if result, ok := setterResult[s]; ok {
			return result
		}
	}
	return nil
}

type ptrSetterDoc struct {
	Field *setterType `bson:"_"`
}

type valSetterDoc struct {
	Field setterType `bson:"_"`
}

func TestUnmarshalAllItemsWithPtrSetter(t *testing.T) {
	for _, item := range allItems {
		for i := 0; i != 2; i++ {
			var field *setterType
			if i == 0 {
				obj := &ptrSetterDoc{}
				err := bson.Unmarshal([]byte(wrapInDoc(item.data)), obj)
				require.NoError(t, err)
				field = obj.Field
			} else {
				obj := &valSetterDoc{}
				err := bson.Unmarshal([]byte(wrapInDoc(item.data)), obj)
				require.NoError(t, err)
				field = &obj.Field
			}
			if item.data == "" {
				// Nothing to unmarshal. Should be untouched.
				if i == 0 {
					require.Nil(t, field)
				} else {
					require.Nil(t, field.received)
				}
			} else {
				expected := item.obj.(bson.M)["_"]
				require.NotNil(t, field)
				require.Equal(t, field.received, expected)
			}
		}
	}
}

func TestUnmarshalWholeDocumentWithSetter(t *testing.T) {
	obj := &setterType{}
	err := bson.Unmarshal([]byte(sampleItems[0].data), obj)
	require.NoError(t, err)
	require.Equal(t, obj.received, bson.M{"hello": "world"})
}

func TestUnmarshalSetterOmits(t *testing.T) {
	setterResult["2"] = &bson.TypeError{}
	setterResult["4"] = &bson.TypeError{}
	defer func() {
		delete(setterResult, "2")
		delete(setterResult, "4")
	}()

	m := map[string]*setterType{}
	data := wrapInDoc("\x02abc\x00\x02\x00\x00\x001\x00" +
		"\x02def\x00\x02\x00\x00\x002\x00" +
		"\x02ghi\x00\x02\x00\x00\x003\x00" +
		"\x02jkl\x00\x02\x00\x00\x004\x00")
	err := bson.Unmarshal([]byte(data), m)
	require.NoError(t, err)
	require.NotNil(t, m["abc"])
	require.Nil(t, m["def"])
	require.NotNil(t, m["ghi"])
	require.Nil(t, m["jkl"])

	require.Equal(t, m["abc"].received, "1")
	require.Equal(t, m["ghi"].received, "3")
}

func TestUnmarshalSetterErrors(t *testing.T) {
	boom := errors.New("BOOM")
	setterResult["2"] = boom
	defer delete(setterResult, "2")

	m := map[string]*setterType{}
	data := wrapInDoc("\x02abc\x00\x02\x00\x00\x001\x00" +
		"\x02def\x00\x02\x00\x00\x002\x00" +
		"\x02ghi\x00\x02\x00\x00\x003\x00")
	err := bson.Unmarshal([]byte(data), m)
	require.EqualError(t, err, "BOOM")
	require.NotNil(t, m["abc"])
	require.Nil(t, m["def"])
	require.Nil(t, m["ghi"])

	require.Equal(t, m["abc"].received, "1")
}

func TestDMap(t *testing.T) {
	d := bson.D{{"a", 1}, {"b", 2}}
	require.Equal(t, d.Map(), bson.M{"a": 1, "b": 2})
}

func TestUnmarshalSetterSetZero(t *testing.T) {
	setterResult["foo"] = bson.SetZero
	defer delete(setterResult, "field")

	data, err := bson.Marshal(bson.M{"field": "foo"})
	require.NoError(t, err)

	m := map[string]*setterType{}
	err = bson.Unmarshal([]byte(data), m)
	require.NoError(t, err)

	value, ok := m["field"]
	require.Equal(t, ok, true)
	require.Nil(t, value)
}

// --------------------------------------------------------------------------
// Getter test cases.

type typeWithGetter struct {
	result interface{}
	err    error
}

func (t *typeWithGetter) GetBSON() (interface{}, error) {
	if t == nil {
		return "<value is nil>", nil
	}
	return t.result, t.err
}

type docWithGetterField struct {
	Field *typeWithGetter `bson:"_"`
}

func TestMarshalAllItemsWithGetter(t *testing.T) {
	for _, item := range allItems {
		if item.data == "" {
			continue
		}
		t.Run(item.data, func(t *testing.T) {
			obj := &docWithGetterField{}
			obj.Field = &typeWithGetter{result: item.obj.(bson.M)["_"]}
			data, err := bson.Marshal(obj)
			require.NoError(t, err)
			require.Equal(t, string(data), wrapInDoc(item.data))
		})
	}
}

func TestMarshalWholeDocumentWithGetter(t *testing.T) {
	obj := &typeWithGetter{result: sampleItems[0].obj}
	data, err := bson.Marshal(obj)
	require.NoError(t, err)
	require.Equal(t, string(data), sampleItems[0].data)
}

func TestGetterErrors(t *testing.T) {
	e := errors.New("oops")

	obj1 := &docWithGetterField{}
	obj1.Field = &typeWithGetter{sampleItems[0].obj, e}
	data, err := bson.Marshal(obj1)
	require.EqualError(t, err, "oops")
	require.Nil(t, data)

	obj2 := &typeWithGetter{sampleItems[0].obj, e}
	data, err = bson.Marshal(obj2)
	require.EqualError(t, err, "oops")
	require.Nil(t, data)
}

type intGetter int64

func (t intGetter) GetBSON() (interface{}, error) {
	return int64(t), nil
}

type typeWithIntGetter struct {
	V intGetter `bson:",minsize"`
}

func TestMarshalShortWithGetter(t *testing.T) {
	obj := typeWithIntGetter{42}
	data, err := bson.Marshal(obj)
	require.NoError(t, err)
	m := bson.M{}
	err = bson.Unmarshal(data, m)
	require.NoError(t, err)
	require.Equal(t, m["v"], 42)
}

func TestMarshalWithGetterNil(t *testing.T) {
	obj := docWithGetterField{}
	data, err := bson.Marshal(obj)
	require.NoError(t, err)
	m := bson.M{}
	err = bson.Unmarshal(data, m)
	require.NoError(t, err)
	require.Equal(t, m, bson.M{"_": "<value is nil>"})
}

// --------------------------------------------------------------------------
// Cross-type conversion tests.

type crossTypeItem struct {
	obj1 interface{}
	obj2 interface{}
}

type condStr struct {
	V string `bson:",omitempty"`
}
type condStrNS struct {
	V string `a:"A" bson:",omitempty" b:"B"`
}
type condBool struct {
	V bool `bson:",omitempty"`
}
type condInt struct {
	V int `bson:",omitempty"`
}
type condUInt struct {
	V uint `bson:",omitempty"`
}
type condFloat struct {
	V float64 `bson:",omitempty"`
}
type condIface struct {
	V interface{} `bson:",omitempty"`
}
type condPtr struct {
	V *bool `bson:",omitempty"`
}
type condSlice struct {
	V []string `bson:",omitempty"`
}
type condMap struct {
	V map[string]int `bson:",omitempty"`
}
type namedCondStr struct {
	V string `bson:"myv,omitempty"`
}
type condTime struct {
	V time.Time `bson:",omitempty"`
}
type condStruct struct {
	V struct{ A []int } `bson:",omitempty"`
}
type condRaw struct {
	V bson.Raw `bson:",omitempty"`
}

type shortInt struct {
	V int64 `bson:",minsize"`
}
type shortUint struct {
	V uint64 `bson:",minsize"`
}
type shortIface struct {
	V interface{} `bson:",minsize"`
}
type shortPtr struct {
	V *int64 `bson:",minsize"`
}
type shortNonEmptyInt struct {
	V int64 `bson:",minsize,omitempty"`
}

type inlineInt struct {
	V struct{ A, B int } `bson:",inline"`
}
type inlineCantPtr struct {
	V *struct{ A, B int } `bson:",inline"`
}
type inlineDupName struct {
	A int
	V struct{ A, B int } `bson:",inline"`
}
type inlineMap struct {
	A int
	M map[string]interface{} `bson:",inline"`
}
type inlineMapInt struct {
	A int
	M map[string]int `bson:",inline"`
}
type inlineMapMyM struct {
	A int
	M MyM `bson:",inline"`
}
type inlineDupMap struct {
	M1 map[string]interface{} `bson:",inline"`
	M2 map[string]interface{} `bson:",inline"`
}
type inlineBadKeyMap struct {
	M map[int]int `bson:",inline"`
}
type inlineUnexported struct {
	M          map[string]interface{} `bson:",inline"`
	unexported `bson:",inline"`
}
type unexported struct {
	A int
}

type getterSetterD bson.D

func (s getterSetterD) GetBSON() (interface{}, error) {
	if len(s) == 0 {
		return bson.D{}, nil
	}
	return bson.D(s[:len(s)-1]), nil
}

func (s *getterSetterD) SetBSON(raw bson.Raw) error {
	var doc bson.D
	err := raw.Unmarshal(&doc)
	doc = append(doc, bson.DocElem{"suffix", true})
	*s = getterSetterD(doc)
	return err
}

type getterSetterInt int

func (i getterSetterInt) GetBSON() (interface{}, error) {
	return bson.D{{"a", int(i)}}, nil
}

func (i *getterSetterInt) SetBSON(raw bson.Raw) error {
	var doc struct{ A int }
	err := raw.Unmarshal(&doc)
	*i = getterSetterInt(doc.A)
	return err
}

type ifaceType interface {
	Hello()
}

type ifaceSlice []ifaceType

func (s *ifaceSlice) SetBSON(raw bson.Raw) error {
	var ns []int
	if err := raw.Unmarshal(&ns); err != nil {
		return err
	}
	*s = make(ifaceSlice, ns[0])
	return nil
}

func (s ifaceSlice) GetBSON() (interface{}, error) {
	return []int{len(s)}, nil
}

type (
	MyString string
	MyBytes  []byte
	MyBool   bool
	MyD      []bson.DocElem
	MyRawD   []bson.RawDocElem
	MyM      map[string]interface{}
)

var (
	truevar  = true
	falsevar = false

	int64var = int64(42)
	int64ptr = &int64var
	intvar   = int(42)
	intptr   = &intvar

	gsintvar = getterSetterInt(42)
)

func parseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

// That's a pretty fun test.  It will dump the first item, generate a zero
// value equivalent to the second one, load the dumped data onto it, and then
// verify that the resulting value is deep-equal to the untouched second value.
// Then, it will do the same in the *opposite* direction!
var twoWayCrossItems = []crossTypeItem{
	// int<=>int
	{&struct{ I int }{42}, &struct{ I int8 }{42}},
	{&struct{ I int }{42}, &struct{ I int32 }{42}},
	{&struct{ I int }{42}, &struct{ I int64 }{42}},
	{&struct{ I int8 }{42}, &struct{ I int32 }{42}},
	{&struct{ I int8 }{42}, &struct{ I int64 }{42}},
	{&struct{ I int32 }{42}, &struct{ I int64 }{42}},

	// uint<=>uint
	{&struct{ I uint }{42}, &struct{ I uint8 }{42}},
	{&struct{ I uint }{42}, &struct{ I uint32 }{42}},
	{&struct{ I uint }{42}, &struct{ I uint64 }{42}},
	{&struct{ I uint8 }{42}, &struct{ I uint32 }{42}},
	{&struct{ I uint8 }{42}, &struct{ I uint64 }{42}},
	{&struct{ I uint32 }{42}, &struct{ I uint64 }{42}},

	// float32<=>float64
	{&struct{ I float32 }{42}, &struct{ I float64 }{42}},

	// int<=>uint
	{&struct{ I uint }{42}, &struct{ I int }{42}},
	{&struct{ I uint }{42}, &struct{ I int8 }{42}},
	{&struct{ I uint }{42}, &struct{ I int32 }{42}},
	{&struct{ I uint }{42}, &struct{ I int64 }{42}},
	{&struct{ I uint8 }{42}, &struct{ I int }{42}},
	{&struct{ I uint8 }{42}, &struct{ I int8 }{42}},
	{&struct{ I uint8 }{42}, &struct{ I int32 }{42}},
	{&struct{ I uint8 }{42}, &struct{ I int64 }{42}},
	{&struct{ I uint32 }{42}, &struct{ I int }{42}},
	{&struct{ I uint32 }{42}, &struct{ I int8 }{42}},
	{&struct{ I uint32 }{42}, &struct{ I int32 }{42}},
	{&struct{ I uint32 }{42}, &struct{ I int64 }{42}},
	{&struct{ I uint64 }{42}, &struct{ I int }{42}},
	{&struct{ I uint64 }{42}, &struct{ I int8 }{42}},
	{&struct{ I uint64 }{42}, &struct{ I int32 }{42}},
	{&struct{ I uint64 }{42}, &struct{ I int64 }{42}},

	// int <=> float
	{&struct{ I int }{42}, &struct{ I float64 }{42}},

	// int <=> bool
	{&struct{ I int }{1}, &struct{ I bool }{true}},
	{&struct{ I int }{0}, &struct{ I bool }{false}},

	// uint <=> float64
	{&struct{ I uint }{42}, &struct{ I float64 }{42}},

	// uint <=> bool
	{&struct{ I uint }{1}, &struct{ I bool }{true}},
	{&struct{ I uint }{0}, &struct{ I bool }{false}},

	// float64 <=> bool
	{&struct{ I float64 }{1}, &struct{ I bool }{true}},
	{&struct{ I float64 }{0}, &struct{ I bool }{false}},

	// string <=> string and string <=> []byte
	{&struct{ S []byte }{[]byte("abc")}, &struct{ S string }{"abc"}},
	{&struct{ S []byte }{[]byte("def")}, &struct{ S bson.Symbol }{"def"}},
	{&struct{ S string }{"ghi"}, &struct{ S bson.Symbol }{"ghi"}},

	// map <=> struct
	{&struct {
		A struct {
			B, C int
		}
	}{struct{ B, C int }{1, 2}},
		map[string]map[string]int{"a": map[string]int{"b": 1, "c": 2}}},

	{&struct{ A bson.Symbol }{"abc"}, map[string]string{"a": "abc"}},
	{&struct{ A bson.Symbol }{"abc"}, map[string][]byte{"a": []byte("abc")}},
	{&struct{ A []byte }{[]byte("abc")}, map[string]string{"a": "abc"}},
	{&struct{ A uint }{42}, map[string]int{"a": 42}},
	{&struct{ A uint }{42}, map[string]float64{"a": 42}},
	{&struct{ A uint }{1}, map[string]bool{"a": true}},
	{&struct{ A int }{42}, map[string]uint{"a": 42}},
	{&struct{ A int }{42}, map[string]float64{"a": 42}},
	{&struct{ A int }{1}, map[string]bool{"a": true}},
	{&struct{ A float64 }{42}, map[string]float32{"a": 42}},
	{&struct{ A float64 }{42}, map[string]int{"a": 42}},
	{&struct{ A float64 }{42}, map[string]uint{"a": 42}},
	{&struct{ A float64 }{1}, map[string]bool{"a": true}},
	{&struct{ A bool }{true}, map[string]int{"a": 1}},
	{&struct{ A bool }{true}, map[string]uint{"a": 1}},
	{&struct{ A bool }{true}, map[string]float64{"a": 1}},
	{&struct{ A **byte }{&byteptr}, map[string]byte{"a": 8}},

	// url.URL <=> string
	{&struct{ URL *url.URL }{parseURL("h://e.c/p")}, map[string]string{"url": "h://e.c/p"}},
	{&struct{ URL url.URL }{*parseURL("h://e.c/p")}, map[string]string{"url": "h://e.c/p"}},

	// Slices
	{&struct{ S []int }{[]int{1, 2, 3}}, map[string][]int{"s": []int{1, 2, 3}}},
	{&struct{ S *[]int }{&[]int{1, 2, 3}}, map[string][]int{"s": []int{1, 2, 3}}},

	// Conditionals
	{&condBool{true}, map[string]bool{"v": true}},
	{&condBool{}, map[string]bool{}},
	{&condInt{1}, map[string]int{"v": 1}},
	{&condInt{}, map[string]int{}},
	{&condUInt{1}, map[string]uint{"v": 1}},
	{&condUInt{}, map[string]uint{}},
	{&condFloat{}, map[string]int{}},
	{&condStr{"yo"}, map[string]string{"v": "yo"}},
	{&condStr{}, map[string]string{}},
	{&condStrNS{"yo"}, map[string]string{"v": "yo"}},
	{&condStrNS{}, map[string]string{}},
	{&condSlice{[]string{"yo"}}, map[string][]string{"v": []string{"yo"}}},
	{&condSlice{}, map[string][]string{}},
	{&condMap{map[string]int{"k": 1}}, bson.M{"v": bson.M{"k": 1}}},
	{&condMap{}, map[string][]string{}},
	{&condIface{"yo"}, map[string]string{"v": "yo"}},
	{&condIface{""}, map[string]string{"v": ""}},
	{&condIface{}, map[string]string{}},
	{&condPtr{&truevar}, map[string]bool{"v": true}},
	{&condPtr{&falsevar}, map[string]bool{"v": false}},
	{&condPtr{}, map[string]string{}},

	{&condTime{time.Unix(123456789, 123e6)}, map[string]time.Time{"v": time.Unix(123456789, 123e6)}},
	{&condTime{}, map[string]string{}},

	{&condStruct{struct{ A []int }{[]int{1}}}, bson.M{"v": bson.M{"a": []interface{}{1}}}},
	{&condStruct{struct{ A []int }{}}, bson.M{}},

	{&condRaw{bson.Raw{Kind: 0x0A, Data: []byte{}}}, bson.M{"v": nil}},
	{&condRaw{bson.Raw{Kind: 0x00}}, bson.M{}},

	{&namedCondStr{"yo"}, map[string]string{"myv": "yo"}},
	{&namedCondStr{}, map[string]string{}},

	{&shortInt{1}, map[string]interface{}{"v": 1}},
	{&shortInt{1 << 30}, map[string]interface{}{"v": 1 << 30}},
	{&shortInt{1 << 31}, map[string]interface{}{"v": int64(1 << 31)}},
	{&shortUint{1 << 30}, map[string]interface{}{"v": 1 << 30}},
	{&shortUint{1 << 31}, map[string]interface{}{"v": int64(1 << 31)}},
	{&shortIface{int64(1) << 31}, map[string]interface{}{"v": int64(1 << 31)}},
	{&shortPtr{int64ptr}, map[string]interface{}{"v": intvar}},

	{&shortNonEmptyInt{1}, map[string]interface{}{"v": 1}},
	{&shortNonEmptyInt{1 << 31}, map[string]interface{}{"v": int64(1 << 31)}},
	{&shortNonEmptyInt{}, map[string]interface{}{}},

	{&inlineInt{struct{ A, B int }{1, 2}}, map[string]interface{}{"a": 1, "b": 2}},
	{&inlineMap{A: 1, M: map[string]interface{}{"b": 2}}, map[string]interface{}{"a": 1, "b": 2}},
	{&inlineMap{A: 1, M: nil}, map[string]interface{}{"a": 1}},
	{&inlineMapInt{A: 1, M: map[string]int{"b": 2}}, map[string]int{"a": 1, "b": 2}},
	{&inlineMapInt{A: 1, M: nil}, map[string]int{"a": 1}},
	{&inlineMapMyM{A: 1, M: MyM{"b": MyM{"c": 3}}}, map[string]interface{}{"a": 1, "b": map[string]interface{}{"c": 3}}},
	{&inlineUnexported{M: map[string]interface{}{"b": 1}, unexported: unexported{A: 2}}, map[string]interface{}{"b": 1, "a": 2}},

	// []byte <=> Binary
	{&struct{ B []byte }{[]byte("abc")}, map[string]bson.Binary{"b": bson.Binary{Data: []byte("abc")}}},

	// []byte <=> MyBytes
	{&struct{ B MyBytes }{[]byte("abc")}, map[string]string{"b": "abc"}},
	{&struct{ B MyBytes }{[]byte{}}, map[string]string{"b": ""}},
	{&struct{ B MyBytes }{}, map[string]bool{}},
	{&struct{ B []byte }{[]byte("abc")}, map[string]MyBytes{"b": []byte("abc")}},

	// bool <=> MyBool
	{&struct{ B MyBool }{true}, map[string]bool{"b": true}},
	{&struct{ B MyBool }{}, map[string]bool{"b": false}},
	{&struct{ B MyBool }{}, map[string]string{}},
	{&struct{ B bool }{}, map[string]MyBool{"b": false}},

	// arrays
	{&struct{ V [2]int }{[...]int{1, 2}}, map[string][2]int{"v": [2]int{1, 2}}},
	{&struct{ V [2]byte }{[...]byte{1, 2}}, map[string][2]byte{"v": [2]byte{1, 2}}},

	// zero time
	{&struct{ V time.Time }{}, map[string]interface{}{"v": time.Time{}}},

	// zero time + 1 second + 1 millisecond; overflows int64 as nanoseconds
	{&struct{ V time.Time }{time.Unix(-62135596799, 1e6).Local()},
		map[string]interface{}{"v": time.Unix(-62135596799, 1e6).Local()}},

	// bson.D <=> []DocElem
	{&bson.D{{"a", bson.D{{"b", 1}, {"c", 2}}}}, &bson.D{{"a", bson.D{{"b", 1}, {"c", 2}}}}},
	{&bson.D{{"a", bson.D{{"b", 1}, {"c", 2}}}}, &MyD{{"a", MyD{{"b", 1}, {"c", 2}}}}},
	{&struct{ V MyD }{MyD{{"a", 1}}}, &bson.D{{"v", bson.D{{"a", 1}}}}},

	// bson.RawD <=> []RawDocElem
	{&bson.RawD{{"a", bson.Raw{0x08, []byte{0x01}}}}, &bson.RawD{{"a", bson.Raw{0x08, []byte{0x01}}}}},
	{&bson.RawD{{"a", bson.Raw{0x08, []byte{0x01}}}}, &MyRawD{{"a", bson.Raw{0x08, []byte{0x01}}}}},

	// bson.M <=> map
	{bson.M{"a": bson.M{"b": 1, "c": 2}}, MyM{"a": MyM{"b": 1, "c": 2}}},
	{bson.M{"a": bson.M{"b": 1, "c": 2}}, map[string]interface{}{"a": map[string]interface{}{"b": 1, "c": 2}}},

	// bson.M <=> map[MyString]
	{bson.M{"a": bson.M{"b": 1, "c": 2}}, map[MyString]interface{}{"a": map[MyString]interface{}{"b": 1, "c": 2}}},

	// json.Number <=> int64, float64
	{&struct{ N json.Number }{"5"}, map[string]interface{}{"n": int64(5)}},
	{&struct{ N json.Number }{"5.05"}, map[string]interface{}{"n": 5.05}},
	{&struct{ N json.Number }{"9223372036854776000"}, map[string]interface{}{"n": float64(1 << 63)}},

	// bson.D <=> non-struct getter/setter
	{&bson.D{{"a", 1}}, &getterSetterD{{"a", 1}, {"suffix", true}}},
	{&bson.D{{"a", 42}}, &gsintvar},

	// Interface slice setter.
	{&struct{ V ifaceSlice }{ifaceSlice{nil, nil, nil}}, bson.M{"v": []interface{}{3}}},
}

// Same thing, but only one way (obj1 => obj2).
var oneWayCrossItems = []crossTypeItem{
	// map <=> struct
	{map[string]interface{}{"a": 1, "b": "2", "c": 3}, map[string]int{"a": 1, "c": 3}},

	// inline map elides badly typed values
	{map[string]interface{}{"a": 1, "b": "2", "c": 3}, &inlineMapInt{A: 1, M: map[string]int{"c": 3}}},

	// Can't decode int into struct.
	{bson.M{"a": bson.M{"b": 2}}, &struct{ A bool }{}},

	// Would get decoded into a int32 too in the opposite direction.
	{&shortIface{int64(1) << 30}, map[string]interface{}{"v": 1 << 30}},

	// Ensure omitempty on struct with private fields works properly.
	{&struct {
		V struct{ v time.Time } `bson:",omitempty"`
	}{}, map[string]interface{}{}},

	// Attempt to marshal slice into RawD (issue #120).
	{bson.M{"x": []int{1, 2, 3}}, &struct{ X bson.RawD }{}},
}

func testCrossPair(t *testing.T, dump interface{}, load interface{}) {
	zero := makeZeroDoc(load)
	data, err := bson.Marshal(dump)
	require.NoError(t, err)
	err = bson.Unmarshal(data, zero)
	require.NoError(t, err)
	require.Equal(t, zero, load)
}

func TestTwoWayCrossPairs(t *testing.T) {
	for i, item := range twoWayCrossItems {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			testCrossPair(t, item.obj1, item.obj2)
			testCrossPair(t, item.obj2, item.obj1)
		})
	}
}

func TestOneWayCrossPairs(t *testing.T) {
	for i, item := range oneWayCrossItems {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			testCrossPair(t, item.obj1, item.obj2)
		})
	}
}

// --------------------------------------------------------------------------
// ObjectId hex representation test.

func TestObjectIdHex(t *testing.T) {
	id := bson.ObjectIdHex("4d88e15b60f486e428412dc9")
	require.Equal(t, id.String(), `ObjectIdHex("4d88e15b60f486e428412dc9")`)
	require.Equal(t, id.Hex(), "4d88e15b60f486e428412dc9")
}

func TestIsObjectIdHex(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"4d88e15b60f486e428412dc9", true},
		{"4d88e15b60f486e428412dc", false},
		{"4d88e15b60f486e428412dc9e", false},
		{"4d88e15b60f486e428412dcx", false},
	}
	for _, test := range tests {
		t.Run(test.id, func(t *testing.T) {
			require.Equal(t, bson.IsObjectIdHex(test.id), test.valid)
		})
	}
}

// --------------------------------------------------------------------------
// ObjectId parts extraction tests.

type objectIdParts struct {
	id        bson.ObjectId
	timestamp int64
	machine   []byte
	pid       uint16
	counter   int32
}

var objectIds = []objectIdParts{
	objectIdParts{
		bson.ObjectIdHex("4d88e15b60f486e428412dc9"),
		1300816219,
		[]byte{0x60, 0xf4, 0x86},
		0xe428,
		4271561,
	},
	objectIdParts{
		bson.ObjectIdHex("000000000000000000000000"),
		0,
		[]byte{0x00, 0x00, 0x00},
		0x0000,
		0,
	},
	objectIdParts{
		bson.ObjectIdHex("00000000aabbccddee000001"),
		0,
		[]byte{0xaa, 0xbb, 0xcc},
		0xddee,
		1,
	},
}

func TestObjectIdPartsExtraction(t *testing.T) {
	for _, v := range objectIds {
		t.Run(v.id.String(), func(t *testing.T) {
			ti := time.Unix(v.timestamp, 0)
			require.Equal(t, v.id.Time(), ti)
			require.Equal(t, v.id.Machine(), v.machine)
			require.Equal(t, v.id.Pid(), v.pid)
			require.Equal(t, v.id.Counter(), v.counter)
		})
	}
}

func TestNow(t *testing.T) {
	before := time.Now()
	time.Sleep(1e6)
	now := bson.Now()
	time.Sleep(1e6)
	after := time.Now()
	require.True(t, now.After(before) && now.Before(after))
}

// --------------------------------------------------------------------------
// ObjectId generation tests.

func TestNewObjectId(t *testing.T) {
	// Generate 10 ids
	ids := make([]bson.ObjectId, 10)
	for i := 0; i < 10; i++ {
		ids[i] = bson.NewObjectId()
	}
	for i := 1; i < 10; i++ {
		t.Run(ids[i].String(), func(t *testing.T) {
			prevId := ids[i-1]
			id := ids[i]
			// Test for uniqueness among all other 9 generated ids
			for j, tid := range ids {
				if j != i {
					require.NotEqual(t, id, tid)
				}
			}
			// Check that timestamp was incremented and is within 30 seconds of the previous one
			secs := id.Time().Sub(prevId.Time()).Seconds()
			require.True(t, secs >= 0 && secs <= 30)
			// Check that machine ids are the same
			require.Equal(t, id.Machine(), prevId.Machine())
			// Check that pids are the same
			require.Equal(t, id.Pid(), prevId.Pid())
			// Test for proper increment
			delta := int(id.Counter() - prevId.Counter())
			require.Equal(t, delta, 1)
		})
	}
}

func TestNewObjectIdWithTime(t *testing.T) {
	ti := time.Unix(12345678, 0)
	id := bson.NewObjectIdWithTime(ti)
	require.Equal(t, id.Time(), ti)
	require.Equal(t, id.Machine(), []byte{0x00, 0x00, 0x00})
	require.Equal(t, int(id.Pid()), 0)
	require.Equal(t, int(id.Counter()), 0)
}

// --------------------------------------------------------------------------
// ObjectId JSON marshalling.

type jsonType struct {
	Id bson.ObjectId
}

var jsonIdTests = []struct {
	value     jsonType
	json      string
	marshal   bool
	unmarshal bool
	error     string
}{{
	value:     jsonType{Id: bson.ObjectIdHex("4d88e15b60f486e428412dc9")},
	json:      `{"Id":"4d88e15b60f486e428412dc9"}`,
	marshal:   true,
	unmarshal: true,
}, {
	value:     jsonType{},
	json:      `{"Id":""}`,
	marshal:   true,
	unmarshal: true,
}, {
	value:     jsonType{},
	json:      `{"Id":null}`,
	marshal:   false,
	unmarshal: true,
}, {
	json:      `{"Id":"4d88e15b60f486e428412dc9A"}`,
	error:     `invalid ObjectId in JSON: "4d88e15b60f486e428412dc9A"`,
	marshal:   false,
	unmarshal: true,
}, {
	json:      `{"Id":"4d88e15b60f486e428412dcZ"}`,
	error:     `invalid ObjectId in JSON: "4d88e15b60f486e428412dcZ" (encoding/hex: invalid byte: U+005A 'Z')`,
	marshal:   false,
	unmarshal: true,
}}

func TestObjectIdJSONMarshaling(t *testing.T) {
	for _, test := range jsonIdTests {
		if test.marshal {
			t.Run("Marshal-"+test.json, func(t *testing.T) {
				data, err := json.Marshal(&test.value)
				if test.error == "" {
					require.NoError(t, err)
					require.Equal(t, string(data), test.json)
				} else {
					require.EqualError(t, err, test.error)
				}
			})
		}

		if test.unmarshal {
			t.Run("Unmarshal-"+test.json, func(t *testing.T) {
				var value jsonType
				err := json.Unmarshal([]byte(test.json), &value)
				if test.error == "" {
					require.NoError(t, err)
					require.Equal(t, value, test.value)
				} else {
					require.EqualError(t, err, test.error)
				}
			})
		}
	}
}

// --------------------------------------------------------------------------
// ObjectId Text encoding.TextUnmarshaler.

var textIdTests = []struct {
	value     bson.ObjectId
	text      string
	marshal   bool
	unmarshal bool
	error     string
}{{
	value:     bson.ObjectIdHex("4d88e15b60f486e428412dc9"),
	text:      "4d88e15b60f486e428412dc9",
	marshal:   true,
	unmarshal: true,
}, {
	text:      "",
	marshal:   true,
	unmarshal: true,
}, {
	text:      "4d88e15b60f486e428412dc9A",
	marshal:   false,
	unmarshal: true,
	error:     `invalid ObjectId: 4d88e15b60f486e428412dc9A`,
}, {
	text:      "4d88e15b60f486e428412dcZ",
	marshal:   false,
	unmarshal: true,
	error:     `invalid ObjectId: 4d88e15b60f486e428412dcZ (encoding/hex: invalid byte: U+005A 'Z')`,
}}

func TestObjectIdTextMarshaling(t *testing.T) {
	for _, test := range textIdTests {
		if test.marshal {
			t.Run("Marshal-"+test.text, func(t *testing.T) {
				data, err := test.value.MarshalText()
				if test.error == "" {
					require.NoError(t, err)
					require.Equal(t, string(data), test.text)
				} else {
					require.EqualError(t, err, test.error)
				}
			})
		}

		if test.unmarshal {
			t.Run("Unmarshal-"+test.text, func(t *testing.T) {
				err := test.value.UnmarshalText([]byte(test.text))
				if test.error == "" {
					require.NoError(t, err)
					if test.value != "" {
						value := bson.ObjectIdHex(test.text)
						require.Equal(t, value, test.value)
					}
				} else {
					require.EqualError(t, err, test.error)
				}
			})
		}
	}
}

// --------------------------------------------------------------------------
// ObjectId XML marshalling.

type xmlType struct {
	Id bson.ObjectId
}

var xmlIdTests = []struct {
	value     xmlType
	xml       string
	marshal   bool
	unmarshal bool
	error     string
}{{
	value:     xmlType{Id: bson.ObjectIdHex("4d88e15b60f486e428412dc9")},
	xml:       "<xmlType><Id>4d88e15b60f486e428412dc9</Id></xmlType>",
	marshal:   true,
	unmarshal: true,
}, {
	value:     xmlType{},
	xml:       "<xmlType><Id></Id></xmlType>",
	marshal:   true,
	unmarshal: true,
}, {
	xml:       "<xmlType><Id>4d88e15b60f486e428412dc9A</Id></xmlType>",
	marshal:   false,
	unmarshal: true,
	error:     `invalid ObjectId: 4d88e15b60f486e428412dc9A`,
}, {
	xml:       "<xmlType><Id>4d88e15b60f486e428412dcZ</Id></xmlType>",
	marshal:   false,
	unmarshal: true,
	error:     `invalid ObjectId: 4d88e15b60f486e428412dcZ (encoding/hex: invalid byte: U+005A 'Z')`,
}}

func TestObjectIdXMLMarshaling(t *testing.T) {
	for _, test := range xmlIdTests {
		if test.marshal {
			t.Run("Marshal-"+test.xml, func(t *testing.T) {
				data, err := xml.Marshal(&test.value)
				if test.error == "" {
					require.NoError(t, err)
					require.Equal(t, string(data), test.xml)
				} else {
					require.EqualError(t, err, test.error)
				}
			})
		}

		if test.unmarshal {
			t.Run("Unarshal-"+test.xml, func(t *testing.T) {
				var value xmlType
				err := xml.Unmarshal([]byte(test.xml), &value)
				if test.error == "" {
					require.NoError(t, err)
					require.Equal(t, value, test.value)
				} else {
					require.EqualError(t, err, test.error)
				}
			})
		}
	}
}

// --------------------------------------------------------------------------
// Some simple benchmarks.

type BenchT struct {
	A, B, C, D, E, F string
}

type BenchRawT struct {
	A string
	B int
	C bson.M
	D []float64
}

func BenchmarkUnmarhsalStruct(b *testing.B) {
	v := BenchT{A: "A", D: "D", E: "E"}
	data, err := bson.Marshal(&v)
	if err != nil {
		panic(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = bson.Unmarshal(data, &v)
	}
	if err != nil {
		panic(err)
	}
}

func BenchmarkUnmarhsalMap(b *testing.B) {
	m := bson.M{"a": "a", "d": "d", "e": "e"}
	data, err := bson.Marshal(&m)
	if err != nil {
		panic(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = bson.Unmarshal(data, &m)
	}
	if err != nil {
		panic(err)
	}
}

func BenchmarkUnmarshalRaw(b *testing.B) {
	var err error
	m := BenchRawT{
		A: "test_string",
		B: 123,
		C: bson.M{
			"subdoc_int": 12312,
			"subdoc_doc": bson.M{"1": 1},
		},
		D: []float64{0.0, 1.3333, -99.9997, 3.1415},
	}
	data, err := bson.Marshal(&m)
	if err != nil {
		panic(err)
	}
	raw := bson.Raw{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = bson.Unmarshal(data, &raw)
	}
	if err != nil {
		panic(err)
	}
}

func BenchmarkNewObjectId(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bson.NewObjectId()
	}
}
