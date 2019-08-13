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

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestBuilderAppend(t *testing.T) {
	bits := math.Float64bits(3.14159)
	pi := make([]byte, 8)
	binary.LittleEndian.PutUint64(pi, bits)

	testCases := []struct {
		name     string
		fn       string
		params   []interface{}
		expected []byte
	}{
		{
			"AppendType",
			"AppendType",
			[]interface{}{bsontype.Null},
			[]byte{byte(bsontype.Null)},
		},
		{
			"AppendKey",
			"AppendKey",
			[]interface{}{"foobar"},
			[]byte{'f', 'o', 'o', 'b', 'a', 'r', 0x00},
		},
		{
			"AppendHeader",
			"AppendHeader",
			[]interface{}{bsontype.Null, "foobar"},
			[]byte{byte(bsontype.Null), 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
		},
		{
			"AppendValueElement",
			"AppendValueElement",
			[]interface{}{"testing", Value{Type: bsontype.Boolean, Data: []byte{0x01}}},
			[]byte{byte(bsontype.Boolean), 't', 'e', 's', 't', 'i', 'n', 'g', 0x00, 0x01},
		},
		{
			"AppendDouble",
			"AppendDouble",
			[]interface{}{float64(3.14159)},
			pi,
		},
		{
			"AppendDoubleElement",
			"AppendDoubleElement",
			[]interface{}{"foobar", float64(3.14159)},
			append([]byte{byte(bsontype.Double), 'f', 'o', 'o', 'b', 'a', 'r', 0x00}, pi...),
		},
		{
			"AppendString",
			"AppendString",
			[]interface{}{"barbaz"},
			[]byte{0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00},
		},
		{
			"AppendStringElement",
			"AppendStringElement",
			[]interface{}{"foobar", "barbaz"},
			[]byte{byte(bsontype.String),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00,
			},
		},
		{
			"AppendDocument",
			"AppendDocument",
			[]interface{}{[]byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			[]byte{0x05, 0x00, 0x00, 0x00, 0x00},
		},
		{
			"AppendDocumentElement",
			"AppendDocumentElement",
			[]interface{}{"foobar", []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			[]byte{byte(bsontype.EmbeddedDocument),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x05, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			"AppendArray",
			"AppendArray",
			[]interface{}{[]byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			[]byte{0x05, 0x00, 0x00, 0x00, 0x00},
		},
		{
			"AppendArrayElement",
			"AppendArrayElement",
			[]interface{}{"foobar", []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			[]byte{byte(bsontype.Array),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x05, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			"AppendBinary Subtype2",
			"AppendBinary",
			[]interface{}{byte(0x02), []byte{0x01, 0x02, 0x03}},
			[]byte{0x07, 0x00, 0x00, 0x00, 0x02, 0x03, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
		},
		{
			"AppendBinaryElement Subtype 2",
			"AppendBinaryElement",
			[]interface{}{"foobar", byte(0x02), []byte{0x01, 0x02, 0x03}},
			[]byte{byte(bsontype.Binary),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x07, 0x00, 0x00, 0x00,
				0x02,
				0x03, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03,
			},
		},
		{
			"AppendBinary",
			"AppendBinary",
			[]interface{}{byte(0xFF), []byte{0x01, 0x02, 0x03}},
			[]byte{0x03, 0x00, 0x00, 0x00, 0xFF, 0x01, 0x02, 0x03},
		},
		{
			"AppendBinaryElement",
			"AppendBinaryElement",
			[]interface{}{"foobar", byte(0xFF), []byte{0x01, 0x02, 0x03}},
			[]byte{byte(bsontype.Binary),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x03, 0x00, 0x00, 0x00,
				0xFF,
				0x01, 0x02, 0x03,
			},
		},
		{
			"AppendUndefinedElement",
			"AppendUndefinedElement",
			[]interface{}{"foobar"},
			[]byte{byte(bsontype.Undefined), 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
		},
		{
			"AppendObjectID",
			"AppendObjectID",
			[]interface{}{
				primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			},
			[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
		},
		{
			"AppendObjectIDElement",
			"AppendObjectIDElement",
			[]interface{}{
				"foobar",
				primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			},
			[]byte{byte(bsontype.ObjectID),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C,
			},
		},
		{
			"AppendBoolean (true)",
			"AppendBoolean",
			[]interface{}{true},
			[]byte{0x01},
		},
		{
			"AppendBoolean (false)",
			"AppendBoolean",
			[]interface{}{false},
			[]byte{0x00},
		},
		{
			"AppendBooleanElement",
			"AppendBooleanElement",
			[]interface{}{"foobar", true},
			[]byte{byte(bsontype.Boolean), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x01},
		},
		{
			"AppendDateTime",
			"AppendDateTime",
			[]interface{}{int64(256)},
			[]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			"AppendDateTimeElement",
			"AppendDateTimeElement",
			[]interface{}{"foobar", int64(256)},
			[]byte{byte(bsontype.DateTime), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			"AppendNullElement",
			"AppendNullElement",
			[]interface{}{"foobar"},
			[]byte{byte(bsontype.Null), 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
		},
		{
			"AppendRegex",
			"AppendRegex",
			[]interface{}{"bar", "baz"},
			[]byte{'b', 'a', 'r', 0x00, 'b', 'a', 'z', 0x00},
		},
		{
			"AppendRegexElement",
			"AppendRegexElement",
			[]interface{}{"foobar", "bar", "baz"},
			[]byte{byte(bsontype.Regex),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				'b', 'a', 'r', 0x00, 'b', 'a', 'z', 0x00,
			},
		},
		{
			"AppendDBPointer",
			"AppendDBPointer",
			[]interface{}{
				"foobar",
				primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			},
			[]byte{
				0x07, 0x00, 0x00, 0x00, 'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C,
			},
		},
		{
			"AppendDBPointerElement",
			"AppendDBPointerElement",
			[]interface{}{
				"foobar",
				"barbaz",
				primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			},
			[]byte{byte(bsontype.DBPointer),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00,
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C,
			},
		},
		{
			"AppendJavaScript",
			"AppendJavaScript",
			[]interface{}{"barbaz"},
			[]byte{0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00},
		},
		{
			"AppendJavaScriptElement",
			"AppendJavaScriptElement",
			[]interface{}{"foobar", "barbaz"},
			[]byte{byte(bsontype.JavaScript),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00,
			},
		},
		{
			"AppendSymbol",
			"AppendSymbol",
			[]interface{}{"barbaz"},
			[]byte{0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00},
		},
		{
			"AppendSymbolElement",
			"AppendSymbolElement",
			[]interface{}{"foobar", "barbaz"},
			[]byte{byte(bsontype.Symbol),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00,
			},
		},
		{
			"AppendCodeWithScope",
			"AppendCodeWithScope",
			[]interface{}{"foobar", []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			[]byte{
				0x14, 0x00, 0x00, 0x00,
				0x07, 0x00, 0x00, 0x00, 'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x05, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			"AppendCodeWithScopeElement",
			"AppendCodeWithScopeElement",
			[]interface{}{"foobar", "barbaz", []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			[]byte{byte(bsontype.CodeWithScope),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x14, 0x00, 0x00, 0x00,
				0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00,
				0x05, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			"AppendInt32",
			"AppendInt32",
			[]interface{}{int32(256)},
			[]byte{0x00, 0x01, 0x00, 0x00},
		},
		{
			"AppendInt32Element",
			"AppendInt32Element",
			[]interface{}{"foobar", int32(256)},
			[]byte{byte(bsontype.Int32), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x00, 0x01, 0x00, 0x00},
		},
		{
			"AppendTimestamp",
			"AppendTimestamp",
			[]interface{}{uint32(65536), uint32(256)},
			[]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00},
		},
		{
			"AppendTimestampElement",
			"AppendTimestampElement",
			[]interface{}{"foobar", uint32(65536), uint32(256)},
			[]byte{byte(bsontype.Timestamp), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00},
		},
		{
			"AppendInt64",
			"AppendInt64",
			[]interface{}{int64(4294967296)},
			[]byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00},
		},
		{
			"AppendInt64Element",
			"AppendInt64Element",
			[]interface{}{"foobar", int64(4294967296)},
			[]byte{byte(bsontype.Int64), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00},
		},
		{
			"AppendDecimal128",
			"AppendDecimal128",
			[]interface{}{primitive.NewDecimal128(4294967296, 65536)},
			[]byte{
				0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
			},
		},
		{
			"AppendDecimal128Element",
			"AppendDecimal128Element",
			[]interface{}{"foobar", primitive.NewDecimal128(4294967296, 65536)},
			[]byte{
				byte(bsontype.Decimal128), 'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
			},
		},
		{
			"AppendMaxKeyElement",
			"AppendMaxKeyElement",
			[]interface{}{"foobar"},
			[]byte{byte(bsontype.MaxKey), 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
		},
		{
			"AppendMinKeyElement",
			"AppendMinKeyElement",
			[]interface{}{"foobar"},
			[]byte{byte(bsontype.MinKey), 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := NewBuilder()
			fn := reflect.ValueOf(builder).MethodByName(tc.fn)
			if fn.Kind() != reflect.Func {
				t.Fatalf("fn must be of kind Func but is a %v", fn.Kind())
			}
			if fn.Type().NumIn() != len(tc.params) {
				t.Fatalf("tc.params must match the number of params in tc.fn. params %d; fn %d", fn.Type().NumIn(), len(tc.params))
			}
			if fn.Type().NumOut() != 1 || fn.Type().Out(0) != reflect.TypeOf(&Builder{}) {
				t.Fatalf("fn must have one return parameter and it must be a *Builder.")
			}
			params := make([]reflect.Value, 0, len(tc.params))
			for _, param := range tc.params {
				params = append(params, reflect.ValueOf(param))
			}
			_ = reflect.ValueOf(builder).MethodByName(tc.fn).Call(params)
			got := builder.bson
			want := tc.expected
			if !bytes.Equal(got, want) {
				t.Errorf("Did not receive expected bytes. got %v; want %v", got, want)
			}
		})
	}
}

func TestBuilderBuild(t *testing.T) {
	t.Run("TestBuildOneElement", func(t *testing.T) {
		expected := []byte{0x11, 0x00, 0x00, 0x00, 0x1, 0x70, 0x69, 0x00, 0x6e, 0x86, 0x1b, 0xf0, 0xf9, 0x21, 0x9, 0x40, 0x00}
		builder := NewBuilder()
		builder = builder.AppendDocumentStart().AppendDoubleElement("pi", 3.14159).AppendDocumentEnd(builder.index)
		result := builder.bson
		if !bytes.Equal(result, expected) {
			t.Errorf("Documents do not match. got %v; want %v", result, expected)
		}
	})
	t.Run("TestBuildTwoElements", func(t *testing.T) {
		expected := []byte{
			0x24, 0x00, 0x00, 0x00, 0x01, 0x70, 0x69, 0x00, 0x6e, 0x86, 0x1b, 0xf0,
			0xf9, 0x21, 0x09, 0x40, 0x02, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x00, 0x08,
			0x00, 0x00, 0x00, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0x21, 0x21, 0x00, 0x00,
		}
		builder := NewBuilder()
		builder = builder.AppendDocumentStart().AppendDoubleElement("pi", 3.14159).AppendStringElement("hello", "world!!").AppendDocumentEnd(builder.index)
		result := builder.bson
		if !bytes.Equal(result, expected) {
			t.Errorf("Documents do not match. got %v; want %v", result, expected)
		}
	})
}
