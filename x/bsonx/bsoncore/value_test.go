// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"fmt"
	"reflect"
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
		fn       any
		val      Value
		panicErr error
		ret      []any
	}{
		{
			"Double/Not Double", Value.Double,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Double", TypeString},
			nil,
		},
		{
			"Double/Insufficient Bytes", Value.Double,
			Value{Type: TypeDouble, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}),
			nil,
		},
		{
			"Double/Success", Value.Double,
			Value{Type: TypeDouble, Data: AppendDouble(nil, 3.14159)},
			nil,
			[]any{float64(3.14159)},
		},
		{
			"DoubleOK/Not Double", Value.DoubleOK,
			Value{Type: TypeString},
			nil,
			[]any{float64(0), false},
		},
		{
			"DoubleOK/Insufficient Bytes", Value.DoubleOK,
			Value{Type: TypeDouble, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			nil,
			[]any{float64(0), false},
		},
		{
			"DoubleOK/Success", Value.DoubleOK,
			Value{Type: TypeDouble, Data: AppendDouble(nil, 3.14159)},
			nil,
			[]any{float64(3.14159), true},
		},
		{
			"StringValue/Not String", Value.StringValue,
			Value{Type: TypeDouble},
			ElementTypeError{"bsoncore.Value.StringValue", TypeDouble},
			nil,
		},
		{
			"StringValue/Insufficient Bytes", Value.StringValue,
			Value{Type: TypeString, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}),
			nil,
		},
		{
			"StringValue/Zero Length", Value.StringValue,
			Value{Type: TypeString, Data: []byte{0x00, 0x00, 0x00, 0x00}},
			NewInsufficientBytesError([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0x00, 0x00, 0x00, 0x00}),
			nil,
		},
		{
			"StringValue/Success", Value.StringValue,
			Value{Type: TypeString, Data: AppendString(nil, "hello, world!")},
			nil,
			[]any{"hello, world!"},
		},
		{
			"StringValueOK/Not String", Value.StringValueOK,
			Value{Type: TypeDouble},
			nil,
			[]any{"", false},
		},
		{
			"StringValueOK/Insufficient Bytes", Value.StringValueOK,
			Value{Type: TypeString, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			nil,
			[]any{"", false},
		},
		{
			"StringValueOK/Zero Length", Value.StringValueOK,
			Value{Type: TypeString, Data: []byte{0x00, 0x00, 0x00, 0x00}},
			nil,
			[]any{"", false},
		},
		{
			"StringValueOK/Success", Value.StringValueOK,
			Value{Type: TypeString, Data: AppendString(nil, "hello, world!")},
			nil,
			[]any{"hello, world!", true},
		},
		{
			"Document/Not Document", Value.Document,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Document", TypeString},
			nil,
		},
		{
			"Document/Insufficient Bytes", Value.Document,
			Value{Type: TypeEmbeddedDocument, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}),
			nil,
		},
		{
			"Document/Success", Value.Document,
			Value{Type: TypeEmbeddedDocument, Data: []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			nil,
			[]any{Document{0x05, 0x00, 0x00, 0x00, 0x00}},
		},
		{
			"DocumentOK/Not Document", Value.DocumentOK,
			Value{Type: TypeString},
			nil,
			[]any{Document(nil), false},
		},
		{
			"DocumentOK/Insufficient Bytes", Value.DocumentOK,
			Value{Type: TypeEmbeddedDocument, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			nil,
			[]any{Document(nil), false},
		},
		{
			"DocumentOK/Success", Value.DocumentOK,
			Value{Type: TypeEmbeddedDocument, Data: []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			nil,
			[]any{Document{0x05, 0x00, 0x00, 0x00, 0x00}, true},
		},
		{
			"Array/Not Array", Value.Array,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Array", TypeString},
			nil,
		},
		{
			"Array/Insufficient Bytes", Value.Array,
			Value{Type: TypeArray, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}),
			nil,
		},
		{
			"Array/Success", Value.Array,
			Value{Type: TypeArray, Data: []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			nil,
			[]any{Array{0x05, 0x00, 0x00, 0x00, 0x00}},
		},
		{
			"ArrayOK/Not Array", Value.ArrayOK,
			Value{Type: TypeString},
			nil,
			[]any{Array(nil), false},
		},
		{
			"ArrayOK/Insufficient Bytes", Value.ArrayOK,
			Value{Type: TypeArray, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			nil,
			[]any{Array(nil), false},
		},
		{
			"ArrayOK/Success", Value.ArrayOK,
			Value{Type: TypeArray, Data: []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			nil,
			[]any{Array{0x05, 0x00, 0x00, 0x00, 0x00}, true},
		},
		{
			"Binary/Not Binary", Value.Binary,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Binary", TypeString},
			nil,
		},
		{
			"Binary/Insufficient Bytes", Value.Binary,
			Value{Type: TypeBinary, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}),
			nil,
		},
		{
			"Binary/Success", Value.Binary,
			Value{Type: TypeBinary, Data: AppendBinary(nil, 0xFF, []byte{0x01, 0x02, 0x03})},
			nil,
			[]any{byte(0xFF), []byte{0x01, 0x02, 0x03}},
		},
		{
			"BinaryOK/Not Binary", Value.BinaryOK,
			Value{Type: TypeString},
			nil,
			[]any{byte(0x00), []byte(nil), false},
		},
		{
			"BinaryOK/Insufficient Bytes", Value.BinaryOK,
			Value{Type: TypeBinary, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			nil,
			[]any{byte(0x00), []byte(nil), false},
		},
		{
			"BinaryOK/Success", Value.BinaryOK,
			Value{Type: TypeBinary, Data: AppendBinary(nil, 0xFF, []byte{0x01, 0x02, 0x03})},
			nil,
			[]any{byte(0xFF), []byte{0x01, 0x02, 0x03}, true},
		},
		{
			"ObjectID/Not ObjectID", Value.ObjectID,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.ObjectID", TypeString},
			nil,
		},
		{
			"ObjectID/Insufficient Bytes", Value.ObjectID,
			Value{Type: TypeObjectID, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}),
			nil,
		},
		{
			"ObjectID/Success", Value.ObjectID,
			Value{Type: TypeObjectID, Data: AppendObjectID(nil, [12]byte{0x01, 0x02})},
			nil,
			[]any{[12]byte{0x01, 0x02}},
		},
		{
			"ObjectIDOK/Not ObjectID", Value.ObjectIDOK,
			Value{Type: TypeString},
			nil,
			[]any{[12]byte{}, false},
		},
		{
			"ObjectIDOK/Insufficient Bytes", Value.ObjectIDOK,
			Value{Type: TypeObjectID, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			nil,
			[]any{[12]byte{}, false},
		},
		{
			"ObjectIDOK/Success", Value.ObjectIDOK,
			Value{Type: TypeObjectID, Data: AppendObjectID(nil, [12]byte{0x01, 0x02})},
			nil,
			[]any{[12]byte{0x01, 0x02}, true},
		},
		{
			"Boolean/Not Boolean", Value.Boolean,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Boolean", TypeString},
			nil,
		},
		{
			"Boolean/Insufficient Bytes", Value.Boolean,
			Value{Type: TypeBoolean, Data: []byte{}},
			NewInsufficientBytesError([]byte{}, []byte{}),
			nil,
		},
		{
			"Boolean/Success", Value.Boolean,
			Value{Type: TypeBoolean, Data: AppendBoolean(nil, true)},
			nil,
			[]any{true},
		},
		{
			"BooleanOK/Not Boolean", Value.BooleanOK,
			Value{Type: TypeString},
			nil,
			[]any{false, false},
		},
		{
			"BooleanOK/Insufficient Bytes", Value.BooleanOK,
			Value{Type: TypeBoolean, Data: []byte{}},
			nil,
			[]any{false, false},
		},
		{
			"BooleanOK/Success", Value.BooleanOK,
			Value{Type: TypeBoolean, Data: AppendBoolean(nil, true)},
			nil,
			[]any{true, true},
		},
		{
			"DateTime/Not DateTime", Value.DateTime,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.DateTime", TypeString},
			nil,
		},
		{
			"DateTime/Insufficient Bytes", Value.DateTime,
			Value{Type: TypeDateTime, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{}, []byte{}),
			nil,
		},
		{
			"DateTime/Success", Value.DateTime,
			Value{Type: TypeDateTime, Data: AppendDateTime(nil, 12345)},
			nil,
			[]any{int64(12345)},
		},
		{
			"DateTimeOK/Not DateTime", Value.DateTimeOK,
			Value{Type: TypeString},
			nil,
			[]any{int64(0), false},
		},
		{
			"DateTimeOK/Insufficient Bytes", Value.DateTimeOK,
			Value{Type: TypeDateTime, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{int64(0), false},
		},
		{
			"DateTimeOK/Success", Value.DateTimeOK,
			Value{Type: TypeDateTime, Data: AppendDateTime(nil, 12345)},
			nil,
			[]any{int64(12345), true},
		},
		{
			"Time/Not DateTime", Value.Time,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Time", TypeString},
			nil,
		},
		{
			"Time/Insufficient Bytes", Value.Time,
			Value{Type: TypeDateTime, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"Time/Success", Value.Time,
			Value{Type: TypeDateTime, Data: AppendTime(nil, now)},
			nil,
			[]any{now},
		},
		{
			"TimeOK/Not DateTime", Value.TimeOK,
			Value{Type: TypeString},
			nil,
			[]any{time.Time{}, false},
		},
		{
			"TimeOK/Insufficient Bytes", Value.TimeOK,
			Value{Type: TypeDateTime, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{time.Time{}, false},
		},
		{
			"TimeOK/Success", Value.TimeOK,
			Value{Type: TypeDateTime, Data: AppendTime(nil, now)},
			nil,
			[]any{now, true},
		},
		{
			"Regex/Not Regex", Value.Regex,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Regex", TypeString},
			nil,
		},
		{
			"Regex/Insufficient Bytes", Value.Regex,
			Value{Type: TypeRegex, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"Regex/Success", Value.Regex,
			Value{Type: TypeRegex, Data: AppendRegex(nil, "/abcdefg/", "hijkl")},
			nil,
			[]any{"/abcdefg/", "hijkl"},
		},
		{
			"RegexOK/Not Regex", Value.RegexOK,
			Value{Type: TypeString},
			nil,
			[]any{"", "", false},
		},
		{
			"RegexOK/Insufficient Bytes", Value.RegexOK,
			Value{Type: TypeRegex, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{"", "", false},
		},
		{
			"RegexOK/Success", Value.RegexOK,
			Value{Type: TypeRegex, Data: AppendRegex(nil, "/abcdefg/", "hijkl")},
			nil,
			[]any{"/abcdefg/", "hijkl", true},
		},
		{
			"DBPointer/Not DBPointer", Value.DBPointer,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.DBPointer", TypeString},
			nil,
		},
		{
			"DBPointer/Insufficient Bytes", Value.DBPointer,
			Value{Type: TypeDBPointer, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"DBPointer/Success", Value.DBPointer,
			Value{Type: TypeDBPointer, Data: AppendDBPointer(nil, "foobar", oid)},
			nil,
			[]any{"foobar", oid},
		},
		{
			"DBPointerOK/Not DBPointer", Value.DBPointerOK,
			Value{Type: TypeString},
			nil,
			[]any{"", [12]byte{}, false},
		},
		{
			"DBPointerOK/Insufficient Bytes", Value.DBPointerOK,
			Value{Type: TypeDBPointer, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{"", [12]byte{}, false},
		},
		{
			"DBPointerOK/Success", Value.DBPointerOK,
			Value{Type: TypeDBPointer, Data: AppendDBPointer(nil, "foobar", oid)},
			nil,
			[]any{"foobar", oid, true},
		},
		{
			"JavaScript/Not JavaScript", Value.JavaScript,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.JavaScript", TypeString},
			nil,
		},
		{
			"JavaScript/Insufficient Bytes", Value.JavaScript,
			Value{Type: TypeJavaScript, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"JavaScript/Success", Value.JavaScript,
			Value{Type: TypeJavaScript, Data: AppendJavaScript(nil, "var hello = 'world';")},
			nil,
			[]any{"var hello = 'world';"},
		},
		{
			"JavaScriptOK/Not JavaScript", Value.JavaScriptOK,
			Value{Type: TypeString},
			nil,
			[]any{"", false},
		},
		{
			"JavaScriptOK/Insufficient Bytes", Value.JavaScriptOK,
			Value{Type: TypeJavaScript, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{"", false},
		},
		{
			"JavaScriptOK/Success", Value.JavaScriptOK,
			Value{Type: TypeJavaScript, Data: AppendJavaScript(nil, "var hello = 'world';")},
			nil,
			[]any{"var hello = 'world';", true},
		},
		{
			"Symbol/Not Symbol", Value.Symbol,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Symbol", TypeString},
			nil,
		},
		{
			"Symbol/Insufficient Bytes", Value.Symbol,
			Value{Type: TypeSymbol, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"Symbol/Success", Value.Symbol,
			Value{Type: TypeSymbol, Data: AppendSymbol(nil, "symbol123456")},
			nil,
			[]any{"symbol123456"},
		},
		{
			"SymbolOK/Not Symbol", Value.SymbolOK,
			Value{Type: TypeString},
			nil,
			[]any{"", false},
		},
		{
			"SymbolOK/Insufficient Bytes", Value.SymbolOK,
			Value{Type: TypeSymbol, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{"", false},
		},
		{
			"SymbolOK/Success", Value.SymbolOK,
			Value{Type: TypeSymbol, Data: AppendSymbol(nil, "symbol123456")},
			nil,
			[]any{"symbol123456", true},
		},
		{
			"CodeWithScope/Not CodeWithScope", Value.CodeWithScope,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.CodeWithScope", TypeString},
			nil,
		},
		{
			"CodeWithScope/Insufficient Bytes", Value.CodeWithScope,
			Value{Type: TypeCodeWithScope, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"CodeWithScope/Success", Value.CodeWithScope,
			Value{Type: TypeCodeWithScope, Data: AppendCodeWithScope(nil, "var hello = 'world';", Document{0x05, 0x00, 0x00, 0x00, 0x00})},
			nil,
			[]any{"var hello = 'world';", Document{0x05, 0x00, 0x00, 0x00, 0x00}},
		},
		{
			"CodeWithScopeOK/Not CodeWithScope", Value.CodeWithScopeOK,
			Value{Type: TypeString},
			nil,
			[]any{"", Document(nil), false},
		},
		{
			"CodeWithScopeOK/Insufficient Bytes", Value.CodeWithScopeOK,
			Value{Type: TypeCodeWithScope, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{"", Document(nil), false},
		},
		{
			"CodeWithScopeOK/Success", Value.CodeWithScopeOK,
			Value{Type: TypeCodeWithScope, Data: AppendCodeWithScope(nil, "var hello = 'world';", Document{0x05, 0x00, 0x00, 0x00, 0x00})},
			nil,
			[]any{"var hello = 'world';", Document{0x05, 0x00, 0x00, 0x00, 0x00}, true},
		},
		{
			"Int32/Not Int32", Value.Int32,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Int32", TypeString},
			nil,
		},
		{
			"Int32/Insufficient Bytes", Value.Int32,
			Value{Type: TypeInt32, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"Int32/Success", Value.Int32,
			Value{Type: TypeInt32, Data: AppendInt32(nil, 1234)},
			nil,
			[]any{int32(1234)},
		},
		{
			"Int32OK/Not Int32", Value.Int32OK,
			Value{Type: TypeString},
			nil,
			[]any{int32(0), false},
		},
		{
			"Int32OK/Insufficient Bytes", Value.Int32OK,
			Value{Type: TypeInt32, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{int32(0), false},
		},
		{
			"Int32OK/Success", Value.Int32OK,
			Value{Type: TypeInt32, Data: AppendInt32(nil, 1234)},
			nil,
			[]any{int32(1234), true},
		},
		{
			"Timestamp/Not Timestamp", Value.Timestamp,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Timestamp", TypeString},
			nil,
		},
		{
			"Timestamp/Insufficient Bytes", Value.Timestamp,
			Value{Type: TypeTimestamp, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"Timestamp/Success", Value.Timestamp,
			Value{Type: TypeTimestamp, Data: AppendTimestamp(nil, 12345, 67890)},
			nil,
			[]any{uint32(12345), uint32(67890)},
		},
		{
			"TimestampOK/Not Timestamp", Value.TimestampOK,
			Value{Type: TypeString},
			nil,
			[]any{uint32(0), uint32(0), false},
		},
		{
			"TimestampOK/Insufficient Bytes", Value.TimestampOK,
			Value{Type: TypeTimestamp, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{uint32(0), uint32(0), false},
		},
		{
			"TimestampOK/Success", Value.TimestampOK,
			Value{Type: TypeTimestamp, Data: AppendTimestamp(nil, 12345, 67890)},
			nil,
			[]any{uint32(12345), uint32(67890), true},
		},
		{
			"Int64/Not Int64", Value.Int64,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Int64", TypeString},
			nil,
		},
		{
			"Int64/Insufficient Bytes", Value.Int64,
			Value{Type: TypeInt64, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"Int64/Success", Value.Int64,
			Value{Type: TypeInt64, Data: AppendInt64(nil, 1234567890)},
			nil,
			[]any{int64(1234567890)},
		},
		{
			"Int64OK/Not Int64", Value.Int64OK,
			Value{Type: TypeString},
			nil,
			[]any{int64(0), false},
		},
		{
			"Int64OK/Insufficient Bytes", Value.Int64OK,
			Value{Type: TypeInt64, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{int64(0), false},
		},
		{
			"Int64OK/Success", Value.Int64OK,
			Value{Type: TypeInt64, Data: AppendInt64(nil, 1234567890)},
			nil,
			[]any{int64(1234567890), true},
		},
		{
			"Decimal128/Not Decimal128", Value.Decimal128,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.Decimal128", TypeString},
			nil,
		},
		{
			"Decimal128/Insufficient Bytes", Value.Decimal128,
			Value{Type: TypeDecimal128, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"Decimal128/Success", Value.Decimal128,
			Value{Type: TypeDecimal128, Data: AppendDecimal128(nil, 12345, 67890)},
			nil,
			[]any{uint64(12345), uint64(67890)},
		},
		{
			"Decimal128OK/Not Decimal128", Value.Decimal128OK,
			Value{Type: TypeString},
			nil,
			[]any{uint64(0), uint64(0), false},
		},
		{
			"Decimal128OK/Insufficient Bytes", Value.Decimal128OK,
			Value{Type: TypeDecimal128, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{uint64(0), uint64(0), false},
		},
		{
			"Decimal128OK/Success", Value.Decimal128OK,
			Value{Type: TypeDecimal128, Data: AppendDecimal128(nil, 12345, 67890)},
			nil,
			[]any{uint64(12345), uint64(67890), true},
		},
		{
			"AsInt32/Not Number", Value.AsInt32,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.AsInt32", TypeString},
			nil,
		},
		{
			"AsInt32/Double/Insufficient Bytes", Value.AsInt32,
			Value{Type: TypeDouble, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"AsInt32/Int32/Insufficient Bytes", Value.AsInt32,
			Value{Type: TypeInt32, Data: []byte{0x01, 0x02}},
			NewInsufficientBytesError([]byte{0x01, 0x02}, []byte{0x01, 0x02}),
			nil,
		},
		{
			"AsInt32/Int64/Insufficient Bytes", Value.AsInt32,
			Value{Type: TypeInt64, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"AsInt32/Decimal128", Value.AsInt32,
			Value{Type: TypeDecimal128, Data: AppendDecimal128(nil, 12345, 67890)},
			ElementTypeError{"bsoncore.Value.AsInt32", TypeDecimal128},
			nil,
		},
		{
			"AsInt32/From Double", Value.AsInt32,
			Value{Type: TypeDouble, Data: AppendDouble(nil, 42.7)},
			nil,
			[]any{int32(42)},
		},
		{
			"AsInt32/From Int32", Value.AsInt32,
			Value{Type: TypeInt32, Data: AppendInt32(nil, 12345)},
			nil,
			[]any{int32(12345)},
		},
		{
			"AsInt32/From Int64", Value.AsInt32,
			Value{Type: TypeInt64, Data: AppendInt64(nil, 98765)},
			nil,
			[]any{int32(98765)},
		},
		{
			"AsInt32OK/Not Number", Value.AsInt32OK,
			Value{Type: TypeString},
			nil,
			[]any{int32(0), false},
		},
		{
			"AsInt32OK/Double/Insufficient Bytes", Value.AsInt32OK,
			Value{Type: TypeDouble, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{int32(0), false},
		},
		{
			"AsInt32OK/Int32/Insufficient Bytes", Value.AsInt32OK,
			Value{Type: TypeInt32, Data: []byte{0x01, 0x02}},
			nil,
			[]any{int32(0), false},
		},
		{
			"AsInt32OK/Int64/Insufficient Bytes", Value.AsInt32OK,
			Value{Type: TypeInt64, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{int32(0), false},
		},
		{
			"AsInt32OK/Decimal128", Value.AsInt32OK,
			Value{Type: TypeDecimal128, Data: AppendDecimal128(nil, 12345, 67890)},
			nil,
			[]any{int32(0), false},
		},
		{
			"AsInt32OK/From Double", Value.AsInt32OK,
			Value{Type: TypeDouble, Data: AppendDouble(nil, 42.7)},
			nil,
			[]any{int32(42), true},
		},
		{
			"AsInt32OK/From Int32", Value.AsInt32OK,
			Value{Type: TypeInt32, Data: AppendInt32(nil, 12345)},
			nil,
			[]any{int32(12345), true},
		},
		{
			"AsInt32OK/From Int64", Value.AsInt32OK,
			Value{Type: TypeInt64, Data: AppendInt64(nil, 98765)},
			nil,
			[]any{int32(98765), true},
		},
		{
			"AsInt64/Not Number", Value.AsInt64,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.AsInt64", TypeString},
			nil,
		},
		{
			"AsInt64/Double/Insufficient Bytes", Value.AsInt64,
			Value{Type: TypeDouble, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"AsInt64/Int32/Insufficient Bytes", Value.AsInt64,
			Value{Type: TypeInt32, Data: []byte{0x01, 0x02}},
			NewInsufficientBytesError([]byte{0x01, 0x02}, []byte{0x01, 0x02}),
			nil,
		},
		{
			"AsInt64/Int64/Insufficient Bytes", Value.AsInt64,
			Value{Type: TypeInt64, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"AsInt64/Decimal128", Value.AsInt64,
			Value{Type: TypeDecimal128, Data: AppendDecimal128(nil, 12345, 67890)},
			ElementTypeError{"bsoncore.Value.AsInt64", TypeDecimal128},
			nil,
		},
		{
			"AsInt64/From Double", Value.AsInt64,
			Value{Type: TypeDouble, Data: AppendDouble(nil, 123456.9)},
			nil,
			[]any{int64(123456)},
		},
		{
			"AsInt64/From Int32", Value.AsInt64,
			Value{Type: TypeInt32, Data: AppendInt32(nil, 12345)},
			nil,
			[]any{int64(12345)},
		},
		{
			"AsInt64/From Int64", Value.AsInt64,
			Value{Type: TypeInt64, Data: AppendInt64(nil, 9876543210)},
			nil,
			[]any{int64(9876543210)},
		},
		{
			"AsInt64OK/Not Number", Value.AsInt64OK,
			Value{Type: TypeString},
			nil,
			[]any{int64(0), false},
		},
		{
			"AsInt64OK/Double/Insufficient Bytes", Value.AsInt64OK,
			Value{Type: TypeDouble, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{int64(0), false},
		},
		{
			"AsInt64OK/Int32/Insufficient Bytes", Value.AsInt64OK,
			Value{Type: TypeInt32, Data: []byte{0x01, 0x02}},
			nil,
			[]any{int64(0), false},
		},
		{
			"AsInt64OK/Int64/Insufficient Bytes", Value.AsInt64OK,
			Value{Type: TypeInt64, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{int64(0), false},
		},
		{
			"AsInt64OK/Decimal128", Value.AsInt64OK,
			Value{Type: TypeDecimal128, Data: AppendDecimal128(nil, 12345, 67890)},
			nil,
			[]any{int64(0), false},
		},
		{
			"AsInt64OK/From Double", Value.AsInt64OK,
			Value{Type: TypeDouble, Data: AppendDouble(nil, 123456.9)},
			nil,
			[]any{int64(123456), true},
		},
		{
			"AsInt64OK/From Int32", Value.AsInt64OK,
			Value{Type: TypeInt32, Data: AppendInt32(nil, 12345)},
			nil,
			[]any{int64(12345), true},
		},
		{
			"AsInt64OK/From Int64", Value.AsInt64OK,
			Value{Type: TypeInt64, Data: AppendInt64(nil, 9876543210)},
			nil,
			[]any{int64(9876543210), true},
		},
		{
			"AsFloat64/Not Number", Value.AsFloat64,
			Value{Type: TypeString},
			ElementTypeError{"bsoncore.Value.AsFloat64", TypeString},
			nil,
		},
		{
			"AsFloat64/Double/Insufficient Bytes", Value.AsFloat64,
			Value{Type: TypeDouble, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"AsFloat64/Int32/Insufficient Bytes", Value.AsFloat64,
			Value{Type: TypeInt32, Data: []byte{0x01, 0x02}},
			NewInsufficientBytesError([]byte{0x01, 0x02}, []byte{0x01, 0x02}),
			nil,
		},
		{
			"AsFloat64/Int64/Insufficient Bytes", Value.AsFloat64,
			Value{Type: TypeInt64, Data: []byte{0x01, 0x02, 0x03}},
			NewInsufficientBytesError([]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}),
			nil,
		},
		{
			"AsFloat64/Decimal128", Value.AsFloat64,
			Value{Type: TypeDecimal128, Data: AppendDecimal128(nil, 12345, 67890)},
			ElementTypeError{"bsoncore.Value.AsFloat64", TypeDecimal128},
			nil,
		},
		{
			"AsFloat64/From Double", Value.AsFloat64,
			Value{Type: TypeDouble, Data: AppendDouble(nil, 3.14159)},
			nil,
			[]any{float64(3.14159)},
		},
		{
			"AsFloat64/From Int32", Value.AsFloat64,
			Value{Type: TypeInt32, Data: AppendInt32(nil, 42)},
			nil,
			[]any{float64(42)},
		},
		{
			"AsFloat64/From Int64", Value.AsFloat64,
			Value{Type: TypeInt64, Data: AppendInt64(nil, 1234567890)},
			nil,
			[]any{float64(1234567890)},
		},
		{
			"AsFloat64OK/Not Number", Value.AsFloat64OK,
			Value{Type: TypeString},
			nil,
			[]any{float64(0), false},
		},
		{
			"AsFloat64OK/Double/Insufficient Bytes", Value.AsFloat64OK,
			Value{Type: TypeDouble, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{float64(0), false},
		},
		{
			"AsFloat64OK/Int32/Insufficient Bytes", Value.AsFloat64OK,
			Value{Type: TypeInt32, Data: []byte{0x01, 0x02}},
			nil,
			[]any{float64(0), false},
		},
		{
			"AsFloat64OK/Int64/Insufficient Bytes", Value.AsFloat64OK,
			Value{Type: TypeInt64, Data: []byte{0x01, 0x02, 0x03}},
			nil,
			[]any{float64(0), false},
		},
		{
			"AsFloat64OK/Decimal128", Value.AsFloat64OK,
			Value{Type: TypeDecimal128, Data: AppendDecimal128(nil, 12345, 67890)},
			nil,
			[]any{float64(0), false},
		},
		{
			"AsFloat64OK/From Double", Value.AsFloat64OK,
			Value{Type: TypeDouble, Data: AppendDouble(nil, 3.14159)},
			nil,
			[]any{float64(3.14159), true},
		},
		{
			"AsFloat64OK/From Int32", Value.AsFloat64OK,
			Value{Type: TypeInt32, Data: AppendInt32(nil, 42)},
			nil,
			[]any{float64(42), true},
		},
		{
			"AsFloat64OK/From Int64", Value.AsFloat64OK,
			Value{Type: TypeInt64, Data: AppendInt64(nil, 1234567890)},
			nil,
			[]any{float64(1234567890), true},
		},
		{
			"Timestamp.String/Success", Value.String,
			Value{Type: TypeTimestamp, Data: AppendTimestamp(nil, 12345, 67890)},
			nil,
			[]any{"{\"$timestamp\":{\"t\":12345,\"i\":67890}}"},
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

var valueStringTestCases = []struct {
	description string
	val         Value
	want        string
}{
	{
		description: "string value",
		val: Value{
			Type: TypeString, Data: AppendString(nil, "abcdefgh"),
		},
		want: `"abcdefgh"`,
	},

	{
		description: "value with special characters",
		val: Value{
			Type: TypeString,
			Data: AppendString(nil, "!@#$%^&*()"),
		},
		want: `"!@#$%^\u0026*()"`,
	},

	{
		description: "TypeEmbeddedDocument",
		val: Value{
			Type: TypeEmbeddedDocument,
			Data: BuildDocument(nil,
				AppendInt32Element(nil, "number", 123),
			),
		},
		want: `{"number": {"$numberInt":"123"}}`,
	},

	{
		description: "TypeArray",
		val: Value{
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
			),
		},
		want: `["abc",{"$numberInt":"123"},true]`,
	},

	{
		description: "TypeDouble",
		val: Value{
			Type: TypeDouble,
			Data: AppendDouble(nil, 123.456),
		},
		want: `{"$numberDouble":"123.456"}`,
	},

	{
		description: "TypeBinary",
		val: Value{
			Type: TypeBinary,
			Data: AppendBinary(nil, 0x00, []byte{0x01, 0x02, 0x03}),
		},
		want: `{"$binary":{"base64":"AQID","subType":"00"}}`,
	},

	{
		description: "TypeUndefined",
		val: Value{
			Type: TypeUndefined,
		},
		want: `{"$undefined":true}`,
	},

	{
		description: "TypeObjectID",
		val: Value{
			Type: TypeObjectID,
			Data: AppendObjectID(nil, [12]byte{0x60, 0xd4, 0xc2, 0x1f, 0x4e, 0x60, 0x4a, 0x0c, 0x8b, 0x2e, 0x9c, 0x3f}),
		},
		want: `{"$oid":"60d4c21f4e604a0c8b2e9c3f"}`,
	},

	{
		description: "TypeBoolean",
		val: Value{
			Type: TypeBoolean,
			Data: AppendBoolean(nil, true),
		},
		want: `true`,
	},

	{
		description: "TypeDateTime",
		val: Value{
			Type: TypeDateTime,
			Data: AppendDateTime(nil, 1234567890),
		},
		want: `{"$date":{"$numberLong":"1234567890"}}`,
	},

	{
		description: "TypeNull",
		val: Value{
			Type: TypeNull,
		},
		want: `null`,
	},

	{
		description: "TypeRegex",
		val: Value{
			Type: TypeRegex,
			Data: AppendRegex(nil, "pattern", "i"),
		},
		want: `{"$regularExpression":{"pattern":"pattern","options":"i"}}`,
	},

	{
		description: "TypeDBPointer",
		val: Value{
			Type: TypeDBPointer,
			Data: AppendDBPointer(nil, "namespace", [12]byte{0x60, 0xd4, 0xc2, 0x1f, 0x4e, 0x60, 0x4a, 0x0c, 0x8b, 0x2e, 0x9c, 0x3f}),
		},
		want: `{"$dbPointer":{"$ref":"namespace","$id":{"$oid":"60d4c21f4e604a0c8b2e9c3f"}}}`,
	},

	{
		description: "TypeJavaScript",
		val: Value{
			Type: TypeJavaScript,
			Data: AppendJavaScript(nil, "code"),
		},
		want: `{"$code":"code"}`,
	},

	{
		description: "TypeSymbol",
		val: Value{
			Type: TypeSymbol,
			Data: AppendSymbol(nil, "symbol"),
		},
		want: `{"$symbol":"symbol"}`,
	},

	{
		description: "TypeCodeWithScope",
		val: Value{
			Type: TypeCodeWithScope,
			Data: AppendCodeWithScope(nil, "code",
				BuildDocument(nil, AppendStringElement(nil, "key", "value")),
			),
		},
		want: `{"$code":code,"$scope":{"key": "value"}}`,
	},

	{
		description: "TypeInt32",
		val: Value{
			Type: TypeInt32,
			Data: AppendInt32(nil, 123),
		},
		want: `{"$numberInt":"123"}`,
	},

	{
		description: "TypeTimestamp",
		val: Value{
			Type: TypeTimestamp,
			Data: AppendTimestamp(nil, 123, 456),
		},
		want: `{"$timestamp":{"t":123,"i":456}}`,
	},

	{
		description: "TypeInt64",
		val: Value{
			Type: TypeInt64,
			Data: AppendInt64(nil, 1234567890),
		},
		want: `{"$numberLong":"1234567890"}`,
	},

	{
		description: "TypeDecimal128",
		val: Value{
			Type: TypeDecimal128,
			Data: AppendDecimal128(nil, 0x3040000000000000, 0x0000000000000000),
		},
		want: `{"$numberDecimal":"0"}`,
	},

	{
		description: "TypeMinKey",
		val: Value{
			Type: TypeMinKey,
		},
		want: `{"$minKey":1}`,
	},

	{
		description: "TypeMaxKey",
		val: Value{
			Type: TypeMaxKey,
		},
		want: `{"$maxKey":1}`,
	},
}

func TestValue_String(t *testing.T) {
	for _, tc := range valueStringTestCases {
		t.Run(tc.description, func(t *testing.T) {
			got := tc.val.String()
			assert.Equal(t, tc.want, got, "expected string %s, got %s", tc.want, got)
		})
	}
}

func TestValue_StringN(t *testing.T) {
	for _, tc := range valueStringTestCases {
		for n := -1; n <= len(tc.want)+1; n++ {
			t.Run(fmt.Sprintf("%s n==%d", tc.description, n), func(t *testing.T) {
				got, truncated := tc.val.StringN(n)
				l := n
				toBeTruncated := true
				if l >= len(tc.want) || l < 0 {
					l = len(tc.want)
					toBeTruncated = false
				}
				want := tc.want[:l]
				assert.Equal(t, want, got, "expected string %s, got %s", tc.want, got)
				assert.Equal(t, toBeTruncated, truncated, "expected truncated to be %t, got %t", toBeTruncated, truncated)
			})
		}
	}
}

func TestArray_StringN_Multibyte(t *testing.T) {
	multiByteString := Value{
		Type: TypeString,
		Data: AppendString(nil, "𨉟呐㗂越"),
	}
	for i, tc := range []struct {
		n         int
		want      string
		truncated bool
	}{
		{6, `"𨉟`, true},
		{8, `"𨉟`, true},
		{10, `"𨉟呐`, true},
		{15, `"𨉟呐㗂越"`, false},
		{21, `"𨉟呐㗂越"`, false},
	} {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			got, truncated := multiByteString.StringN(tc.n)
			assert.Equal(t, tc.want, got, "expected string %s, got %s", tc.want, got)
			assert.Equal(t, tc.truncated, truncated, "expected truncated to be %t, got %t", tc.truncated, truncated)
		})
	}
}
