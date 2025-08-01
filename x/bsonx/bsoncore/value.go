// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/internal/bsoncoreutil"
	"go.mongodb.org/mongo-driver/v2/internal/decimal128"
)

// ElementTypeError specifies that a method to obtain a BSON value an incorrect type was called on a bson.Value.
type ElementTypeError struct {
	Method string
	Type   Type
}

// Error implements the error interface.
func (ete ElementTypeError) Error() string {
	return "Call of " + ete.Method + " on " + ete.Type.String() + " type"
}

// Value represents a BSON value with a type and raw bytes.
type Value struct {
	Type Type
	Data []byte
}

// Validate ensures the value is a valid BSON value.
func (v Value) Validate() error {
	_, _, valid := readValue(v.Data, v.Type)
	if !valid {
		return NewInsufficientBytesError(v.Data, v.Data)
	}
	return nil
}

// IsNumber returns true if the type of v is a numeric BSON type.
func (v Value) IsNumber() bool {
	switch v.Type {
	case TypeDouble, TypeInt32, TypeInt64, TypeDecimal128:
		return true
	default:
		return false
	}
}

// AsInt32 returns a BSON number as an int32. If the BSON type is not a numeric one, this method
// will panic.
func (v Value) AsInt32() int32 {
	if !v.IsNumber() {
		panic(ElementTypeError{"bsoncore.Value.AsInt32", v.Type})
	}
	var i32 int32
	switch v.Type {
	case TypeDouble:
		f64, _, ok := ReadDouble(v.Data)
		if !ok {
			panic(NewInsufficientBytesError(v.Data, v.Data))
		}
		i32 = int32(f64)
	case TypeInt32:
		var ok bool
		i32, _, ok = ReadInt32(v.Data)
		if !ok {
			panic(NewInsufficientBytesError(v.Data, v.Data))
		}
	case TypeInt64:
		i64, _, ok := ReadInt64(v.Data)
		if !ok {
			panic(NewInsufficientBytesError(v.Data, v.Data))
		}
		i32 = int32(i64)
	case TypeDecimal128:
		panic(ElementTypeError{"bsoncore.Value.AsInt32", v.Type})
	}
	return i32
}

// AsInt32OK functions the same as AsInt32 but returns a boolean instead of panicking. False
// indicates an error.
func (v Value) AsInt32OK() (int32, bool) {
	if !v.IsNumber() {
		return 0, false
	}
	var i32 int32
	switch v.Type {
	case TypeDouble:
		f64, _, ok := ReadDouble(v.Data)
		if !ok {
			return 0, false
		}
		i32 = int32(f64)
	case TypeInt32:
		var ok bool
		i32, _, ok = ReadInt32(v.Data)
		if !ok {
			return 0, false
		}
	case TypeInt64:
		i64, _, ok := ReadInt64(v.Data)
		if !ok {
			return 0, false
		}
		i32 = int32(i64)
	case TypeDecimal128:
		return 0, false
	}
	return i32, true
}

// AsInt64 returns a BSON number as an int64. If the BSON type is not a numeric one, this method
// will panic.
func (v Value) AsInt64() int64 {
	if !v.IsNumber() {
		panic(ElementTypeError{"bsoncore.Value.AsInt64", v.Type})
	}
	var i64 int64
	switch v.Type {
	case TypeDouble:
		f64, _, ok := ReadDouble(v.Data)
		if !ok {
			panic(NewInsufficientBytesError(v.Data, v.Data))
		}
		i64 = int64(f64)
	case TypeInt32:
		var ok bool
		i32, _, ok := ReadInt32(v.Data)
		if !ok {
			panic(NewInsufficientBytesError(v.Data, v.Data))
		}
		i64 = int64(i32)
	case TypeInt64:
		var ok bool
		i64, _, ok = ReadInt64(v.Data)
		if !ok {
			panic(NewInsufficientBytesError(v.Data, v.Data))
		}
	case TypeDecimal128:
		panic(ElementTypeError{"bsoncore.Value.AsInt64", v.Type})
	}
	return i64
}

