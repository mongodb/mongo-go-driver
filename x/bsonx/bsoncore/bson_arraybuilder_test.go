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

	"go.mongodb.org/mongo-driver/bson/primitive"
)

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
			NewArrayBuilder().AppendInt32,
			[]interface{}{int32(256)},
			BuildDocumentFromElements(nil, AppendInt32Element(nil, "0", int32(256))),
		},
		{
			"AppendDouble",
			NewArrayBuilder().AppendDouble,
			[]interface{}{float64(3.14159)},
			BuildDocumentFromElements(nil, AppendDoubleElement(nil, "0", float64(3.14159))),
		},
		{
			"AppendString",
			NewArrayBuilder().AppendString,
			[]interface{}{"x"},
			BuildDocumentFromElements(nil, AppendStringElement(nil, "0", "x")),
		},
		{
			"AppendDocument",
			NewArrayBuilder().AppendDocument,
			[]interface{}{[]byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			BuildDocumentFromElements(nil, AppendDocumentElement(nil, "0", []byte{0x05, 0x00, 0x00, 0x00, 0x00})),
		},
		{
			"AppendArray",
			NewArrayBuilder().AppendArray,
			[]interface{}{[]byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			BuildDocumentFromElements(nil, AppendArrayElement(nil, "0", []byte{0x05, 0x00, 0x00, 0x00, 0x00})),
		},
		{
			"AppendBinary",
			NewArrayBuilder().AppendBinary,
			[]interface{}{byte(0x02), []byte{0x01, 0x02, 0x03}},
			BuildDocumentFromElements(nil, AppendBinaryElement(nil, "0", byte(0x02), []byte{0x01, 0x02, 0x03})),
		},
		{
			"AppendObjectID",
			NewArrayBuilder().AppendObjectID,
			[]interface{}{
				primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			},
			BuildDocumentFromElements(nil, AppendObjectIDElement(nil, "0",
				primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C})),
		},
		{
			"AppendBoolean",
			NewArrayBuilder().AppendBoolean,
			[]interface{}{true},
			BuildDocumentFromElements(nil, AppendBooleanElement(nil, "0", true)),
		},
		{
			"AppendDateTime",
			NewArrayBuilder().AppendDateTime,
			[]interface{}{int64(256)},
			BuildDocumentFromElements(nil, AppendDateTimeElement(nil, "0", int64(256))),
		},
		{
			"AppendNull",
			NewArrayBuilder().AppendNull,
			[]interface{}{},
			BuildDocumentFromElements(nil, AppendNullElement(nil, "0")),
		},
		{
			"AppendRegex",
			NewArrayBuilder().AppendRegex,
			[]interface{}{"bar", "baz"},
			BuildDocumentFromElements(nil, AppendRegexElement(nil, "0", "bar", "baz")),
		},
		{
			"AppendJavaScript",
			NewArrayBuilder().AppendJavaScript,
			[]interface{}{"barbaz"},
			BuildDocumentFromElements(nil, AppendJavaScriptElement(nil, "0", "barbaz")),
		},
		{
			"AppendCodeWithScope",
			NewArrayBuilder().AppendCodeWithScope,
			[]interface{}{"barbaz", Document([]byte{0x05, 0x00, 0x00, 0x00, 0x00})},
			BuildDocumentFromElements(nil, AppendCodeWithScopeElement(nil, "0", "barbaz", Document([]byte{0x05, 0x00, 0x00, 0x00, 0x00}))),
		},
		{
			"AppendTimestamp",
			NewArrayBuilder().AppendTimestamp,
			[]interface{}{uint32(65536), uint32(256)},
			BuildDocumentFromElements(nil, AppendTimestampElement(nil, "0", uint32(65536), uint32(256))),
		},
		{
			"AppendInt64",
			NewArrayBuilder().AppendInt64,
			[]interface{}{int64(4294967296)},
			BuildDocumentFromElements(nil, AppendInt64Element(nil, "0", int64(4294967296))),
		},
		{
			"AppendDecimal128",
			NewArrayBuilder().AppendDecimal128,
			[]interface{}{primitive.NewDecimal128(4294967296, 65536)},
			BuildDocumentFromElements(nil, AppendDecimal128Element(nil, "0", primitive.NewDecimal128(4294967296, 65536))),
		},
		{
			"AppendMaxKey",
			NewArrayBuilder().AppendMaxKey,
			[]interface{}{},
			BuildDocumentFromElements(nil, AppendMaxKeyElement(nil, "0")),
		},
		{
			"AppendMinKey",
			NewArrayBuilder().AppendMinKey,
			[]interface{}{},
			BuildDocumentFromElements(nil, AppendMinKeyElement(nil, "0")),
		},
		{
			"AppendSymbol",
			NewArrayBuilder().AppendSymbol,
			[]interface{}{"barbaz"},
			BuildDocumentFromElements(nil, AppendSymbolElement(nil, "0", "barbaz")),
		},
		{
			"AppendDBPointer",
			NewArrayBuilder().AppendDBPointer,
			[]interface{}{"barbaz",
				primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}},
			BuildDocumentFromElements(nil, AppendDBPointerElement(nil, "0", "barbaz",
				primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C})),
		},
		{
			"AppendUndefined",
			NewArrayBuilder().AppendUndefined,
			[]interface{}{},
			BuildDocumentFromElements(nil, AppendUndefinedElement(nil, "0")),
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
			got := results[0].Interface().(*ArrayBuilder).Build()
			want := tc.expected
			if !bytes.Equal(got, want) {
				t.Errorf("Did not receive expected bytes. got %v; want %v", got, want)
			}
		})
	}
	t.Run("TestBuildTwoElementsArray", func(t *testing.T) {
		intArr := BuildDocumentFromElements(nil, AppendInt32Element(nil, "0", int32(1)))
		expected := BuildDocumentFromElements(nil, AppendArrayElement(AppendInt32Element(nil, "0", int32(3)), "1", intArr))
		elem := NewArrayBuilder().AppendInt32(int32(1)).Build()
		result := NewArrayBuilder().AppendInt32(int32(3)).AppendArray(elem).Build()
		if !bytes.Equal(result, expected) {
			t.Errorf("Arrays do not match. got %v; want %v", result, expected)
		}
	})
	t.Run("TestBuildInlineArray", func(t *testing.T) {
		docElement := BuildDocumentFromElements(nil, AppendInt32Element(nil, "0", int32(256)))
		var expected Document
		expected = BuildDocumentFromElements(nil, AppendArrayElement(nil, "0", docElement))
		result := NewArrayBuilder().StartArray().AppendInt32(int32(256)).FinishArray().Build()
		if !bytes.Equal(result, expected) {
			t.Errorf("Documents do not match. got %v; want %v", result, expected)
		}
	})
	t.Run("TestBuildNestedInlineArray", func(t *testing.T) {
		docElement := BuildDocumentFromElements(nil, AppendDoubleElement(nil, "0", 3.14))
		docInline := BuildDocumentFromElements(nil, AppendArrayElement(nil, "0", docElement))
		var expected Document
		expected = BuildDocumentFromElements(nil, AppendArrayElement(nil, "0", docInline))
		result := NewArrayBuilder().StartArray().StartArray().AppendDouble(3.14).FinishArray().FinishArray().Build()
		if !bytes.Equal(result, expected) {
			t.Errorf("Documents do not match. got %v; want %v", result, expected)
		}
	})
}
