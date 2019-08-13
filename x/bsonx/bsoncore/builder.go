// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"fmt"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Builder builds each BSON document
type Builder struct {
	bson  []byte
	index int32
	error error
}

// NewBuilder constructs and returns a new Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// AppendType will append t to Builder.bson and return the extended buffer.
func (b *Builder) AppendType(t bsontype.Type) *Builder {
	b.bson = append(b.bson, byte(t))
	return b
}

// AppendKey will append key to Builder.bson and return the extended buffer.
func (b *Builder) AppendKey(key string) *Builder {
	b.bson = append(b.bson, key+string(0x00)...)
	return b
}

// AppendHeader will append Type t and key to Builder.bson and return the extended
// buffer.
func (b *Builder) AppendHeader(t bsontype.Type, key string) *Builder {
	b = b.AppendType(t)
	b.bson = append(b.bson, key...)
	b.bson = append(b.bson, 0x00)
	return b
}

// AppendValueElement appends value to Builder.bson as an element using key as the element's key.
func (b *Builder) AppendValueElement(key string, value Value) *Builder {
	b = b.AppendHeader(value.Type, key)
	b.bson = append(b.bson, value.Data...)
	return b
}

// AppendDouble will append f to Builder.bson and return the extended buffer.
func (b *Builder) AppendDouble(f float64) *Builder {
	b.bson = appendu64(b.bson, math.Float64bits(f))
	return b
}

// AppendDoubleElement will append a BSON double element using key and f to Builder.bson
// and return the extended buffer.
func (b *Builder) AppendDoubleElement(key string, f float64) *Builder {
	b = b.AppendHeader(bsontype.Double, key)
	b = b.AppendDouble(f)
	return b
}

// AppendString will append s to Builder.bson and return the extended buffer.
func (b *Builder) AppendString(s string) *Builder {
	b.bson = appendstring(b.bson, s)
	return b
}

// AppendStringElement will append a BSON string element using key and val to Builder.bson
// and return the extended buffer.
func (b *Builder) AppendStringElement(key, val string) *Builder {
	b = b.AppendHeader(bsontype.String, key)
	b = b.AppendString(val)
	return b
}

// AppendDocumentStart reserves a document's length and returns the index where the length begins.
// This index can later be used to write the length of the document.
//
// TODO(skriptble): We really need AppendDocumentStart and AppendDocumentEnd.
// AppendDocumentStart would handle calling ReserveLength and providing the index of the start of
// the document. AppendDocumentEnd would handle taking that start index, adding the null byte,
// calculating the length, and filling in the length at the start of the document.
func (b *Builder) AppendDocumentStart() *Builder {
	b.index, b.bson = ReserveLength(b.bson)
	return b
}

// AppendDocumentStartInline functions the same as AppendDocumentStart but takes a pointer to the
// index int32 which allows this function to be used inline.
func (b *Builder) AppendDocumentStartInline(index *int32) *Builder {
	b = b.AppendDocumentStart()
	*index = b.index
	return b
}

// AppendDocumentElementStart writes a document element header and then reserves the length bytes.
func (b *Builder) AppendDocumentElementStart(key string) *Builder {
	b = b.AppendHeader(bsontype.EmbeddedDocument, key)
	b = b.AppendDocumentStart()
	return b
}

// AppendDocumentEnd writes the null byte for a document and updates the length of the document.
// The index should be the beginning of the document's length bytes.
func (b *Builder) AppendDocumentEnd(index int32) *Builder {
	if int(index) > len(b.bson)-4 {
		b.error = fmt.Errorf("not enough bytes available after index to write length")
		return b
	}
	b.bson = append(b.bson, 0x00)
	b.bson = UpdateLength(b.bson, index, int32(len(b.bson[index:])))
	return b
}

// AppendDocument will append doc to Builder.bson and return the extended buffer.
func (b *Builder) AppendDocument(doc []byte) *Builder {
	b.bson = append(b.bson, doc...)
	return b
}

// AppendDocumentElement will append a BSON embeded document element using key
// and doc to Builder.bson and return the extended buffer.
func (b *Builder) AppendDocumentElement(key string, doc []byte) *Builder {
	b = b.AppendHeader(bsontype.EmbeddedDocument, key)
	b = b.AppendDocument(doc)
	return b
}

// AppendArrayStart appends the length bytes to an array and then returns the index of the start
// of those length bytes.
func (b *Builder) AppendArrayStart() *Builder {
	b.index, b.bson = ReserveLength(b.bson)
	return b
}

// AppendArrayElementStart appends an array element header and then the length bytes for an array,
// returning the index where the length starts.
func (b *Builder) AppendArrayElementStart(key string) *Builder {
	b = b.AppendHeader(bsontype.Array, key)
	b = b.AppendArrayStart()
	return b
}

// AppendArrayEnd appends the null byte to an array and calculates the length, inserting that
// calculated length starting at index.
func (b *Builder) AppendArrayEnd(index int32) *Builder {
	b = b.AppendDocumentEnd(index)
	return b
}