// AsInt64OK functions the same as AsInt64 but returns a boolean instead of panicking. False
// indicates an error.
func (v Value) AsInt64OK() (int64, bool) {
	if !v.IsNumber() {
		return 0, false
	}
	var i64 int64
	switch v.Type {
	case TypeDouble:
		f64, _, ok := ReadDouble(v.Data)
		if !ok {
			return 0, false
		}
		i64 = int64(f64)
	case TypeInt32:
		var ok bool
		i32, _, ok := ReadInt32(v.Data)
		if !ok {
			return 0, false
		}
		i64 = int64(i32)
	case TypeInt64:
		var ok bool
		i64, _, ok = ReadInt64(v.Data)
		if !ok {
			return 0, false
		}
	case TypeDecimal128:
		return 0, false
	}
	return i64, true
}

// AsFloat64 returns a BSON number as an float64. If the BSON type is not a numeric one, this method
// will panic.
//
// TODO(GODRIVER-2751): Implement AsFloat64.
// func (v Value) AsFloat64() float64

// AsFloat64OK functions the same as AsFloat64 but returns a boolean instead of panicking. False
// indicates an error.
//
// TODO(GODRIVER-2751): Implement AsFloat64OK.
// func (v Value) AsFloat64OK() (float64, bool)

// Equal compaes v to v2 and returns true if they are equal.
func (v Value) Equal(v2 Value) bool {
	if v.Type != v2.Type {
		return false
	}

	return bytes.Equal(v.Data, v2.Data)
}

func idHex(id [12]byte) string {
	var buf [24]byte
	hex.Encode(buf[:], id[:])
	return string(buf[:])
}

// String implements the fmt.String interface. This method will return values in extended JSON
// format. If the value is not valid, this returns an empty string
func (v Value) String() string {
	str, _ := v.StringN(-1)
	return str
}

// StringN will return values in extended JSON format that will stringify a value upto N bytes.
// If N is non-negative, it will truncate the string to N bytes. Otherwise, it will return the full
// string representation. The second return value indicates whether the string was truncated or not.
// If the value is not valid, this returns an empty string
func (v Value) StringN(n int) (string, bool) {
	var str string
	switch v.Type {
	case TypeString:
		s, ok := v.StringValueOK()
		if !ok {
			return "", false
		}
		str = escapeString(s)
	case TypeEmbeddedDocument:
		doc, ok := v.DocumentOK()
		if !ok {
			return "", false
		}
		return doc.StringN(n)
	case TypeArray:
		arr, ok := v.ArrayOK()
		if !ok {
			return "", false
		}
		return arr.StringN(n)
	case TypeDouble:
		f64, ok := v.DoubleOK()
		if !ok {
			return "", false
		}
		str = fmt.Sprintf(`{"$numberDouble":"%s"}`, formatDouble(f64))
	case TypeBinary:
		subtype, data, ok := v.BinaryOK()
		if !ok {
			return "", false
		}
		str = fmt.Sprintf(`{"$binary":{"base64":"%s","subType":"%02x"}}`, base64.StdEncoding.EncodeToString(data), subtype)
	case TypeUndefined:
		str = `{"$undefined":true}`
	case TypeObjectID:
		oid, ok := v.ObjectIDOK()
		if !ok {
			return "", false
		}
		str = fmt.Sprintf(`{"$oid":"%s"}`, idHex(oid))
	case TypeBoolean:
		b, ok := v.BooleanOK()
		if !ok {
			return "", false
		}
		str = strconv.FormatBool(b)
	case TypeDateTime:
		dt, ok := v.DateTimeOK()
		if !ok {
			return "", false
		}
		str = fmt.Sprintf(`{"$date":{"$numberLong":"%d"}}`, dt)
	case TypeNull:
		str = "null"
	case TypeRegex:
		pattern, options, ok := v.RegexOK()
		if !ok {
			return "", false
		}
		str = fmt.Sprintf(
			`{"$regularExpression":{"pattern":%s,"options":"%s"}}`,
			escapeString(pattern), sortStringAlphebeticAscending(options),
		)
	case TypeDBPointer:
		ns, pointer, ok := v.DBPointerOK()
		if !ok {
			return "", false
		}
		str = fmt.Sprintf(`{"$dbPointer":{"$ref":%s,"$id":{"$oid":"%s"}}}`, escapeString(ns), idHex(pointer))
	case TypeJavaScript:
		js, ok := v.JavaScriptOK()
		if !ok {
			return "", false
		}
		str = fmt.Sprintf(`{"$code":%s}`, escapeString(js))
	case TypeSymbol:
		symbol, ok := v.SymbolOK()
		if !ok {
			return "", false
		}
		str = fmt.Sprintf(`{"$symbol":%s}`, escapeString(symbol))
	case TypeCodeWithScope:
		code, scope, ok := v.CodeWithScopeOK()
		if !ok {
			return "", false
		}
		str = fmt.Sprintf(`{"$code":%s,"$scope":%s}`, code, scope)
	case TypeInt32:
		i32, ok := v.Int32OK()
		if !ok {
			return "", false
		}
		str = fmt.Sprintf(`{"$numberInt":"%d"}`, i32)
	case TypeTimestamp:
		t, i, ok := v.TimestampOK()
		if !ok {
			return "", false
		}
		str = fmt.Sprintf(`{"$timestamp":{"t":%v,"i":%v}}`, t, i)
	case TypeInt64:
		i64, ok := v.Int64OK()
		if !ok {
			return "", false
		}
		str = fmt.Sprintf(`{"$numberLong":"%d"}`, i64)
	case TypeDecimal128:
		h, l, ok := v.Decimal128OK()
		if !ok {
			return "", false
		}
		str = fmt.Sprintf(`{"$numberDecimal":"%s"}`, decimal128.String(h, l))
	case TypeMinKey:
		str = `{"$minKey":1}`
	case TypeMaxKey:
		str = `{"$maxKey":1}`
	default:
		str = ""
	}
	if n >= 0 && len(str) > n {
		return bsoncoreutil.Truncate(str, n), true
	}
	return str, false
}

