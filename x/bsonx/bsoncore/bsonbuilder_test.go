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

func emptyDocumentBuilder() *DocumentBuilder {
	return &DocumentBuilder{}
}

func emptyArrayBuilder() *ArrayBuilder {
	a := ArrayBuilder{}
	a.indexes = append(a.indexes, int32(0))
	a.key = append(a.key, 0)
	return &a
}

func TestDocumentBuilder(t *testing.T) {
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
			"AppendInt32",
			emptyDocumentBuilder().AppendInt32,
			[]interface{}{"foobar", int32(256)},
			[]byte{byte(bsontype.Int32), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x00, 0x01, 0x00, 0x00},
		},
		{
			"AppendDouble",
			emptyDocumentBuilder().AppendDouble,
			[]interface{}{"foobar", float64(3.14159)},
			append([]byte{byte(bsontype.Double), 'f', 'o', 'o', 'b', 'a', 'r', 0x00}, pi...),
		},
		{
			"AppendString",
			emptyDocumentBuilder().AppendString,
			[]interface{}{"foobar", "x"},
			[]byte{byte(bsontype.String), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x02, 0x00, 0x00, 0x00, 'x', 0x00},
		},
		{
			"AppendDocument",
			emptyDocumentBuilder().AppendDocument,
			[]interface{}{"foobar", []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			[]byte{byte(bsontype.EmbeddedDocument),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x05, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			"AppendArray",
			emptyDocumentBuilder().AppendArray,
			[]interface{}{"foobar", []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			[]byte{byte(bsontype.Array),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x05, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			"AppendBinary",
			emptyDocumentBuilder().AppendBinary,
			[]interface{}{"foobar", byte(0x02), []byte{0x01, 0x02, 0x03}},
			[]byte{byte(bsontype.Binary),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x07, 0x00, 0x00, 0x00,
				0x02,
				0x03, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03,
			},
		},
		{
			"AppendObjectID",
			emptyDocumentBuilder().AppendObjectID,
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
			"AppendBoolean",
			emptyDocumentBuilder().AppendBoolean,
			[]interface{}{"foobar", true},
			[]byte{byte(bsontype.Boolean), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x01},
		},
		{
			"AppendDateTime",
			emptyDocumentBuilder().AppendDateTime,
			[]interface{}{"foobar", int64(256)},
			[]byte{byte(bsontype.DateTime), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			"AppendNull",
			emptyDocumentBuilder().AppendNull,
			[]interface{}{"foobar"},
			[]byte{byte(bsontype.Null), 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
		},
		{
			"AppendRegex",
			emptyDocumentBuilder().AppendRegex,
			[]interface{}{"foobar", "bar", "baz"},
			[]byte{byte(bsontype.Regex),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				'b', 'a', 'r', 0x00, 'b', 'a', 'z', 0x00,
			},
		},
		{
			"AppendJavaScript",
			emptyDocumentBuilder().AppendJavaScript,
			[]interface{}{"foobar", "barbaz"},
			[]byte{byte(bsontype.JavaScript),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00,
			},
		},
		{
			"AppendCodeWithScope",
			emptyDocumentBuilder().AppendCodeWithScope,
			[]interface{}{"foobar", "barbaz", []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			[]byte{byte(bsontype.CodeWithScope),
				'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x14, 0x00, 0x00, 0x00,
				0x07, 0x00, 0x00, 0x00, 'b', 'a', 'r', 'b', 'a', 'z', 0x00,
				0x05, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			"AppendTimestamp",
			emptyDocumentBuilder().AppendTimestamp,
			[]interface{}{"foobar", uint32(65536), uint32(256)},
			[]byte{byte(bsontype.Timestamp), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00},
		},
		{
			"AppendInt64",
			emptyDocumentBuilder().AppendInt64,
			[]interface{}{"foobar", int64(4294967296)},
			[]byte{byte(bsontype.Int64), 'f', 'o', 'o', 'b', 'a', 'r', 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00},
		},
		{
			"AppendDecimal128",
			emptyDocumentBuilder().AppendDecimal128,
			[]interface{}{"foobar", primitive.NewDecimal128(4294967296, 65536)},
			[]byte{
				byte(bsontype.Decimal128), 'f', 'o', 'o', 'b', 'a', 'r', 0x00,
				0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
			},
		},
		{
			"AppendMaxKey",
			emptyDocumentBuilder().AppendMaxKey,
			[]interface{}{"foobar"},
			[]byte{byte(bsontype.MaxKey), 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
		},
		{
			"AppendMinKey",
			emptyDocumentBuilder().AppendMinKey,
			[]interface{}{"foobar"},
			[]byte{byte(bsontype.MinKey), 'f', 'o', 'o', 'b', 'a', 'r', 0x00},
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
			if fn.Type().NumOut() != 1 || fn.Type().Out(0) != reflect.TypeOf(&DocumentBuilder{}) {
				t.Fatalf("fn must have one return parameter and it must be a DocumentBuilder.")
			}
			params := make([]reflect.Value, 0, len(tc.params))
			for _, param := range tc.params {
				params = append(params, reflect.ValueOf(param))
			}
			results := fn.Call(params)
			got := results[0].Interface().(*DocumentBuilder).doc
			want := tc.expected
			if !bytes.Equal(got, want) {
				t.Errorf("Did not receive expected bytes. got %v; want %v", got, want)
			}
		})
	}
	t.Run("TestBuildOneElement", func(t *testing.T) {
		expected := []byte{0x0c, 0x00, 0x00, 0x00, 0x10, 'x', 0x00, 0x00, 0x01, 0x00, 0x00, 0x00}
		result := NewDocumentBuilder().AppendInt32("x", int32(256)).Build()
		if !bytes.Equal(result, expected) {
			t.Errorf("Documents do not match. got %v; want %v", result, expected)
		}
	})
	t.Run("TestBuildTwoElements", func(t *testing.T) {
		expected := []byte{
			0x1b, 0x00, 0x00, 0x00, 0x10, 'x', 0x00, 0x03, 0x00, 0x00, 0x00, 0x04,
			'y', 0x00, 0x0c, 0x00, 0x00, 0x00, 0x10, '0', 0x00, 0x01, 0x00, 0x00,
			0x00, 0x00, 0x00,
		}
		elem := NewArrayBuilder().AppendInt32(int32(1)).Build()
		result := NewDocumentBuilder().AppendInt32("x", int32(3)).AppendArray("y", elem).Build()
		if !bytes.Equal(result, expected) {
			t.Errorf("Documents do not match. got %v; want %v", result, expected)
		}
	})
}

func TestArrayBuilder(t *testing.T) {
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
			"AppendInt32",
			emptyArrayBuilder().AppendInt32,
			[]interface{}{int32(256)},
			[]byte{byte(bsontype.Int32), '0', 0x00, 0x00, 0x01, 0x00, 0x00},
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
			if fn.Type().NumOut() != 1 || fn.Type().Out(0) != reflect.TypeOf(&ArrayBuilder{}) {
				t.Fatalf("fn must have one return parameter and it must be an ArrayBuilder.")
			}
			params := make([]reflect.Value, 0, len(tc.params))
			for _, param := range tc.params {
				params = append(params, reflect.ValueOf(param))
			}
			results := fn.Call(params)
			got := results[0].Interface().(*ArrayBuilder).arr
			want := tc.expected
			if !bytes.Equal(got, want) {
				t.Errorf("Did not receive expected bytes. got %v; want %v", got, want)
			}
		})
	}
	t.Run("TestBuildOneElementArray", func(t *testing.T) {
		expected := []byte{0x0c, 0x00, 0x00, 0x00, 0x10, '0', 0x00, 0x00, 0x01, 0x00, 0x00, 0x00}
		result := NewArrayBuilder().AppendInt32(int32(256)).Build()
		if !bytes.Equal(result, expected) {
			t.Errorf("Arrays do not match. got %v; want %v", result, expected)
		}
	})
	t.Run("TestBuildTwoElementsArray", func(t *testing.T) {
		expected := []byte{
			0x1b, 0x00, 0x00, 0x00, 0x10, '0', 0x00, 0x03, 0x00, 0x00, 0x00, 0x04,
			'1', 0x00, 0x0c, 0x00, 0x00, 0x00, 0x10, '0', 0x00, 0x01, 0x00, 0x00,
			0x00, 0x00, 0x00,
		}
		elem := NewArrayBuilder().AppendInt32(int32(1)).Build()
		result := NewArrayBuilder().AppendInt32(int32(3)).AppendArray(elem).Build()
		if !bytes.Equal(result, expected) {
			t.Errorf("Arrays do not match. got %v; want %v", result, expected)
		}
	})
}
