// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
)

func TestValue(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("invalid", func(t *testing.T) {
			t.Parallel()

			v := Value{Type: TypeDouble, Data: []byte{0x01, 0x02, 0x03, 0x04}}
			want := NewInsufficientBytesError(v.Data, v.Data)
			got := v.Validate()
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("value", func(t *testing.T) {
			t.Parallel()

			v := Value{Type: TypeDouble, Data: AppendDouble(nil, 3.14159)}
			var want error
			got := v.Validate()
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
	})

	t.Run("IsNumber", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name  string
			val   Value
			isnum bool
		}{
			{"double", Value{Type: TypeDouble}, true},
			{"int32", Value{Type: TypeInt32}, true},
			{"int64", Value{Type: TypeInt64}, true},
			{"decimal128", Value{Type: TypeDecimal128}, true},
			{"string", Value{Type: TypeString}, false},
			{"regex", Value{Type: TypeRegex}, false},
		}

		for _, tc := range testCases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				isnum := tc.val.IsNumber()
				if isnum != tc.isnum {
					t.Errorf("IsNumber did not return the expected boolean. got %t; want %t", isnum, tc.isnum)
				}
			})
		}
	})

	now := time.Now().Truncate(time.Millisecond)
	var oid [12]byte

	testCases := []struct {
		name     string
		fn       interface{}
		val      Value
		panicErr error
		ret      []interface{}
	}{
		{
			"Double/Not Double", Value.Double, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Double", TypeString},
			nil,
		},
		{
			"Double/Insufficient Bytes", Value.Double, Value{Type: TypeDouble, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}),
			nil,
		},
		{
			"Double/Success", Value.Double, Value{Type: TypeDouble, Data: AppendDouble(nil, 3.14159)},
			nil,
			[]interface{}{float64(3.14159)},
		},
		{
			"DoubleOK/Not Double", Value.DoubleOK, Value{Type: TypeString},
			nil,
			[]interface{}{float64(0), false},
		},
		{
			"DoubleOK/Insufficient Bytes", Value.DoubleOK, Value{Type: TypeDouble, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			nil,
			[]interface{}{float64(0), false},
		},
		{
			"DoubleOK/Success", Value.DoubleOK, Value{Type: TypeDouble, Data: AppendDouble(nil, 3.14159)},
			nil,
			[]interface{}{float64(3.14159), true},
		},
		{
			"StringValue/Not String", Value.StringValue, Value{Type: TypeDouble},
			ElementTypeError{"bsoncore.Value.StringValue", TypeDouble},
			nil,
		},
		{
			"StringValue/Insufficient Bytes", Value.StringValue, Value{Type: TypeString, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}),
			nil,
		},
		{
			"StringValue/Zero Length", Value.StringValue, Value{Type: TypeString, Data: []byte{0x00, 0x00, 0x00, 0x00}},
			NewInsufficientBytesError([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0x00, 0x00, 0x00, 0x00}),
			nil,
		},
		{
			"StringValue/Success", Value.StringValue, Value{Type: TypeString, Data: AppendString(nil, "hello, world!")},
			nil,
			[]interface{}{"hello, world!"},
		},
		{
			"StringValueOK/Not String", Value.StringValueOK, Value{Type: TypeDouble},
			nil,
			[]interface{}{"", false},
		},
		{
			"StringValueOK/Insufficient Bytes", Value.StringValueOK, Value{Type: TypeString, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			nil,
			[]interface{}{"", false},
		},
		{
			"StringValueOK/Zero Length", Value.StringValueOK, Value{Type: TypeString, Data: []byte{0x00, 0x00, 0x00, 0x00}},
			nil,
			[]interface{}{"", false},
		},
		{
			"StringValueOK/Success", Value.StringValueOK, Value{Type: TypeString, Data: AppendString(nil, "hello, world!")},
			nil,
			[]interface{}{"hello, world!", true},
		},
		{
			"Document/Not Document", Value.Document, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Document", TypeString},
			nil,
		},
		{
			"Document/Insufficient Bytes", Value.Document, Value{Type: TypeEmbeddedDocument, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}),
			nil,
		},
		{
			"Document/Success", Value.Document, Value{Type: TypeEmbeddedDocument, Data: []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			nil,
			[]interface{}{Document{0x05, 0x00, 0x00, 0x00, 0x00}},
		},
		{
			"DocumentOK/Not Document", Value.DocumentOK, Value{Type: TypeString},
			nil,
			[]interface{}{Document(nil), false},
		},
		{
			"DocumentOK/Insufficient Bytes", Value.DocumentOK, Value{Type: TypeEmbeddedDocument, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			nil,
			[]interface{}{Document(nil), false},
		},
		{
			"DocumentOK/Success", Value.DocumentOK, Value{Type: TypeEmbeddedDocument, Data: []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			nil,
			[]interface{}{Document{0x05, 0x00, 0x00, 0x00, 0x00}, true},
		},
		{
			"Array/Not Array", Value.Array, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Array", TypeString},
			nil,
		},
		{
			"Array/Insufficient Bytes", Value.Array, Value{Type: TypeArray, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}),
			nil,
		},
		{
			"Array/Success", Value.Array, Value{Type: TypeArray, Data: []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			nil,
			[]interface{}{Array{0x05, 0x00, 0x00, 0x00, 0x00}},
		},
		{
			"ArrayOK/Not Array", Value.ArrayOK, Value{Type: TypeString},
			nil,
			[]interface{}{Array(nil), false},
		},
		{
			"ArrayOK/Insufficient Bytes", Value.ArrayOK, Value{Type: TypeArray, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			nil,
			[]interface{}{Array(nil), false},
		},
		{
			"ArrayOK/Success", Value.ArrayOK, Value{Type: TypeArray, Data: []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			nil,
			[]interface{}{Array{0x05, 0x00, 0x00, 0x00, 0x00}, true},
		},
		{
			"Binary/Not Binary", Value.Binary, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Binary", TypeString},
			nil,
		},
		{
			"Binary/Insufficient Bytes", Value.Binary, Value{Type: TypeBinary, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}),
			nil,
		},
		{
			"Binary/Success", Value.Binary, Value{Type: TypeBinary, Data: AppendBinary(nil, 0xFF, []byte{0x01, 0x02, 0x03})},
			nil,
			[]interface{}{byte(0xFF), []byte{0x01, 0x02, 0x03}},
		},
		{
			"BinaryOK/Not Binary", Value.BinaryOK, Value{Type: TypeString},
			nil,
			[]interface{}{byte(0x00), []byte(nil), false},
		},
		{
			"BinaryOK/Insufficient Bytes", Value.BinaryOK, Value{Type: TypeBinary, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			nil,
			[]interface{}{byte(0x00), []byte(nil), false},
		},
		{
			"BinaryOK/Success", Value.BinaryOK, Value{Type: TypeBinary, Data: AppendBinary(nil, 0xFF, []byte{0x01, 0x02, 0x03})},
			nil,
			[]interface{}{byte(0xFF), []byte{0x01, 0x02, 0x03}, true},
		},
		{
			"ObjectID/Not ObjectID", Value.ObjectID, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.ObjectID", TypeString},
			nil,
		},
		{
			"ObjectID/Insufficient Bytes", Value.ObjectID, Value{Type: TypeObjectID, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}),
			nil,
		},
		{
			"ObjectID/Success", Value.ObjectID, Value{Type: TypeObjectID, Data: AppendObjectID(nil, [12]byte{0x01, 0x02})},
			nil,
			[]interface{}{[12]byte{0x01, 0x02}},
		},
		{
			"ObjectIDOK/Not ObjectID", Value.ObjectIDOK, Value{Type: TypeString},
			nil,
			[]interface{}{[12]byte{}, false},
		},
		{
			"ObjectIDOK/Insufficient Bytes", Value.ObjectIDOK, Value{Type: TypeObjectID, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			nil,
			[]interface{}{[12]byte{}, false},
		},
		{
			"ObjectIDOK/Success", Value.ObjectIDOK, Value{Type: TypeObjectID, Data: AppendObjectID(nil, [12]byte{0x01, 0x02})},
			nil,
			[]interface{}{[12]byte{0x01, 0x02}, true},
		},
		{
			"Boolean/Not Boolean", Value.Boolean, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Boolean", TypeString},
			nil,
		},
		{
			"Boolean/Insufficient Bytes", Value.Boolean, Value{Type: TypeBoolean, Data: []byte{}},
			NewInsufficientBytesError([]byte{}, []byte{}),
			nil,
		},
		{
			"Boolean/Success", Value.Boolean, Value{Type: TypeBoolean, Data: AppendBoolean(nil, true)},
			nil,
			[]interface{}{true},
		},
		{
			"BooleanOK/Not Boolean", Value.BooleanOK, Value{Type: TypeString},
			nil,
			[]interface{}{false, false},
		},
		{
			"BooleanOK/Insufficient Bytes", Value.BooleanOK, Value{Type: TypeBoolean, Data: []byte{}},
			nil,
			[]interface{}{false, false},
		},
		{
			"BooleanOK/Success", Value.BooleanOK, Value{Type: TypeBoolean, Data: AppendBoolean(nil, true)},
			nil,
			[]interface{}{true, true},
		},
		{
			"DateTime/Not DateTime", Value.DateTime, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.DateTime", TypeString},
			nil,
		},
		{
			"DateTime/Insufficient Bytes", Value.DateTime, Value{Type: TypeDateTime, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{}, []byte{}),
			nil,
		},
		{
			"DateTime/Success", Value.DateTime, Value{Type: TypeDateTime, Data: AppendDateTime(nil, 12345)},
			nil,
			[]interface{}{int64(12345)},
		},
		{
			"DateTimeOK/Not DateTime", Value.DateTimeOK, Value{Type: TypeString},
			nil,
			[]interface{}{int64(0), false},
		},
		{
			"DateTimeOK/Insufficient Bytes", Value.DateTimeOK, Value{Type: TypeDateTime, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]interface{}{int64(0), false},
		},
		{
			"DateTimeOK/Success", Value.DateTimeOK, Value{Type: TypeDateTime, Data: AppendDateTime(nil, 12345)},
			nil,
			[]interface{}{int64(12345), true},
		},
		{
			"Time/Not DateTime", Value.Time, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Time", TypeString},
			nil,
		},
		{
			"Time/Insufficient Bytes", Value.Time, Value{Type: TypeDateTime, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"Time/Success", Value.Time, Value{Type: TypeDateTime, Data: AppendTime(nil, now)},
			nil,
			[]interface{}{now},
		},
		{
			"TimeOK/Not DateTime", Value.TimeOK, Value{Type: TypeString},
			nil,
			[]interface{}{time.Time{}, false},
		},
		{
			"TimeOK/Insufficient Bytes", Value.TimeOK, Value{Type: TypeDateTime, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]interface{}{time.Time{}, false},
		},
		{
			"TimeOK/Success", Value.TimeOK, Value{Type: TypeDateTime, Data: AppendTime(nil, now)},
			nil,
			[]interface{}{now, true},
		},
		{
			"Regex/Not Regex", Value.Regex, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Regex", TypeString},
			nil,
		},
		{
			"Regex/Insufficient Bytes", Value.Regex, Value{Type: TypeRegex, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"Regex/Success", Value.Regex, Value{Type: TypeRegex, Data: AppendRegex(nil, "/abcdefg/", "hijkl")},
			nil,
			[]interface{}{"/abcdefg/", "hijkl"},
		},
		{
			"RegexOK/Not Regex", Value.RegexOK, Value{Type: TypeString},
			nil,
			[]interface{}{"", "", false},
		},
		{
			"RegexOK/Insufficient Bytes", Value.RegexOK, Value{Type: TypeRegex, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]interface{}{"", "", false},
		},
		{
			"RegexOK/Success", Value.RegexOK, Value{Type: TypeRegex, Data: AppendRegex(nil, "/abcdefg/", "hijkl")},
			nil,
			[]interface{}{"/abcdefg/", "hijkl", true},
		},
		{
			"DBPointer/Not DBPointer", Value.DBPointer, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.DBPointer", TypeString},
			nil,
		},
		{
			"DBPointer/Insufficient Bytes", Value.DBPointer, Value{Type: TypeDBPointer, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"DBPointer/Success", Value.DBPointer, Value{Type: TypeDBPointer, Data: AppendDBPointer(nil, "foobar", oid)},
			nil,
			[]interface{}{"foobar", oid},
		},
		{
			"DBPointerOK/Not DBPointer", Value.DBPointerOK, Value{Type: TypeString},
			nil,
			[]interface{}{"", [12]byte{}, false},
		},
		{
			"DBPointerOK/Insufficient Bytes", Value.DBPointerOK, Value{Type: TypeDBPointer, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]interface{}{"", [12]byte{}, false},
		},
		{
			"DBPointerOK/Success", Value.DBPointerOK, Value{Type: TypeDBPointer, Data: AppendDBPointer(nil, "foobar", oid)},
			nil,
			[]interface{}{"foobar", oid, true},
		},
		{
			"JavaScript/Not JavaScript", Value.JavaScript, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.JavaScript", TypeString},
			nil,
		},
		{
			"JavaScript/Insufficient Bytes", Value.JavaScript, Value{Type: TypeJavaScript, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"JavaScript/Success", Value.JavaScript, Value{Type: TypeJavaScript, Data: AppendJavaScript(nil, "var hello = 'world';")},
			nil,
			[]interface{}{"var hello = 'world';"},
		},
		{
			"JavaScriptOK/Not JavaScript", Value.JavaScriptOK, Value{Type: TypeString},
			nil,
			[]interface{}{"", false},
		},
		{
			"JavaScriptOK/Insufficient Bytes", Value.JavaScriptOK, Value{Type: TypeJavaScript, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]interface{}{"", false},
		},
		{
			"JavaScriptOK/Success", Value.JavaScriptOK, Value{Type: TypeJavaScript, Data: AppendJavaScript(nil, "var hello = 'world';")},
			nil,
			[]interface{}{"var hello = 'world';", true},
		},
		{
			"Symbol/Not Symbol", Value.Symbol, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Symbol", TypeString},
			nil,
		},
		{
			"Symbol/Insufficient Bytes", Value.Symbol, Value{Type: TypeSymbol, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"Symbol/Success", Value.Symbol, Value{Type: TypeSymbol, Data: AppendSymbol(nil, "symbol123456")},
			nil,
			[]interface{}{"symbol123456"},
		},
		{
			"SymbolOK/Not Symbol", Value.SymbolOK, Value{Type: TypeString},
			nil,
			[]interface{}{"", false},
		},
		{
			"SymbolOK/Insufficient Bytes", Value.SymbolOK, Value{Type: TypeSymbol, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]interface{}{"", false},
		},
		{
			"SymbolOK/Success", Value.SymbolOK, Value{Type: TypeSymbol, Data: AppendSymbol(nil, "symbol123456")},
			nil,
			[]interface{}{"symbol123456", true},
		},
		{
			"CodeWithScope/Not CodeWithScope", Value.CodeWithScope, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.CodeWithScope", TypeString},
			nil,
		},
		{
			"CodeWithScope/Insufficient Bytes", Value.CodeWithScope, Value{Type: TypeCodeWithScope, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"CodeWithScope/Success", Value.CodeWithScope, Value{Type: TypeCodeWithScope, Data: AppendCodeWithScope(nil, "var hello = 'world';", Document{0x05, 0x00, 0x00, 0x00, 0x00})},
			nil,
			[]interface{}{"var hello = 'world';", Document{0x05, 0x00, 0x00, 0x00, 0x00}},
		},
		{
			"CodeWithScopeOK/Not CodeWithScope", Value.CodeWithScopeOK, Value{Type: TypeString},
			nil,
			[]interface{}{"", Document(nil), false},
		},
		{
			"CodeWithScopeOK/Insufficient Bytes", Value.CodeWithScopeOK, Value{Type: TypeCodeWithScope, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]interface{}{"", Document(nil), false},
		},
		{
			"CodeWithScopeOK/Success", Value.CodeWithScopeOK, Value{Type: TypeCodeWithScope, Data: AppendCodeWithScope(nil, "var hello = 'world';", Document{0x05, 0x00, 0x00, 0x00, 0x00})},
			nil,
			[]interface{}{"var hello = 'world';", Document{0x05, 0x00, 0x00, 0x00, 0x00}, true},
		},
		{
			"Int32/Not Int32", Value.Int32, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Int32", TypeString},
			nil,
		},
		{
			"Int32/Insufficient Bytes", Value.Int32, Value{Type: TypeInt32, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"Int32/Success", Value.Int32, Value{Type: TypeInt32, Data: AppendInt32(nil, 1234)},
			nil,
			[]interface{}{int32(1234)},
		},
		{
			"Int32OK/Not Int32", Value.Int32OK, Value{Type: TypeString},
			nil,
			[]interface{}{int32(0), false},
		},
		{
			"Int32OK/Insufficient Bytes", Value.Int32OK, Value{Type: TypeInt32, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]interface{}{int32(0), false},
		},
		{
			"Int32OK/Success", Value.Int32OK, Value{Type: TypeInt32, Data: AppendInt32(nil, 1234)},
			nil,
			[]interface{}{int32(1234), true},
		},
		{
			"Timestamp/Not Timestamp", Value.Timestamp, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Timestamp", TypeString},
			nil,
		},
		{
			"Timestamp/Insufficient Bytes", Value.Timestamp, Value{Type: TypeTimestamp, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"Timestamp/Success", Value.Timestamp, Value{Type: TypeTimestamp, Data: AppendTimestamp(nil, 12345, 67890)},
			nil,
			[]interface{}{uint32(12345), uint32(67890)},
		},
		{
			"TimestampOK/Not Timestamp", Value.TimestampOK, Value{Type: TypeString},
			nil,
			[]interface{}{uint32(0), uint32(0), false},
		},
		{
			"TimestampOK/Insufficient Bytes", Value.TimestampOK, Value{Type: TypeTimestamp, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]interface{}{uint32(0), uint32(0), false},
		},
		{
			"TimestampOK/Success", Value.TimestampOK, Value{Type: TypeTimestamp, Data: AppendTimestamp(nil, 12345, 67890)},
			nil,
			[]interface{}{uint32(12345), uint32(67890), true},
		},
		{
			"Int64/Not Int64", Value.Int64, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Int64", TypeString},
			nil,
		},
		{
			"Int64/Insufficient Bytes", Value.Int64, Value{Type: TypeInt64, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"Int64/Success", Value.Int64, Value{Type: TypeInt64, Data: AppendInt64(nil, 1234567890)},
			nil,
			[]interface{}{int64(1234567890)},
		},
		{
			"Int64OK/Not Int64", Value.Int64OK, Value{Type: TypeString},
			nil,
			[]interface{}{int64(0), false},
		},
		{
			"Int64OK/Insufficient Bytes", Value.Int64OK, Value{Type: TypeInt64, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]interface{}{int64(0), false},
		},
		{
			"Int64OK/Success", Value.Int64OK, Value{Type: TypeInt64, Data: AppendInt64(nil, 1234567890)},
			nil,
			[]interface{}{int64(1234567890), true},
		},
		{
			"Decimal128/Not Decimal128", Value.Decimal128, Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Decimal128", TypeString},
			nil,
		},
		{
			"Decimal128/Insufficient Bytes", Value.Decimal128, Value{Type: TypeDecimal128, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"Decimal128/Success", Value.Decimal128, Value{Type: TypeDecimal128, Data: AppendDecimal128(nil, 12345, 67890)},
			nil,
			[]interface{}{uint64(12345), uint64(67890)},
		},
		{
			"Decimal128OK/Not Decimal128", Value.Decimal128OK, Value{Type: TypeString},
			nil,
			[]interface{}{uint64(0), uint64(0), false},
		},
		{
			"Decimal128OK/Insufficient Bytes", Value.Decimal128OK, Value{Type: TypeDecimal128, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]interface{}{uint64(0), uint64(0), false},
		},
		{
			"Decimal128OK/Success", Value.Decimal128OK, Value{Type: TypeDecimal128, Data: AppendDecimal128(nil, 12345, 67890)},
			nil,
			[]interface{}{uint64(12345), uint64(67890), true},
		},
		{
			"Timestamp.String/Success", Value.String, Value{Type: TypeTimestamp, Data: AppendTimestamp(nil, 12345, 67890)},
			nil,
			[]interface{}{"{\"$timestamp\":{\"t\":12345,\"i\":67890}}"},
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				err := recover()
				if !cmp.Equal(err, tc.panicErr, cmp.Comparer(compareErrors)) {
					t.Errorf("Did not receive expected panic error. got %v; want %v", err, tc.panicErr)
				}
			}()

			fn := reflect.ValueOf(tc.fn)
			if fn.Kind() != reflect.Func || fn.Type().NumIn() != 1 || fn.Type().In(0) != reflect.TypeOf(Value{}) {
				t.Fatalf("test case field fn must be a function with 1 parameter that is a Value, but it is %v", fn.Type())
			}
			got := fn.Call([]reflect.Value{reflect.ValueOf(tc.val)})
			want := make([]reflect.Value, 0, len(tc.ret))
			for _, ret := range tc.ret {
				want = append(want, reflect.ValueOf(ret))
			}
			if len(got) != len(want) {
				t.Fatalf("incorrect number of values returned. got %d; want %d", len(got), len(want))
			}

			for idx := range got {
				gotv, wantv := got[idx].Interface(), want[idx].Interface()
				if !cmp.Equal(gotv, wantv) {
					t.Errorf("return values at index %d are not equal. got %v; want %v", idx, gotv, wantv)
				}
			}
		})
	}
}

func TestValue_StringN(t *testing.T) {
	var buf strings.Builder
	for i := 0; i < 16000000; i++ {
		buf.WriteString("abcdefgh")
	}
	str1k := buf.String()
	str128 := str1k[:128]
	testObjectID := [12]byte{0x60, 0xd4, 0xc2, 0x1f, 0x4e, 0x60, 0x4a, 0x0c, 0x8b, 0x2e, 0x9c, 0x3f}

	testCases := []struct {
		description string
		n           int
		val         Value
		want        string
	}{
		// n = 0 cases
		{"n=0, single value", 0, Value{
			Type: TypeString, Data: AppendString(nil, "abcdefgh")}, ""},

		{"n=0, large string value", 0, Value{
			Type: TypeString, Data: AppendString(nil, "abcdefgh")}, ""},

		{"n=0, value with special characters", 0, Value{
			Type: TypeString, Data: AppendString(nil, "!@#$%^&*()")}, ""},

		// n < 0 cases
		{"n<0, single value", -1, Value{
			Type: TypeString, Data: AppendString(nil, "abcdefgh")}, ""},

		{"n<0, large string value", -1, Value{
			Type: TypeString, Data: AppendString(nil, "abcdefgh")}, ""},

		{"n<0, value with special characters", -1, Value{
			Type: TypeString, Data: AppendString(nil, "!@#$%^&*()")}, ""},

		// n > 0 cases
		{"n>0, string LT n", 4, Value{
			Type: TypeString, Data: AppendString(nil, "foo")}, `"foo`},

		{"n>0, string GT n", 10, Value{
			Type: TypeString, Data: AppendString(nil, "foo")}, `"foo"`},

		{"n>0, string EQ n", 5, Value{
			Type: TypeString, Data: AppendString(nil, "foo")}, `"foo"`},

		{"n>0, multi-byte string LT n", 10, Value{
			Type: TypeString, Data: AppendString(nil, "𨉟呐㗂越")}, `"𨉟呐`},

		{"n>0, multi-byte string GT n", 21, Value{
			Type: TypeString, Data: AppendString(nil, "𨉟呐㗂越")}, `"𨉟呐㗂越"`},

		{"n>0, multi-byte string EQ n", 15, Value{
			Type: TypeString, Data: AppendString(nil, "𨉟呐㗂越")}, `"𨉟呐㗂越"`},

		{"n>0, multi-byte string exact character boundary", 6, Value{
			Type: TypeString, Data: AppendString(nil, "𨉟呐㗂越")}, `"𨉟`},

		{"n>0, multi-byte string mid character", 8, Value{
			Type: TypeString, Data: AppendString(nil, "𨉟呐㗂越")}, `"𨉟`},

		{"n>0, multi-byte string edge case", 10, Value{
			Type: TypeString, Data: AppendString(nil, "𨉟呐㗂越")}, `"𨉟呐`},

		{"n>0, single value", 10, Value{
			Type: TypeString, Data: AppendString(nil, str128)}, `"abcdefgha`},

		{"n>0, large string value", 10, Value{
			Type: TypeString, Data: AppendString(nil, str1k)}, `"abcdefgha`},

		{"n>0, value with special characters", 5, Value{
			Type: TypeString, Data: AppendString(nil, "!@#$%^&*()")}, `"!@#$`},

		// Extended cases for each type
		{"n>0, TypeEmbeddedDocument", 10, Value{
			Type: TypeEmbeddedDocument, Data: BuildDocument(nil,
				AppendStringElement(nil, "key", "value"))}, `{"key": "v`},

		{"n>0, TypeArray", 10, Value{
			Type: TypeArray,
			Data: BuildArray(nil,
				Value{
					Type: TypeString,
					Data: AppendString(nil, "abc"),
				},
				Value{
					Type: TypeInt32,
					Data: AppendInt32(nil, 123),
				},
				Value{
					Type: TypeBoolean,
					Data: AppendBoolean(nil, true),
				},
			)}, `["abc",{"$`},

		{"n>0, TypeDouble", 10, Value{
			Type: TypeDouble, Data: AppendDouble(nil, 123.456)}, `{"$numberD`},

		{"n>0, TypeBinary", 10, Value{
			Type: TypeBinary, Data: AppendBinary(nil, 0x00, []byte{0x01, 0x02, 0x03})}, `{"$binary"`},

		{"n>0, TypeUndefined", 10, Value{
			Type: TypeUndefined}, `{"$undefin`},

		{"n>0, TypeObjectID", 10, Value{
			Type: TypeObjectID, Data: AppendObjectID(nil, testObjectID)}, `{"$oid":"6`},

		{"n>0, TypeBoolean", 3, Value{
			Type: TypeBoolean, Data: AppendBoolean(nil, true)}, `tru`},

		{"n>0, TypeDateTime", 10, Value{
			Type: TypeDateTime, Data: AppendDateTime(nil, 1234567890)}, `{"$date":{`},

		{"n>0, TypeNull", 3, Value{
			Type: TypeNull}, `nul`},

		{"n>0, TypeRegex", 10, Value{
			Type: TypeRegex, Data: AppendRegex(nil, "pattern", "options")}, `{"$regular`},

		{"n>0, TypeDBPointer", 15, Value{
			Type: TypeDBPointer, Data: AppendDBPointer(nil, "namespace", testObjectID)}, `{"$dbPointer":{`},

		{"n>0, TypeJavaScript", 15, Value{
			Type: TypeJavaScript, Data: AppendJavaScript(nil, "code")}, `{"$code":"code"`},

		{"n>0, TypeSymbol", 10, Value{
			Type: TypeSymbol, Data: AppendSymbol(nil, "symbol")}, `{"$symbol"`},

		{"n>0, TypeCodeWithScope", 10, Value{
			Type: TypeCodeWithScope, Data: AppendCodeWithScope(nil, "code", BuildDocument(nil,
				AppendStringElement(nil, "key", "value")))}, `{"$code":c`},

		{"n>0, TypeInt32", 10, Value{
			Type: TypeInt32, Data: AppendInt32(nil, 123)}, `{"$numberI`},

		{"n>0, TypeTimestamp", 10, Value{
			Type: TypeTimestamp, Data: AppendTimestamp(nil, 123, 456)}, `{"$timesta`},

		{"n>0, TypeInt64", 10, Value{
			Type: TypeInt64, Data: AppendInt64(nil, 1234567890)}, `{"$numberL`},

		{"n>0, TypeDecimal128", 10, Value{
			Type: TypeDecimal128, Data: AppendDecimal128(nil, 0x3040000000000000, 0x0000000000000000)}, `{"$numberD`},

		{"n>0, TypeMinKey", 10, Value{
			Type: TypeMinKey}, `{"$minKey"`},

		{"n>0, TypeMaxKey", 10, Value{
			Type: TypeMaxKey}, `{"$maxKey"`},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			got := tc.val.StringN(tc.n)
			assert.Equal(t, tc.want, got)
			if tc.n >= 0 {
				assert.LessOrEqual(t, len(got), tc.n)
			}
		})
	}
}