// DebugString outputs a human readable version of Document. It will attempt to stringify the
// valid components of the document even if the entire document is not valid.
func (v Value) DebugString() string {
	switch v.Type {
	case TypeString:
		str, ok := v.StringValueOK()
		if !ok {
			return "<malformed>"
		}
		return escapeString(str)
	case TypeEmbeddedDocument:
		doc, ok := v.DocumentOK()
		if !ok {
			return "<malformed>"
		}
		return doc.DebugString()
	case TypeArray:
		arr, ok := v.ArrayOK()
		if !ok {
			return "<malformed>"
		}
		return arr.DebugString()
	case TypeCodeWithScope:
		code, scope, ok := v.CodeWithScopeOK()
		if !ok {
			return ""
		}
		return fmt.Sprintf(`{"$code":%s,"$scope":%s}`, code, scope.DebugString())
	default:
		str := v.String()
		if str == "" {
			return "<malformed>"
		}
		return str
	}
}

// Double returns the float64 value for this element.
// It panics if e's BSON type is not TypeDouble.
func (v Value) Double() float64 {
	if v.Type != TypeDouble {
		panic(ElementTypeError{"bsoncore.Value.Double", v.Type})
	}
	f64, _, ok := ReadDouble(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return f64
}

// DoubleOK is the same as Double, but returns a boolean instead of panicking.
func (v Value) DoubleOK() (float64, bool) {
	if v.Type != TypeDouble {
		return 0, false
	}
	f64, _, ok := ReadDouble(v.Data)
	if !ok {
		return 0, false
	}
	return f64, true
}

// StringValue returns the string balue for this element.
// It panics if e's BSON type is not TypeString.
//
// NOTE: This method is called StringValue to avoid a collision with the String method which
// implements the fmt.Stringer interface.
func (v Value) StringValue() string {
	if v.Type != TypeString {
		panic(ElementTypeError{"bsoncore.Value.StringValue", v.Type})
	}
	str, _, ok := ReadString(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return str
}

// StringValueOK is the same as StringValue, but returns a boolean instead of
// panicking.
func (v Value) StringValueOK() (string, bool) {
	if v.Type != TypeString {
		return "", false
	}
	str, _, ok := ReadString(v.Data)
	if !ok {
		return "", false
	}
	return str, true
}

// Document returns the BSON document the Value represents as a Document. It panics if the
// value is a BSON type other than document.
func (v Value) Document() Document {
	if v.Type != TypeEmbeddedDocument {
		panic(ElementTypeError{"bsoncore.Value.Document", v.Type})
	}
	doc, _, ok := ReadDocument(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return doc
}

// DocumentOK is the same as Document, except it returns a boolean
// instead of panicking.
func (v Value) DocumentOK() (Document, bool) {
	if v.Type != TypeEmbeddedDocument {
		return nil, false
	}
	doc, _, ok := ReadDocument(v.Data)
	if !ok {
		return nil, false
	}
	return doc, true
}

// Array returns the BSON array the Value represents as an Array. It panics if the
// value is a BSON type other than array.
func (v Value) Array() Array {
	if v.Type != TypeArray {
		panic(ElementTypeError{"bsoncore.Value.Array", v.Type})
	}
	arr, _, ok := ReadArray(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return arr
}

// ArrayOK is the same as Array, except it returns a boolean instead
// of panicking.
func (v Value) ArrayOK() (Array, bool) {
	if v.Type != TypeArray {
		return nil, false
	}
	arr, _, ok := ReadArray(v.Data)
	if !ok {
		return nil, false
	}
	return arr, true
}

// Binary returns the BSON binary value the Value represents. It panics if the value is a BSON type
// other than binary.
func (v Value) Binary() (subtype byte, data []byte) {
	if v.Type != TypeBinary {
		panic(ElementTypeError{"bsoncore.Value.Binary", v.Type})
	}
	subtype, data, _, ok := ReadBinary(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return subtype, data
}

// BinaryOK is the same as Binary, except it returns a boolean instead of
// panicking.
func (v Value) BinaryOK() (subtype byte, data []byte, ok bool) {
	if v.Type != TypeBinary {
		return 0x00, nil, false
	}
	subtype, data, _, ok = ReadBinary(v.Data)
	if !ok {
		return 0x00, nil, false
	}
	return subtype, data, true
}

// ObjectID returns the BSON objectid value the Value represents. It panics if the value is a BSON
// type other than objectid.
func (v Value) ObjectID() objectID {
	if v.Type != TypeObjectID {
		panic(ElementTypeError{"bsoncore.Value.ObjectID", v.Type})
	}
	oid, _, ok := ReadObjectID(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return oid
}

// ObjectIDOK is the same as ObjectID, except it returns a boolean instead of
// panicking.
func (v Value) ObjectIDOK() (objectID, bool) {
	if v.Type != TypeObjectID {
		return objectID{}, false
	}
	oid, _, ok := ReadObjectID(v.Data)
	if !ok {
		return objectID{}, false
	}
	return oid, true
}

// Boolean returns the boolean value the Value represents. It panics if the
// value is a BSON type other than boolean.
func (v Value) Boolean() bool {
	if v.Type != TypeBoolean {
		panic(ElementTypeError{"bsoncore.Value.Boolean", v.Type})
	}
	b, _, ok := ReadBoolean(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return b
}

// BooleanOK is the same as Boolean, except it returns a boolean instead of
// panicking.
func (v Value) BooleanOK() (bool, bool) {
	if v.Type != TypeBoolean {
		return false, false
	}
	b, _, ok := ReadBoolean(v.Data)
	if !ok {
		return false, false
	}
	return b, true
}

// DateTime returns the BSON datetime value the Value represents as a
// unix timestamp. It panics if the value is a BSON type other than datetime.
func (v Value) DateTime() int64 {
	if v.Type != TypeDateTime {
		panic(ElementTypeError{"bsoncore.Value.DateTime", v.Type})
	}
	dt, _, ok := ReadDateTime(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return dt
}

// DateTimeOK is the same as DateTime, except it returns a boolean instead of
// panicking.
func (v Value) DateTimeOK() (int64, bool) {
	if v.Type != TypeDateTime {
		return 0, false
	}
	dt, _, ok := ReadDateTime(v.Data)
	if !ok {
		return 0, false
	}
	return dt, true
}

// Time returns the BSON datetime value the Value represents. It panics if the value is a BSON
// type other than datetime.
func (v Value) Time() time.Time {
	if v.Type != TypeDateTime {
		panic(ElementTypeError{"bsoncore.Value.Time", v.Type})
	}
	dt, _, ok := ReadDateTime(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return time.Unix(dt/1000, dt%1000*1000000)
}

// TimeOK is the same as Time, except it returns a boolean instead of
// panicking.
func (v Value) TimeOK() (time.Time, bool) {
	if v.Type != TypeDateTime {
		return time.Time{}, false
	}
	dt, _, ok := ReadDateTime(v.Data)
	if !ok {
		return time.Time{}, false
	}
	return time.Unix(dt/1000, dt%1000*1000000), true
}

// Regex returns the BSON regex value the Value represents. It panics if the value is a BSON
// type other than regex.
func (v Value) Regex() (pattern, options string) {
	if v.Type != TypeRegex {
		panic(ElementTypeError{"bsoncore.Value.Regex", v.Type})
	}
	pattern, options, _, ok := ReadRegex(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return pattern, options
}

// RegexOK is the same as Regex, except it returns a boolean instead of
// panicking.
func (v Value) RegexOK() (pattern, options string, ok bool) {
	if v.Type != TypeRegex {
		return "", "", false
	}
	pattern, options, _, ok = ReadRegex(v.Data)
	if !ok {
		return "", "", false
	}
	return pattern, options, true
}

// DBPointer returns the BSON dbpointer value the Value represents. It panics if the value is a BSON
// type other than DBPointer.
func (v Value) DBPointer() (string, objectID) {
	if v.Type != TypeDBPointer {
		panic(ElementTypeError{"bsoncore.Value.DBPointer", v.Type})
	}
	ns, pointer, _, ok := ReadDBPointer(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return ns, pointer
}

// DBPointerOK is the same as DBPoitner, except that it returns a boolean
// instead of panicking.
func (v Value) DBPointerOK() (string, objectID, bool) {
	if v.Type != TypeDBPointer {
		return "", objectID{}, false
	}
	ns, pointer, _, ok := ReadDBPointer(v.Data)
	if !ok {
		return "", objectID{}, false
	}
	return ns, pointer, true
}

// JavaScript returns the BSON JavaScript code value the Value represents. It panics if the value is
// a BSON type other than JavaScript code.
func (v Value) JavaScript() string {
	if v.Type != TypeJavaScript {
		panic(ElementTypeError{"bsoncore.Value.JavaScript", v.Type})
	}
	js, _, ok := ReadJavaScript(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return js
}

// JavaScriptOK is the same as Javascript, excepti that it returns a boolean
// instead of panicking.
func (v Value) JavaScriptOK() (string, bool) {
	if v.Type != TypeJavaScript {
		return "", false
	}
	js, _, ok := ReadJavaScript(v.Data)
	if !ok {
		return "", false
	}
	return js, true
}

// Symbol returns the BSON symbol value the Value represents. It panics if the value is a BSON
// type other than symbol.
func (v Value) Symbol() string {
	if v.Type != TypeSymbol {
		panic(ElementTypeError{"bsoncore.Value.Symbol", v.Type})
	}
	symbol, _, ok := ReadSymbol(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return symbol
}

// SymbolOK is the same as Symbol, excepti that it returns a boolean
// instead of panicking.
func (v Value) SymbolOK() (string, bool) {
	if v.Type != TypeSymbol {
		return "", false
	}
	symbol, _, ok := ReadSymbol(v.Data)
	if !ok {
		return "", false
	}
	return symbol, true
}

// CodeWithScope returns the BSON JavaScript code with scope the Value represents.
// It panics if the value is a BSON type other than JavaScript code with scope.
func (v Value) CodeWithScope() (string, Document) {
	if v.Type != TypeCodeWithScope {
		panic(ElementTypeError{"bsoncore.Value.CodeWithScope", v.Type})
	}
	code, scope, _, ok := ReadCodeWithScope(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return code, scope
}

// CodeWithScopeOK is the same as CodeWithScope, except that it returns a boolean instead of
// panicking.
func (v Value) CodeWithScopeOK() (string, Document, bool) {
	if v.Type != TypeCodeWithScope {
		return "", nil, false
	}
	code, scope, _, ok := ReadCodeWithScope(v.Data)
	if !ok {
		return "", nil, false
	}
	return code, scope, true
}

// Int32 returns the int32 the Value represents. It panics if the value is a BSON type other than
// int32.
func (v Value) Int32() int32 {
	if v.Type != TypeInt32 {
		panic(ElementTypeError{"bsoncore.Value.Int32", v.Type})
	}
	i32, _, ok := ReadInt32(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return i32
}

// Int32OK is the same as Int32, except that it returns a boolean instead of
// panicking.
func (v Value) Int32OK() (int32, bool) {
	if v.Type != TypeInt32 {
		return 0, false
	}
	i32, _, ok := ReadInt32(v.Data)
	if !ok {
		return 0, false
	}
	return i32, true
}

// Timestamp returns the BSON timestamp value the Value represents. It panics if the value is a
// BSON type other than timestamp.
func (v Value) Timestamp() (t, i uint32) {
	if v.Type != TypeTimestamp {
		panic(ElementTypeError{"bsoncore.Value.Timestamp", v.Type})
	}
	t, i, _, ok := ReadTimestamp(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return t, i
}

// TimestampOK is the same as Timestamp, except that it returns a boolean
// instead of panicking.
func (v Value) TimestampOK() (t, i uint32, ok bool) {
	if v.Type != TypeTimestamp {
		return 0, 0, false
	}
	t, i, _, ok = ReadTimestamp(v.Data)
	if !ok {
		return 0, 0, false
	}
	return t, i, true
}

// Int64 returns the int64 the Value represents. It panics if the value is a BSON type other than
// int64.
func (v Value) Int64() int64 {
	if v.Type != TypeInt64 {
		panic(ElementTypeError{"bsoncore.Value.Int64", v.Type})
	}
	i64, _, ok := ReadInt64(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return i64
}

// Int64OK is the same as Int64, except that it returns a boolean instead of
// panicking.
func (v Value) Int64OK() (int64, bool) {
	if v.Type != TypeInt64 {
		return 0, false
	}
	i64, _, ok := ReadInt64(v.Data)
	if !ok {
		return 0, false
	}
	return i64, true
}

// Decimal128 returns the decimal the Value represents. It panics if the value is a BSON type other than
// decimal.
func (v Value) Decimal128() (uint64, uint64) {
	if v.Type != TypeDecimal128 {
		panic(ElementTypeError{"bsoncore.Value.Decimal128", v.Type})
	}
	h, l, _, ok := ReadDecimal128(v.Data)
	if !ok {
		panic(NewInsufficientBytesError(v.Data, v.Data))
	}
	return h, l
}

// Decimal128OK is the same as Decimal128, except that it returns a boolean
// instead of panicking.
func (v Value) Decimal128OK() (uint64, uint64, bool) {
	if v.Type != TypeDecimal128 {
		return 0, 0, false
	}
	h, l, _, ok := ReadDecimal128(v.Data)
	if !ok {
		return 0, 0, false
	}
	return h, l, true
}

var hexChars = "0123456789abcdef"

func escapeString(s string) string {
	escapeHTML := true
	var buf bytes.Buffer
	buf.WriteByte('"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if htmlSafeSet[b] || (!escapeHTML && safeSet[b]) {
				i++
				continue
			}
			if start < i {
				buf.WriteString(s[start:i])
			}
			switch b {
			case '\\', '"':
				buf.WriteByte('\\')
				buf.WriteByte(b)
			case '\n':
				buf.WriteByte('\\')
				buf.WriteByte('n')
			case '\r':
				buf.WriteByte('\\')
				buf.WriteByte('r')
			case '\t':
				buf.WriteByte('\\')
				buf.WriteByte('t')
			case '\b':
				buf.WriteByte('\\')
				buf.WriteByte('b')
			case '\f':
				buf.WriteByte('\\')
				buf.WriteByte('f')
			default:
				// This encodes bytes < 0x20 except for \t, \n and \r.
				// If escapeHTML is set, it also escapes <, >, and &
				// because they can lead to security holes when
				// user-controlled strings are rendered into JSON
				// and served to some browsers.
				buf.WriteString(`\u00`)
				buf.WriteByte(hexChars[b>>4])
				buf.WriteByte(hexChars[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				buf.WriteString(s[start:i])
			}
			buf.WriteString(`\ufffd`)
			i += size
			start = i
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				buf.WriteString(s[start:i])
			}
			buf.WriteString(`\u202`)
			buf.WriteByte(hexChars[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		buf.WriteString(s[start:])
	}
	buf.WriteByte('"')
	return buf.String()
}

func formatDouble(f float64) string {
	var s string
	switch {
	case math.IsInf(f, 1):
		s = "Infinity"
	case math.IsInf(f, -1):
		s = "-Infinity"
	case math.IsNaN(f):
		s = "NaN"
	default:
		// Print exactly one decimalType place for integers; otherwise, print as many are necessary to
		// perfectly represent it.
		s = strconv.FormatFloat(f, 'G', -1, 64)
		if !strings.ContainsRune(s, '.') {
			s += ".0"
		}
	}

	return s
}

type sortableString []rune

func (ss sortableString) Len() int {
	return len(ss)
}

func (ss sortableString) Less(i, j int) bool {
	return ss[i] < ss[j]
}

func (ss sortableString) Swap(i, j int) {
	ss[i], ss[j] = ss[j], ss[i]
}

func sortStringAlphebeticAscending(s string) string {
	ss := sortableString([]rune(s))
	sort.Sort(ss)
	return string([]rune(ss))
}
