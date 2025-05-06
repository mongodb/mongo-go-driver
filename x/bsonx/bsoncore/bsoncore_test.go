// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
)

func compareErrors(err1, err2 error) bool {
	if err1 == nil && err2 == nil {
		return true
	}

	if err1 == nil || err2 == nil {
		return false
	}

	if err1.Error() != err2.Error() {
		return false
	}

	return true
}

func TestAppend(t *testing.T) {
	bits := math.Float64bits(3.14159)
	pi := make([]byte, 8)
	binary.LittleEndian.PutUint64(pi, bits)

	testCases := []struct {
		name     string
		fn       interface{}
		params   []interface{}
		expected []byte
	}{
		{
			"AppendType",
			AppendType,
			[]interface{}{make([]byte, 0), TypeNull},
			[]byte{byte(TypeNull)},
		},
		{
			"AppendKey",
			AppendKey,
			[]interface{}{make([]byte, 0), "foobar"},
			[]byte{'f', 'o', 'o', 'b', 'a', 'r', 0x00},
		},
		{
			"AppendHeader",
			AppendHeader,
			[]interface{}{make([]byte, 0), TypeNull, "foobar"},
			[]byte{byte(TypeNull), 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
		},
		{
			"AppendValueElement",
			AppendValueElement,
			[]interface{}{make([]byte, 0), "testing", Value{Type: TypeBoolean, Data: []byte{0x01}}},
			[]byte{byte(TypeBoolean), 't', 'e', 's', 't', 'i', 'n', 'g', 0x00, 0x01},
		},
		{
			"AppendDouble",
			AppendDouble,
			[]interface{}{make([]byte, 0), float64(3.14159)},
			pi,
		},
		{
			"AppendDoubleElement",
			AppendDoubleElement,
			[]interface{}{make([]byte, 0), "foobar", float64(3.14159)},
			append([]byte{byte(TypeDouble), 'f', 'o', 'o', 'b', 'a', 'r', 0x00}, pi...),
		},
		{
			"AppendString",
			AppendString,
			[]interface{}{make([]byte, 0), "barbaz"},
			[]byte{0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00},
		},
		{
			"AppendStringElement",
			AppendStringElement,
			[]interface{}{make([]byte, 0), "foobar", "barbaz"},
			[]byte{byte(TypeString),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00,
			},
		},
		{
			"AppendDocument",
			AppendDocument,
			[]interface{}{[]byte{0x05, 0x00, 0x00, 0x00, 0x00}, []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			[]byte{0x05, 0x00, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00},
		},
		{
			"AppendDocumentElement",
			AppendDocumentElement,
			[]interface{}{make([]byte, 0), "foobar", []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			[]byte{byte(TypeEmbeddedDocument),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x05, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			"AppendArray",
			AppendArray,
			[]interface{}{[]byte{0x05, 0x00, 0x00, 0x00, 0x00}, []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			[]byte{0x05, 0x00, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00},
		},
		{
			"AppendArrayElement",
			AppendArrayElement,
			[]interface{}{make([]byte, 0), "foobar", []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			[]byte{byte(TypeArray),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x05, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			"BuildArray",
			BuildArray,
			[]interface{}{make([]byte, 0), Value{Type: TypeDouble, Data: AppendDouble(nil, 3.14159)}},
			[]byte{
				0x10, 0x00, 0x00, 0x00,
				byte(TypeDouble), '0', 0x00,
				pi[0], pi[1], pi[2], pi[3], pi[4], pi[5], pi[6], pi[7],
				0x00,
			},
		},
		{
			"BuildArrayElement",
			BuildArrayElement,
			[]interface{}{make([]byte, 0), "foobar", Value{Type: TypeDouble, Data: AppendDouble(nil, 3.14159)}},
			[]byte{byte(TypeArray),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x10, 0x00, 0x00, 0x00,
				byte(TypeDouble), '0', 0x00,
				pi[0], pi[1], pi[2], pi[3], pi[4], pi[5], pi[6], pi[7],
				0x00,
			},
		},
		{
			"AppendBinary Subtype2",
			AppendBinary,
			[]interface{}{make([]byte, 0), byte(0x02), []byte{0x01, 0x02, 0x03}},
			[]byte{0x07, 0x00, 0x00, 0x00, 0x02, 0x03, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
		},
		{
			"AppendBinaryElement Subtype 2",
			AppendBinaryElement,
			[]interface{}{make([]byte, 0), "foobar", byte(0x02), []byte{0x01, 0x02, 0x03}},
			[]byte{byte(TypeBinary),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x07, 0x00, 0x00, 0x00,
				0x02,
				0x03, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03,
			},
		},
		{
			"AppendBinary",
			AppendBinary,
			[]interface{}{make([]byte, 0), byte(0xFF), []byte{0x01, 0x02, 0x03}},
			[]byte{0x03, 0x00, 0x00, 0x00, 0xFF, 0x01, 0x02, 0x03},
		},
		{
			"AppendBinaryElement",
			AppendBinaryElement,
			[]interface{}{make([]byte, 0), "foobar", byte(0xFF), []byte{0x01, 0x02, 0x03}},
			[]byte{byte(TypeBinary),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x03, 0x00, 0x00, 0x00,
				0xFF,
				0x01, 0x02, 0x03,
			},
		},
		{
			"AppendUndefinedElement",
			AppendUndefinedElement,
			[]interface{}{make([]byte, 0), "foobar"},
			[]byte{byte(TypeUndefined), 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
		},
		{
			"AppendObjectID",
			AppendObjectID,
			[]interface{}{
				make([]byte, 0),
				[12]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			},
			[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
		},
		{
			"AppendObjectIDElement",
			AppendObjectIDElement,
			[]interface{}{
				make([]byte, 0), "foobar",
				[12]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			},
			[]byte{byte(TypeObjectID),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C,
			},
		},
		{
			"AppendBoolean (true)",
			AppendBoolean,
			[]interface{}{make([]byte, 0), true},
			[]byte{0x01},
		},
		{
			"AppendBoolean (false)",
			AppendBoolean,
			[]interface{}{make([]byte, 0), false},
			[]byte{0x00},
		},
		{
			"AppendBooleanElement",
			AppendBooleanElement,
			[]interface{}{make([]byte, 0), "foobar", true},
			[]byte{byte(TypeBoolean), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x01},
		},
		{
			"AppendDateTime",
			AppendDateTime,
			[]interface{}{make([]byte, 0), int64(256)},
			[]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			"AppendDateTimeElement",
			AppendDateTimeElement,
			[]interface{}{make([]byte, 0), "foobar", int64(256)},
			[]byte{byte(TypeDateTime), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			"AppendNullElement",
			AppendNullElement,
			[]interface{}{make([]byte, 0), "foobar"},
			[]byte{byte(TypeNull), 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
		},
		{
			"AppendRegex",
			AppendRegex,
			[]interface{}{make([]byte, 0), "bar", "baz"},
			[]byte{'b', 'a', 'r', 0x00, 'b', 'a', 'z', 0x00},
		},
		{
			"AppendRegexElement",
			AppendRegexElement,
			[]interface{}{make([]byte, 0), "foobar", "bar", "baz"},
			[]byte{byte(TypeRegex),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				'b', 'a', 'r', 0x00, 'b', 'a', 'z', 0x00,
			},
		},
		{
			"AppendDBPointer",
			AppendDBPointer,
			[]interface{}{
				make([]byte, 0),
				"foobar",
				[12]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			},
			[]byte{
				0x07, 0x00, 0x00, 0x00, 'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C,
			},
		},
		{
			"AppendDBPointerElement",
			AppendDBPointerElement,
			[]interface{}{
				make([]byte, 0), "foobar",
				"barbaz",
				[12]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			},
			[]byte{byte(TypeDBPointer),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00,
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C,
			},
		},
		{
			"AppendJavaScript",
			AppendJavaScript,
			[]interface{}{make([]byte, 0), "barbaz"},
			[]byte{0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00},
		},
		{
			"AppendJavaScriptElement",
			AppendJavaScriptElement,
			[]interface{}{make([]byte, 0), "foobar", "barbaz"},
			[]byte{byte(TypeJavaScript),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00,
			},
		},
		{
			"AppendSymbol",
			AppendSymbol,
			[]interface{}{make([]byte, 0), "barbaz"},
			[]byte{0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00},
		},
		{
			"AppendSymbolElement",
			AppendSymbolElement,
			[]interface{}{make([]byte, 0), "foobar", "barbaz"},
			[]byte{byte(TypeSymbol),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00,
			},
		},
		{
			"AppendCodeWithScope",
			AppendCodeWithScope,
			[]interface{}{[]byte{0x05, 0x00, 0x00, 0x00, 0x00}, "foobar", []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			[]byte{0x05, 0x00, 0x00, 0x00, 0x00,
				0x14, 0x00, 0x00, 0x00,
				0x07, 0x00, 0x00, 0x00, 'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x05, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			"AppendCodeWithScopeElement",
			AppendCodeWithScopeElement,
			[]interface{}{make([]byte, 0), "foobar", "barbaz", []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			[]byte{byte(TypeCodeWithScope),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x14, 0x00, 0x00, 0x00,
				0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00,
				0x05, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			"AppendInt32",
			AppendInt32,
			[]interface{}{make([]byte, 0), int32(256)},
			[]byte{0x00, 0x01, 0x00, 0x00},
		},
		{
			"AppendInt32Element",
			AppendInt32Element,
			[]interface{}{make([]byte, 0), "foobar", int32(256)},
			[]byte{byte(TypeInt32), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x00, 0x01, 0x00, 0x00},
		},
		{
			"AppendTimestamp",
			AppendTimestamp,
			[]interface{}{make([]byte, 0), uint32(65536), uint32(256)},
			[]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00},
		},
		{
			"AppendTimestampElement",
			AppendTimestampElement,
			[]interface{}{make([]byte, 0), "foobar", uint32(65536), uint32(256)},
			[]byte{byte(TypeTimestamp), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00},
		},
		{
			"AppendInt64",
			AppendInt64,
			[]interface{}{make([]byte, 0), int64(4294967296)},
			[]byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00},
		},
		{
			"AppendInt64Element",
			AppendInt64Element,
			[]interface{}{make([]byte, 0), "foobar", int64(4294967296)},
			[]byte{byte(TypeInt64), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00},
		},
		{
			"AppendDecimal128",
			AppendDecimal128,
			[]interface{}{make([]byte, 0), uint64(4294967296), uint64(65536)},
			[]byte{
				0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
			},
		},
		{
			"AppendDecimal128Element",
			AppendDecimal128Element,
			[]interface{}{make([]byte, 0), "foobar", uint64(4294967296), uint64(65536)},
			[]byte{
				byte(TypeDecimal128), 'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
			},
		},
		{
			"AppendMaxKeyElement",
			AppendMaxKeyElement,
			[]interface{}{make([]byte, 0), "foobar"},
			[]byte{byte(TypeMaxKey), 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
		},
		{
			"AppendMinKeyElement",
			AppendMinKeyElement,
			[]interface{}{make([]byte, 0), "foobar"},
			[]byte{byte(TypeMinKey), 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fn := reflect.ValueOf(tc.fn)
			if fn.Kind() != reflect.Func {
				t.Fatalf("fn must be of kind Func but is a %v", fn.Kind())
			}
			if fn.Type().NumIn() != len(tc.params) {
				t.Fatalf("tc.params must match the number of params in tc.fn. params %d; fn %d", fn.Type().NumIn(), len(tc.params))
			}
			if fn.Type().NumOut() != 1 || fn.Type().Out(0) != reflect.TypeOf([]byte{}) {
				t.Fatalf("fn must have one return parameter and it must be a []byte.")
			}
			params := make([]reflect.Value, 0, len(tc.params))
			for _, param := range tc.params {
				params = append(params, reflect.ValueOf(param))
			}
			results := fn.Call(params)
			got := results[0].Interface().([]byte)
			want := tc.expected
			if !bytes.Equal(got, want) {
				t.Errorf("Did not receive expected bytes. got %v; want %v", got, want)
			}
		})
	}
}

func TestRead(t *testing.T) {
	bits := math.Float64bits(3.14159)
	pi := make([]byte, 8)
	binary.LittleEndian.PutUint64(pi, bits)

	testCases := []struct {
		name     string
		fn       interface{}
		param    []byte
		expected []interface{}
	}{
		{
			"ReadType/not enough bytes",
			ReadType,
			[]byte{},
			[]interface{}{Type(0), []byte{}, false},
		},
		{
			"ReadType/success",
			ReadType,
			[]byte{0x0A},
			[]interface{}{TypeNull, []byte{}, true},
		},
		{
			"ReadKey/not enough bytes",
			ReadKey,
			[]byte{},
			[]interface{}{"", []byte{}, false},
		},
		{
			"ReadKey/success",
			ReadKey,
			[]byte{'f', 'o', 'o', 'b', 'a', 'r', 0x00},
			[]interface{}{"foobar", []byte{}, true},
		},
		{
			"ReadHeader/not enough bytes (type)",
			ReadHeader,
			[]byte{},
			[]interface{}{Type(0), "", []byte{}, false},
		},
		{
			"ReadHeader/not enough bytes (key)",
			ReadHeader,
			[]byte{0x0A, 'f', 'o', 'o'},
			[]interface{}{Type(0), "", []byte{0x0A, 'f', 'o', 'o'}, false},
		},
		{
			"ReadHeader/success",
			ReadHeader,
			[]byte{0x0A, 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
			[]interface{}{TypeNull, "foobar", []byte{}, true},
		},
		{
			"ReadDouble/not enough bytes",
			ReadDouble,
			[]byte{0x01, 0x02, 0x03, 0x04},
			[]interface{}{float64(0.00), []byte{0x01, 0x02, 0x03, 0x04}, false},
		},
		{
			"ReadDouble/success",
			ReadDouble,
			pi,
			[]interface{}{float64(3.14159), []byte{}, true},
		},
		{
			"ReadString/not enough bytes (length)",
			ReadString,
			[]byte{},
			[]interface{}{"", []byte{}, false},
		},
		{
			"ReadString/not enough bytes (value)",
			ReadString,
			[]byte{0x0F, 0x00, 0x00, 0x00},
			[]interface{}{"", []byte{0x0F, 0x00, 0x00, 0x00}, false},
		},
		{
			"ReadString/success",
			ReadString,
			[]byte{0x07, 0x00, 0x00, 0x00, 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
			[]interface{}{"foobar", []byte{}, true},
		},
		{
			"ReadDocument/not enough bytes (length)",
			ReadDocument,
			[]byte{},
			[]interface{}{Document(nil), []byte{}, false},
		},
		{
			"ReadDocument/not enough bytes (value)",
			ReadDocument,
			[]byte{0x0F, 0x00, 0x00, 0x00},
			[]interface{}{Document(nil), []byte{0x0F, 0x00, 0x00, 0x00}, false},
		},
		{
			"ReadDocument/success",
			ReadDocument,
			[]byte{0x0A, 0x00, 0x00, 0x00, 0x0A, 'f', 'o', 'o', 0x00, 0x00},
			[]interface{}{Document{0x0A, 0x00, 0x00, 0x00, 0x0A, 'f', 'o', 'o', 0x00, 0x00}, []byte{}, true},
		},
		{
			"ReadArray/not enough bytes (length)",
			ReadArray,
			[]byte{},
			[]interface{}{Array(nil), []byte{}, false},
		},
		{
			"ReadArray/not enough bytes (value)",
			ReadArray,
			[]byte{0x0F, 0x00, 0x00, 0x00},
			[]interface{}{Array(nil), []byte{0x0F, 0x00, 0x00, 0x00}, false},
		},
		{
			"ReadArray/success",
			ReadArray,
			[]byte{0x08, 0x00, 0x00, 0x00, 0x0A, '0', 0x00, 0x00},
			[]interface{}{Array{0x08, 0x00, 0x00, 0x00, 0x0A, '0', 0x00, 0x00}, []byte{}, true},
		},
		{
			"ReadBinary/not enough bytes (length)",
			ReadBinary,
			[]byte{},
			[]interface{}{byte(0), []byte(nil), []byte{}, false},
		},
		{
			"ReadBinary/not enough bytes (subtype)",
			ReadBinary,
			[]byte{0x0F, 0x00, 0x00, 0x00},
			[]interface{}{byte(0), []byte(nil), []byte{0x0F, 0x00, 0x00, 0x00}, false},
		},
		{
			"ReadBinary/not enough bytes (value)",
			ReadBinary,
			[]byte{0x0F, 0x00, 0x00, 0x00, 0x00},
			[]interface{}{byte(0), []byte(nil), []byte{0x0F, 0x00, 0x00, 0x00, 0x00}, false},
		},
		{
			"ReadBinary/not enough bytes (subtype 2 length)",
			ReadBinary,
			[]byte{0x03, 0x00, 0x00, 0x00, 0x02, 0x0F, 0x00, 0x00},
			[]interface{}{byte(0), []byte(nil), []byte{0x03, 0x00, 0x00, 0x00, 0x02, 0x0F, 0x00, 0x00}, false},
		},
		{
			"ReadBinary/not enough bytes (subtype 2 value)",
			ReadBinary,
			[]byte{0x0F, 0x00, 0x00, 0x00, 0x02, 0x0F, 0x00, 0x00, 0x00, 0x01, 0x02},
			[]interface{}{
				byte(0), []byte(nil),
				[]byte{0x0F, 0x00, 0x00, 0x00, 0x02, 0x0F, 0x00, 0x00, 0x00, 0x01, 0x02}, false,
			},
		},
		{
			"ReadBinary/success (subtype 2)",
			ReadBinary,
			[]byte{0x06, 0x00, 0x00, 0x00, 0x02, 0x02, 0x00, 0x00, 0x00, 0x01, 0x02},
			[]interface{}{byte(0x02), []byte{0x01, 0x02}, []byte{}, true},
		},
		{
			"ReadBinary/success",
			ReadBinary,
			[]byte{0x03, 0x00, 0x00, 0x00, 0xFF, 0x01, 0x02, 0x03},
			[]interface{}{byte(0xFF), []byte{0x01, 0x02, 0x03}, []byte{}, true},
		},
		{
			"ReadObjectID/not enough bytes",
			ReadObjectID,
			[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
			[]interface{}{[12]byte{}, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}, false},
		},
		{
			"ReadObjectID/success",
			ReadObjectID,
			[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			[]interface{}{
				[12]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
				[]byte{}, true,
			},
		},
		{
			"ReadBoolean/not enough bytes",
			ReadBoolean,
			[]byte{},
			[]interface{}{false, []byte{}, false},
		},
		{
			"ReadBoolean/success",
			ReadBoolean,
			[]byte{0x01},
			[]interface{}{true, []byte{}, true},
		},
		{
			"ReadDateTime/not enough bytes",
			ReadDateTime,
			[]byte{0x01, 0x02, 0x03, 0x04},
			[]interface{}{int64(0), []byte{0x01, 0x02, 0x03, 0x04}, false},
		},
		{
			"ReadDateTime/success",
			ReadDateTime,
			[]byte{0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00},
			[]interface{}{int64(65536), []byte{}, true},
		},
		{
			"ReadRegex/not enough bytes (pattern)",
			ReadRegex,
			[]byte{},
			[]interface{}{"", "", []byte{}, false},
		},
		{
			"ReadRegex/not enough bytes (options)",
			ReadRegex,
			[]byte{'f', 'o', 'o', 0x00},
			[]interface{}{"", "", []byte{'f', 'o', 'o', 0x00}, false},
		},
		{
			"ReadRegex/success",
			ReadRegex,
			[]byte{'f', 'o', 'o', 0x00, 'b', 'a', 'r', 0x00},
			[]interface{}{"foo", "bar", []byte{}, true},
		},
		{
			"ReadDBPointer/not enough bytes (ns)",
			ReadDBPointer,
			[]byte{},
			[]interface{}{"", [12]byte{}, []byte{}, false},
		},
		{
			"ReadDBPointer/not enough bytes (objectID)",
			ReadDBPointer,
			[]byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00},
			[]interface{}{"", [12]byte{}, []byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00}, false},
		},
		{
			"ReadDBPointer/success",
			ReadDBPointer,
			[]byte{
				0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00,
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C,
			},
			[]interface{}{
				"foo", [12]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
				[]byte{}, true,
			},
		},
		{
			"ReadJavaScript/not enough bytes (length)",
			ReadJavaScript,
			[]byte{},
			[]interface{}{"", []byte{}, false},
		},
		{
			"ReadJavaScript/not enough bytes (value)",
			ReadJavaScript,
			[]byte{0x0F, 0x00, 0x00, 0x00},
			[]interface{}{"", []byte{0x0F, 0x00, 0x00, 0x00}, false},
		},
		{
			"ReadJavaScript/success",
			ReadJavaScript,
			[]byte{0x07, 0x00, 0x00, 0x00, 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
			[]interface{}{"foobar", []byte{}, true},
		},
		{
			"ReadSymbol/not enough bytes (length)",
			ReadSymbol,
			[]byte{},
			[]interface{}{"", []byte{}, false},
		},
		{
			"ReadSymbol/not enough bytes (value)",
			ReadSymbol,
			[]byte{0x0F, 0x00, 0x00, 0x00},
			[]interface{}{"", []byte{0x0F, 0x00, 0x00, 0x00}, false},
		},
		{
			"ReadSymbol/success",
			ReadSymbol,
			[]byte{0x07, 0x00, 0x00, 0x00, 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
			[]interface{}{"foobar", []byte{}, true},
		},
		{
			"ReadCodeWithScope/ not enough bytes (length)",
			ReadCodeWithScope,
			[]byte{},
			[]interface{}{"", []byte(nil), []byte{}, false},
		},
		{
			"ReadCodeWithScope/ not enough bytes (value)",
			ReadCodeWithScope,
			[]byte{0x0F, 0x00, 0x00, 0x00},
			[]interface{}{"", []byte(nil), []byte{0x0F, 0x00, 0x00, 0x00}, false},
		},
		{
			"ReadCodeWithScope/not enough bytes (code value)",
			ReadCodeWithScope,
			[]byte{
				0x0C, 0x00, 0x00, 0x00,
				0x0F, 0x00, 0x00, 0x00,
				'f', 'o', 'o', 0x00,
			},
			[]interface{}{
				"", []byte(nil),
				[]byte{
					0x0C, 0x00, 0x00, 0x00,
					0x0F, 0x00, 0x00, 0x00,
					'f', 'o', 'o', 0x00,
				},
				false,
			},
		},
		{
			"ReadCodeWithScope/success",
			ReadCodeWithScope,
			[]byte{
				0x19, 0x00, 0x00, 0x00,
				0x07, 0x00, 0x00, 0x00, 'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x0A, 0x00, 0x00, 0x00, 0x0A, 'f', 'o', 'o', 0x00, 0x00,
			},
			[]interface{}{
				"foobar", []byte{0x0A, 0x00, 0x00, 0x00, 0x0A, 'f', 'o', 'o', 0x00, 0x00},
				[]byte{}, true,
			},
		},
		{
			"ReadInt32/not enough bytes",
			ReadInt32,
			[]byte{0x01},
			[]interface{}{int32(0), []byte{0x01}, false},
		},
		{
			"ReadInt32/success",
			ReadInt32,
			[]byte{0x00, 0x01, 0x00, 0x00},
			[]interface{}{int32(256), []byte{}, true},
		},
		{
			"ReadTimestamp/not enough bytes (increment)",
			ReadTimestamp,
			[]byte{},
			[]interface{}{uint32(0), uint32(0), []byte{}, false},
		},
		{
			"ReadTimestamp/not enough bytes (timestamp)",
			ReadTimestamp,
			[]byte{0x00, 0x01, 0x00, 0x00},
			[]interface{}{uint32(0), uint32(0), []byte{0x00, 0x01, 0x00, 0x00}, false},
		},
		{
			"ReadTimestamp/success",
			ReadTimestamp,
			[]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00},
			[]interface{}{uint32(65536), uint32(256), []byte{}, true},
		},
		{
			"ReadInt64/not enough bytes",
			ReadInt64,
			[]byte{0x01},
			[]interface{}{int64(0), []byte{0x01}, false},
		},
		{
			"ReadInt64/success",
			ReadInt64,
			[]byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00},
			[]interface{}{int64(4294967296), []byte{}, true},
		},
		{
			"ReadDecimal128/not enough bytes (low)",
			ReadDecimal128,
			[]byte{},
			[]interface{}{uint64(0), uint64(0), []byte{}, false},
		},
		{
			"ReadDecimal128/not enough bytes (high)",
			ReadDecimal128,
			[]byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00},
			[]interface{}{uint64(0), uint64(0), []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00}, false},
		},
		{
			"ReadDecimal128/success",
			ReadDecimal128,
			[]byte{
				0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
			},
			[]interface{}{uint64(4294967296), uint64(16777216), []byte{}, true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fn := reflect.ValueOf(tc.fn)
			if fn.Kind() != reflect.Func {
				t.Fatalf("fn must be of kind Func but it is a %v", fn.Kind())
			}
			if fn.Type().NumIn() != 1 || fn.Type().In(0) != reflect.TypeOf([]byte{}) {
				t.Fatalf("fn must have one parameter and it must be a []byte.")
			}
			results := fn.Call([]reflect.Value{reflect.ValueOf(tc.param)})
			if len(results) != len(tc.expected) {
				t.Fatalf("Length of results does not match. got %d; want %d", len(results), len(tc.expected))
			}
			for idx := range results {
				got := results[idx].Interface()
				want := tc.expected[idx]
				if !cmp.Equal(got, want) {
					t.Errorf("Result %d does not match. got %v; want %v", idx, got, want)
				}
			}
		})
	}
}

func TestBuild(t *testing.T) {
	testCases := []struct {
		name  string
		elems [][]byte
		want  []byte
	}{
		{
			"one element",
			[][]byte{AppendDoubleElement(nil, "pi", 3.14159)},
			[]byte{0x11, 0x00, 0x00, 0x00, 0x1, 0x70, 0x69, 0x00, 0x6e, 0x86, 0x1b, 0xf0, 0xf9, 0x21, 0x9, 0x40, 0x00},
		},
		{
			"two elements",
			[][]byte{AppendDoubleElement(nil, "pi", 3.14159), AppendStringElement(nil, "hello", "world!!")},
			[]byte{
				0x24, 0x00, 0x00, 0x00, 0x01, 0x70, 0x69, 0x00, 0x6e, 0x86, 0x1b, 0xf0,
				0xf9, 0x21, 0x09, 0x40, 0x02, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x00, 0x08,
				0x00, 0x00, 0x00, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0x21, 0x21, 0x00, 0x00,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("BuildDocument", func(t *testing.T) {
				elems := make([]byte, 0)
				for _, elem := range tc.elems {
					elems = append(elems, elem...)
				}
				got := BuildDocument(nil, elems)
				if !bytes.Equal(got, tc.want) {
					t.Errorf("Documents do not match. got %v; want %v", got, tc.want)
				}
			})
			t.Run("BuildDocumentFromElements", func(t *testing.T) {
				got := BuildDocumentFromElements(nil, tc.elems...)
				if !bytes.Equal(got, tc.want) {
					t.Errorf("Documents do not match. got %v; want %v", got, tc.want)
				}
			})
		})
	}
}

func TestNullBytes(t *testing.T) {
	// Helper function to execute the provided callback and assert that it panics with the expected message. The
	// createBSONFn callback should create a BSON document/array/value and return the stringified version.
	assertBSONCreationPanics := func(t *testing.T, createBSONFn func(), expected string) {
		t.Helper()

		defer func() {
			got := recover()
			assert.Equal(t, expected, got, "expected panic with error %v, got error %v", expected, got)
		}()
		createBSONFn()
	}

	t.Run("element keys", func(t *testing.T) {
		createDocFn := func() {
			NewDocumentBuilder().AppendString("a\x00", "foo")
		}
		assertBSONCreationPanics(t, createDocFn, invalidKeyPanicMsg)
	})
	t.Run("regex values", func(t *testing.T) {
		testCases := []struct {
			name    string
			pattern string
			options string
		}{
			{"null bytes in pattern", "a\x00", "i"},
			{"null bytes in options", "pattern", "i\x00"},
		}
		for _, tc := range testCases {
			t.Run(tc.name+"-AppendRegexElement", func(t *testing.T) {
				createDocFn := func() {
					AppendRegexElement(nil, "foo", tc.pattern, tc.options)
				}
				assertBSONCreationPanics(t, createDocFn, invalidRegexPanicMsg)
			})
			t.Run(tc.name+"-AppendRegex", func(t *testing.T) {
				createValFn := func() {
					AppendRegex(nil, tc.pattern, tc.options)
				}
				assertBSONCreationPanics(t, createValFn, invalidRegexPanicMsg)
			})
		}
	})
	t.Run("sub document field name", func(t *testing.T) {
		createDocFn := func() {
			NewDocumentBuilder().StartDocument("foobar").AppendDocument("a\x00", []byte("foo")).FinishDocument()
		}
		assertBSONCreationPanics(t, createDocFn, invalidKeyPanicMsg)
	})
}

func TestInvalidBytes(t *testing.T) {
	t.Parallel()

	t.Run("read length less than 4 int bytes", func(t *testing.T) {
		t.Parallel()

		_, src, ok := readLengthBytes([]byte{0x01, 0x00, 0x00, 0x00})
		assert.False(t, ok, "expected not ok response for invalid length read")
		assert.Equal(t, 4, len(src), "expected src to contain the size parameter still")
	})
}