// AppendArray will append arr to Builder.bson and return the extended buffer.
func (b *Builder) AppendArray(arr []byte) *Builder {
	b.bson = append(b.bson, arr...)
	return b
}

// AppendArrayElement will append a BSON array element using key and arr to Builder.bson
// and return the extended buffer.
func (b *Builder) AppendArrayElement(key string, arr []byte) *Builder {
	b = b.AppendHeader(bsontype.Array, key)
	b = b.AppendArray(arr)
	return b
}

// AppendBinary will append subtype and byte to Builder.bson and return the extended buffer.
func (b *Builder) AppendBinary(subtype byte, byte []byte) *Builder {
	if subtype == 0x02 {
		b.bson = appendBinarySubtype2(b.bson, subtype, byte)
		return b
	}
	b.bson = append(appendLength(b.bson, int32(len(byte))), subtype)
	b.bson = append(b.bson, byte...)
	return b
}

// AppendBinaryElement will append a BSON binary element using key, subtype, and
// byte to Builder.bson and return the extended buffer.
func (b *Builder) AppendBinaryElement(key string, subtype byte, byte []byte) *Builder {
	b = b.AppendHeader(bsontype.Binary, key)
	b = b.AppendBinary(subtype, byte)
	return b
}

// AppendUndefinedElement will append a BSON undefined element using key to Builder.bson
// and return the extended buffer.
func (b *Builder) AppendUndefinedElement(key string) *Builder {
	b = b.AppendHeader(bsontype.Undefined, key)
	return b
}

// AppendObjectID will append oid to Builder.bson and return the extended buffer.
func (b *Builder) AppendObjectID(oid primitive.ObjectID) *Builder {
	b.bson = append(b.bson, oid[:]...)
	return b
}

// AppendObjectIDElement will append a BSON ObjectID element using key and oid to Builder.bson
// and return the extended buffer.
func (b *Builder) AppendObjectIDElement(key string, oid primitive.ObjectID) *Builder {
	b = b.AppendHeader(bsontype.ObjectID, key)
	b = b.AppendObjectID(oid)
	return b
}

// AppendBoolean will append bool to Builder.bson and return the extended buffer.
func (b *Builder) AppendBoolean(bool bool) *Builder {
	if bool {
		b.bson = append(b.bson, 0x01)
		return b
	}
	b.bson = append(b.bson, 0x00)
	return b
}

// AppendBooleanElement will append a BSON boolean element using key and bool to Builder.bson
// and return the extended buffer.
func (b *Builder) AppendBooleanElement(key string, bool bool) *Builder {
	b = b.AppendHeader(bsontype.Boolean, key)
	b = b.AppendBoolean(bool)
	return b
}

// AppendDateTime will append dt to Builder.bson and return the extended buffer.
func (b *Builder) AppendDateTime(dt int64) *Builder {
	b.bson = appendi64(b.bson, dt)
	return b
}

// AppendDateTimeElement will append a BSON datetime element using key and dt to Builder.bson
// and return the extended buffer.
func (b *Builder) AppendDateTimeElement(key string, dt int64) *Builder {
	b = b.AppendHeader(bsontype.DateTime, key)
	b = b.AppendDateTime(dt)
	return b
}

// AppendTime will append time as a BSON DateTime to Builder.bson and return the extended buffer.
func (b *Builder) AppendTime(t time.Time) *Builder {
	b = b.AppendDateTime(t.Unix()*1000 + int64(t.Nanosecond()/1e6))
	return b
}

// AppendTimeElement will append a BSON datetime element using key and dt to Builder.bson
// and return the extended buffer.
func (b *Builder) AppendTimeElement(key string, t time.Time) *Builder {
	b = b.AppendHeader(bsontype.DateTime, key)
	b = b.AppendTime(t)
	return b
}

// AppendNullElement will append a BSON null element using key to Builder.bson
// and return the extended buffer.
func (b *Builder) AppendNullElement(key string) *Builder {
	b = b.AppendHeader(bsontype.Null, key)
	return b
}

// AppendRegex will append pattern and options to Builder.bson and return the extended buffer.
func (b *Builder) AppendRegex(pattern, options string) *Builder {
	b.bson = append(b.bson, pattern+string(0x00)+options+string(0x00)...)
	return b
}

// AppendRegexElement will append a BSON regex element using key, pattern, and
// options to Builder.bson and return the extended buffer.
func (b *Builder) AppendRegexElement(key, pattern, options string) *Builder {
	b = b.AppendHeader(bsontype.Regex, key)
	b = b.AppendRegex(pattern, options)
	return b
}

// AppendDBPointer will append ns and oid to Builder.bson and return the extended buffer.
func (b *Builder) AppendDBPointer(ns string, oid primitive.ObjectID) *Builder {
	b = b.AppendString(ns)
	b.bson = append(b.bson, oid[:]...)
	return b
}

