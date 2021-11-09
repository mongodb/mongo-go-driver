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
			NewDocumentBuilder().AppendInt32,
			[]interface{}{"foobar", int32(256)},
			BuildDocumentFromElements(nil, AppendInt32Element(nil, "foobar", 256)),
		},
		{
			"AppendDouble",
			NewDocumentBuilder().AppendDouble,
			[]interface{}{"foobar", float64(3.14159)},
			BuildDocumentFromElements(nil, AppendDoubleElement(nil, "foobar", float64(3.14159))),
		},
		{
			"AppendString",
			NewDocumentBuilder().AppendString,
			[]interface{}{"foobar", "x"},
			BuildDocumentFromElements(nil, AppendStringElement(nil, "foobar", "x")),
		},
		{
			"AppendDocument",
			NewDocumentBuilder().AppendDocument,
			[]interface{}{"foobar", []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			BuildDocumentFromElements(nil, AppendDocumentElement(nil, "foobar", []byte{0x05, 0x00, 0x00, 0x00, 0x00})),
		},
		{
			"AppendArray",
			NewDocumentBuilder().AppendArray,
			[]interface{}{"foobar", []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			BuildDocumentFromElements(nil, AppendArrayElement(nil, "foobar", []byte{0x05, 0x00, 0x00, 0x00, 0x00})),
		},
		{
			"AppendBinary",
			NewDocumentBuilder().AppendBinary,
			[]interface{}{"foobar", byte(0x02), []byte{0x01, 0x02, 0x03}},
			BuildDocumentFromElements(nil, AppendBinaryElement(nil, "foobar", byte(0x02), []byte{0x01, 0x02, 0x03})),
		},
		{
			"AppendObjectID",
			NewDocumentBuilder().AppendObjectID,
			[]interface{}{
				"foobar",
				primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			},
			BuildDocumentFromElements(nil, AppendObjectIDElement(nil, "foobar",
				primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C})),
		},
		{
			"AppendBoolean",
			NewDocumentBuilder().AppendBoolean,
			[]interface{}{"foobar", true},
			BuildDocumentFromElements(nil, AppendBooleanElement(nil, "foobar", true)),
		},
		{
			"AppendDateTime",
			NewDocumentBuilder().AppendDateTime,
			[]interface{}{"foobar", int64(256)},
			BuildDocumentFromElements(nil, AppendDateTimeElement(nil, "foobar", int64(256))),
		},
		{
			"AppendNull",
			NewDocumentBuilder().AppendNull,
			[]interface{}{"foobar"},
			BuildDocumentFromElements(nil, AppendNullElement(nil, "foobar")),
		},
		{
			"AppendRegex",
			NewDocumentBuilder().AppendRegex,
			[]interface{}{"foobar", "bar", "baz"},
			BuildDocumentFromElements(nil, AppendRegexElement(nil, "foobar", "bar", "baz")),
		},
		{
			"AppendJavaScript",
			NewDocumentBuilder().AppendJavaScript,
			[]interface{}{"foobar", "barbaz"},
			BuildDocumentFromElements(nil, AppendJavaScriptElement(nil, "foobar", "barbaz")),
		},
		{
			"AppendCodeWithScope",
			NewDocumentBuilder().AppendCodeWithScope,
			[]interface{}{"foobar", "barbaz", Document([]byte{0x05, 0x00, 0x00, 0x00, 0x00})},
			BuildDocumentFromElements(nil, AppendCodeWithScopeElement(nil, "foobar", "barbaz", Document([]byte{0x05, 0x00, 0x00, 0x00, 0x00}))),
		},
		{
			"AppendTimestamp",
			NewDocumentBuilder().AppendTimestamp,
			[]interface{}{"foobar", uint32(65536), uint32(256)},
			BuildDocumentFromElements(nil, AppendTimestampElement(nil, "foobar", uint32(65536), uint32(256))),
		},
		{
			"AppendInt64",
			NewDocumentBuilder().AppendInt64,
			[]interface{}{"foobar", int64(4294967296)},
			BuildDocumentFromElements(nil, AppendInt64Element(nil, "foobar", int64(4294967296))),
		},
		{
			"AppendDecimal128",
			NewDocumentBuilder().AppendDecimal128,
			[]interface{}{"foobar", primitive.NewDecimal128(4294967296, 65536)},
			BuildDocumentFromElements(nil, AppendDecimal128Element(nil, "foobar", primitive.NewDecimal128(4294967296, 65536))),
		},
		{
			"AppendMaxKey",
			NewDocumentBuilder().AppendMaxKey,
			[]interface{}{"foobar"},
			BuildDocumentFromElements(nil, AppendMaxKeyElement(nil, "foobar")),
		},
		{
			"AppendMinKey",
			NewDocumentBuilder().AppendMinKey,
			[]interface{}{"foobar"},
			BuildDocumentFromElements(nil, AppendMinKeyElement(nil, "foobar")),
		},
		{
			"AppendSymbol",
			NewDocumentBuilder().AppendSymbol,
			[]interface{}{"foobar", "barbaz"},
			BuildDocumentFromElements(nil, AppendSymbolElement(nil, "foobar", "barbaz")),
		},
		{
			"AppendDBPointer",
			NewDocumentBuilder().AppendDBPointer,
			[]interface{}{"foobar", "barbaz",
				primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}},
			BuildDocumentFromElements(nil, AppendDBPointerElement(nil, "foobar", "barbaz",
				primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C})),
		},
		{
			"AppendUndefined",
			NewDocumentBuilder().AppendUndefined,
			[]interface{}{"foobar"},
			BuildDocumentFromElements(nil, AppendUndefinedElement(nil, "foobar")),
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
			got := results[0].Interface().(*DocumentBuilder)
			doc := got.Build()
			want := tc.expected
			if !bytes.Equal(doc, want) {
				t.Errorf("Did not receive expected bytes. got %v; want %v", got, want)
			}
		})
	}
	t.Run("TestBuildTwoElements", func(t *testing.T) {
		intArr := BuildDocumentFromElements(nil, AppendInt32Element(nil, "0", int32(1)))
		expected := BuildDocumentFromElements(nil, AppendArrayElement(AppendInt32Element(nil, "x", int32(3)), "y", intArr))
		elem := NewArrayBuilder().AppendInt32(int32(1)).Build()
		result := NewDocumentBuilder().AppendInt32("x", int32(3)).AppendArray("y", elem).Build()
		if !bytes.Equal(result, expected) {
			t.Errorf("Documents do not match. got %v; want %v", result, expected)
		}
	})
	t.Run("TestBuildInlineDocument", func(t *testing.T) {
		docElement := BuildDocumentFromElements(nil, AppendInt32Element(nil, "x", int32(256)))
		expected := Document(BuildDocumentFromElements(nil, AppendDocumentElement(nil, "y", docElement)))
		result := NewDocumentBuilder().StartDocument("y").AppendInt32("x", int32(256)).FinishDocument().Build()
		if !bytes.Equal(result, expected) {
			t.Errorf("Documents do not match. got %v; want %v", result, expected)
		}
	})
	t.Run("TestBuildNestedInlineDocument", func(t *testing.T) {
		docElement := BuildDocumentFromElements(nil, AppendDoubleElement(nil, "x", 3.14))
		docInline := BuildDocumentFromElements(nil, AppendDocumentElement(nil, "y", docElement))
		expected := Document(BuildDocumentFromElements(nil, AppendDocumentElement(nil, "z", docInline)))
		result := NewDocumentBuilder().StartDocument("z").StartDocument("y").AppendDouble("x", 3.14).FinishDocument().FinishDocument().Build()
		if !bytes.Equal(result, expected) {
			t.Errorf("Documents do not match. got %v; want %v", result, expected)
		}
	})
}
