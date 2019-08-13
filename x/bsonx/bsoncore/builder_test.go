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

func TestDocumentBuilderAppend(t *testing.T) {
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
			"AppendValueElement",
			"AppendValueElement",
			[]interface{}{"testing", Value{Type: bsontype.Boolean, Data: []byte{0x01}}},
			[]byte{byte(bsontype.Boolean), 't', 'e', 's', 't', 'i', 'n', 'g', 0x00, 0x01},
		},
		{
			"AppendDoubleElement",
			"AppendDoubleElement",
			[]interface{}{"foobar", float64(3.14159)},
			append([]byte{byte(bsontype.Double), 'f', 'o', 'o', 'b', 'a', 'r', 0x00}, pi...),
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
			"AppendDocumentElement",
			"AppendDocumentElement",
			[]interface{}{"foobar", []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			[]byte{byte(bsontype.EmbeddedDocument),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x05, 0x00, 0x00, 0x00, 0x00,
			},
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
			"AppendBooleanElement",
			"AppendBooleanElement",
			[]interface{}{"foobar", true},
			[]byte{byte(bsontype.Boolean), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x01},
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
			"AppendRegexElement",
			"AppendRegexElement",
			[]interface{}{"foobar", "bar", "baz"},
			[]byte{byte(bsontype.Regex),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				'b', 'a', 'r', 0x00, 'b', 'a', 'z', 0x00,
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
			"AppendJavaScriptElement",
			"AppendJavaScriptElement",
			[]interface{}{"foobar", "barbaz"},
			[]byte{byte(bsontype.JavaScript),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00,
			},
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
			"AppendInt32Element",
			"AppendInt32Element",
			[]interface{}{"foobar", int32(256)},
			[]byte{byte(bsontype.Int32), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x00, 0x01, 0x00, 0x00},
		},
		{
			"AppendTimestampElement",
			"AppendTimestampElement",
			[]interface{}{"foobar", uint32(65536), uint32(256)},
			[]byte{byte(bsontype.Timestamp), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00},
		},
		{
			"AppendInt64Element",
			"AppendInt64Element",
			[]interface{}{"foobar", int64(4294967296)},
			[]byte{byte(bsontype.Int64), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00},
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
			builder := &DocumentBuilder{}
			fn := reflect.ValueOf(builder).MethodByName(tc.fn)
			if fn.Kind() != reflect.Func {
				t.Fatalf("fn must be of kind Func but is a %v", fn.Kind())
			}
			if fn.Type().NumIn() != len(tc.params) {
				t.Fatalf("tc.params must match the number of params in tc.fn. params %d; fn %d", fn.Type().NumIn(), len(tc.params))
			}
			if fn.Type().NumOut() != 1 || fn.Type().Out(0) != reflect.TypeOf(&DocumentBuilder{}) {
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

func TestDocumentBuilderBuild(t *testing.T) {
	t.Run("TestBuildOneElement", func(t *testing.T) {
		expected := []byte{0x11, 0x00, 0x00, 0x00, 0x1, 0x70, 0x69, 0x00, 0x6e, 0x86, 0x1b, 0xf0, 0xf9, 0x21, 0x9, 0x40, 0x00}
		result := NewDocumentBuilder().AppendDoubleElement("pi", 3.14159).Build()
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
		result := NewDocumentBuilder().AppendDoubleElement("pi", 3.14159).AppendStringElement("hello", "world!!").Build()
		if !bytes.Equal(result, expected) {
			t.Errorf("Documents do not match. got %v; want %v", result, expected)
		}
	})
}