// AppendDBPointerElement will append a BSON DBPointer element using key, ns,
// and oid to Builder.bson and return the extended buffer.
func (b *Builder) AppendDBPointerElement(key, ns string, oid primitive.ObjectID) *Builder {
	b = b.AppendHeader(bsontype.DBPointer, key)
	b = b.AppendDBPointer(ns, oid)
	return b
}

// AppendJavaScript will append js to Builder.bson and return the extended buffer.
func (b *Builder) AppendJavaScript(js string) *Builder {
	b = b.AppendString(js)
	return b
}

// AppendJavaScriptElement will append a BSON JavaScript element using key and
// js to Builder.bson and return the extended buffer.
func (b *Builder) AppendJavaScriptElement(key, js string) *Builder {
	b = b.AppendHeader(bsontype.JavaScript, key)
	b = b.AppendJavaScript(js)
	return b
}

// AppendSymbol will append symbol to Builder.bson and return the extended buffer.
func (b *Builder) AppendSymbol(symbol string) *Builder {
	b.bson = appendstring(b.bson, symbol)
	return b
}

// AppendSymbolElement will append a BSON symbol element using key and symbol to Builder.bson
// and return the extended buffer.
func (b *Builder) AppendSymbolElement(key, symbol string) *Builder {
	b = b.AppendHeader(bsontype.Symbol, key)
	b = b.AppendSymbol(symbol)
	return b
}

// AppendCodeWithScope will append code and scope to Builder.bson and return the extended buffer.
func (b *Builder) AppendCodeWithScope(code string, scope []byte) *Builder {
	length := int32(4 + 4 + len(code) + 1 + len(scope)) // length of cws, length of code, code, 0x00, scope
	b.bson = appendLength(b.bson, length)
	b.bson = append(appendstring(b.bson, code), scope...)
	return b
}

// AppendCodeWithScopeElement will append a BSON code with scope element using
// key, code, and scope to Builder.bson and return the extended buffer.
func (b *Builder) AppendCodeWithScopeElement(key, code string, scope []byte) *Builder {
	b = b.AppendHeader(bsontype.CodeWithScope, key)
	b = b.AppendCodeWithScope(code, scope)
	return b
}

// AppendInt32 will append i32 to Builder.bson and return the extended buffer.
func (b *Builder) AppendInt32(i32 int32) *Builder {
	b.bson = appendi32(b.bson, i32)
	return b
}

// AppendInt32Element will append a BSON int32 element using key and i32 to Builder.bson
// and return the extended buffer.
func (b *Builder) AppendInt32Element(key string, i32 int32) *Builder {
	b = b.AppendHeader(bsontype.Int32, key)
	b = b.AppendInt32(i32)
	return b
}

// AppendTimestamp will append t and i to Builder.bson and return the extended buffer.
func (b *Builder) AppendTimestamp(t, i uint32) *Builder {
	b.bson = appendu32(appendu32(b.bson, i), t) // i is the lower 4 bytes, t is the higher 4 bytes
	return b
}

// AppendTimestampElement will append a BSON timestamp element using key, t, and
// i to Builder.bson and return the extended buffer.
func (b *Builder) AppendTimestampElement(key string, t, i uint32) *Builder {
	b = b.AppendHeader(bsontype.Timestamp, key)
	b = b.AppendTimestamp(t, i)
	return b
}

// AppendInt64 will append i64 to Builder.bson and return the extended buffer.
func (b *Builder) AppendInt64(i64 int64) *Builder {
	b.bson = appendi64(b.bson, i64)
	return b
}

// AppendInt64Element will append a BSON int64 element using key and i64 to Builder.bson
// and return the extended buffer.
func (b *Builder) AppendInt64Element(key string, i64 int64) *Builder {
	b = b.AppendHeader(bsontype.Int64, key)
	b = b.AppendInt64(i64)
	return b
}

// AppendDecimal128 will append d128 to Builder.bson and return the extended buffer.
func (b *Builder) AppendDecimal128(d128 primitive.Decimal128) *Builder {
	high, low := d128.GetBytes()
	b.bson = appendu64(appendu64(b.bson, low), high)
	return b
}

// AppendDecimal128Element will append a BSON primitive.28 element using key and
// d128 to Builder.bson and return the extended buffer.
func (b *Builder) AppendDecimal128Element(key string, d128 primitive.Decimal128) *Builder {
	b = b.AppendHeader(bsontype.Decimal128, key)
	b = b.AppendDecimal128(d128)
	return b
}

// AppendMaxKeyElement will append a BSON max key element using key to Builder.bson
// and return the extended buffer.
func (b *Builder) AppendMaxKeyElement(key string) *Builder {
	b = b.AppendHeader(bsontype.MaxKey, key)
	return b
}

// AppendMinKeyElement will append a BSON min key element using key to Builder.bson
// and return the extended buffer.
func (b *Builder) AppendMinKeyElement(key string) *Builder {
	b = b.AppendHeader(bsontype.MinKey, key)
	return b
}
