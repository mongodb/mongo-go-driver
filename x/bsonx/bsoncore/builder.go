// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"math"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DocumentBuilder builds each BSON document
type DocumentBuilder struct {
	bson  []byte
	index []int32
}

// ArrayBuilder builds each BSON array
type ArrayBuilder struct {
	bson  []byte
	index []int32
	key   []int
}

// NewDocumentBuilder constructs and returns a new DocumentBuilder.
func NewDocumentBuilder() *DocumentBuilder {
	b := &DocumentBuilder{}
	b = b.AppendDocumentStart()
	return b
}

// NewArrayBuilder constructs and returns a new ArrayBuilder.
func NewArrayBuilder() *ArrayBuilder {
	a := &ArrayBuilder{}
	a = a.AppendArrayStart()
	return a
}

// AppendValueElement appends value to DocumentBuilder.bson as an element using key as the element's key.
func (b *DocumentBuilder) AppendValueElement(key string, value Value) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, value.Type, key)
	b.bson = append(b.bson, value.Data...)
	return b
}

// AppendDoubleElement will append a BSON double element using key and f to DocumentBuilder.bson
// and return the extended buffer.
func (b *DocumentBuilder) AppendDoubleElement(key string, f float64) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.Double, key)
	b.bson = appendu64(b.bson, math.Float64bits(f))
	return b
}

// AppendDouble will append f to ArrayBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendDouble(f float64) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.Double, strconv.Itoa(a.key[last]))
	a.bson = appendu64(a.bson, math.Float64bits(f))
	a.key[last]++
	return a
}

// AppendStringElement will append a BSON string element using key and val to DocumentBuilder.bson
// and return the extended buffer.
func (b *DocumentBuilder) AppendStringElement(key, val string) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.String, key)
	b.bson = appendstring(b.bson, val)
	return b
}

// AppendString will append s to ArrayBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendString(s string) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.String, strconv.Itoa(a.key[last]))
	a.bson = appendstring(a.bson, s)
	a.key[last]++
	return a
}

// AppendDocumentStart reserves a document's length and returns the index where the length begins.
// This index can later be used to write the length of the document.
//
// TODO(skriptble): We really need AppendDocumentStart and AppendDocumentEnd.
// AppendDocumentStart would handle calling ReserveLength and providing the index of the start of
// the document. AppendDocumentEnd would handle taking that start index, adding the null byte,
// calculating the length, and filling in the length at the start of the document.
func (b *DocumentBuilder) AppendDocumentStart() *DocumentBuilder {
	var idx int32
	idx, b.bson = ReserveLength(b.bson)
	b.index = append(b.index, idx)
	return b
}

// AppendDocumentElementStart writes a document element header and then reserves the length bytes.
func (b *DocumentBuilder) AppendDocumentElementStart(key string) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.EmbeddedDocument, key)
	b = b.AppendDocumentStart()
	return b
}

// AppendDocumentEnd writes the null byte for a document and updates the length of the document.
// The index should be the beginning of the document's length bytes.
func (b *DocumentBuilder) AppendDocumentEnd() *DocumentBuilder {
	last := len(b.index) - 1
	b.bson = append(b.bson, 0x00)
	b.bson = UpdateLength(b.bson, b.index[last], int32(len(b.bson[b.index[last]:])))
	b.index = b.index[:last]
	return b
}

// AppendDocumentElement will append a BSON embeded document element using key
// and doc to DocumentBuilder.bson and return the extended buffer.
func (b *DocumentBuilder) AppendDocumentElement(key string, doc []byte) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.EmbeddedDocument, key)
	b.bson = append(b.bson, doc...)
	return b
}

// AppendDocument will append doc to ArrayBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendDocument(doc []byte) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.EmbeddedDocument, strconv.Itoa(a.key[last]))
	a.bson = append(a.bson, doc...)
	a.key[last]++
	return a
}

// AppendArrayStart appends the length bytes to an array and then returns the index of the start
// of those length bytes.
func (a *ArrayBuilder) AppendArrayStart() *ArrayBuilder {
	var idx int32
	idx, a.bson = ReserveLength(a.bson)
	a.index = append(a.index, idx)
	a.key = append(a.key, 0)
	return a
}

// AppendArrayElementStart appends an array element header and then the length bytes for an array,
// returning the index where the length starts.
func (a *ArrayBuilder) AppendArrayElementStart(key string) *ArrayBuilder {
	a.bson = AppendHeader(a.bson, bsontype.Array, key)
	a = a.AppendArrayStart()
	return a
}

// AppendArrayEnd appends the null byte to an array and calculates the length, inserting that
// calculated length starting at index.
func (a *ArrayBuilder) AppendArrayEnd() *ArrayBuilder {
	last := len(a.index) - 1
	a.bson = append(a.bson, 0x00)
	a.bson = UpdateLength(a.bson, a.index[last], int32(len(a.bson[a.index[last]:])))
	a.index = a.index[:last]
	a.key = a.key[:len(a.key)-1]
	return a
}

// AppendArrayElement will append a BSON array element using key and arr to DocumentBuilder.bson
// and return the extended buffer.
func (b *DocumentBuilder) AppendArrayElement(key string, arr []byte) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.Array, key)
	b.bson = append(b.bson, arr...)
	return b
}

// AppendArray will append arr to ArrayBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendArray(arr []byte) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.Array, strconv.Itoa(a.key[last]))
	a.bson = append(a.bson, arr...)
	a.key[last]++
	return a
}

// AppendBinaryElement will append a BSON binary element using key, subtype, and
// byte to DocumentBuilder.bson and return the extended buffer.
func (b *DocumentBuilder) AppendBinaryElement(key string, subtype byte, byte []byte) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.Binary, key)
	if subtype == 0x02 {
		b.bson = appendBinarySubtype2(b.bson, subtype, byte)
		return b
	}
	b.bson = append(appendLength(b.bson, int32(len(byte))), subtype)
	b.bson = append(b.bson, byte...)
	return b
}

// AppendBinary will append subtype and byte to ArrayBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendBinary(subtype byte, byte []byte) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.Binary, strconv.Itoa(a.key[last]))
	a.key[last]++
	if subtype == 0x02 {
		a.bson = appendBinarySubtype2(a.bson, subtype, byte)
		return a
	}
	a.bson = append(appendLength(a.bson, int32(len(byte))), subtype)
	a.bson = append(a.bson, byte...)
	return a
}

// AppendUndefinedElement will append a BSON undefined element using key to DocumentBuilder.bson
// and return the extended buffer.
func (b *DocumentBuilder) AppendUndefinedElement(key string) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.Undefined, key)
	return b
}

// AppendObjectIDElement will append a BSON ObjectID element using key and oid to DocumentBuilder.bson
// and return the extended buffer.
func (b *DocumentBuilder) AppendObjectIDElement(key string, oid primitive.ObjectID) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.ObjectID, key)
	b.bson = append(b.bson, oid[:]...)
	return b
}

// AppendObjectID will append oid to ArrayBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendObjectID(oid primitive.ObjectID) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.ObjectID, strconv.Itoa(a.key[last]))
	a.bson = append(a.bson, oid[:]...)
	a.key[last]++
	return a
}

// AppendBooleanElement will append a BSON boolean element using key and bool to DocumentBuilder.bson
// and return the extended buffer.
func (b *DocumentBuilder) AppendBooleanElement(key string, bool bool) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.Boolean, key)
	if bool {
		b.bson = append(b.bson, 0x01)
		return b
	}
	b.bson = append(b.bson, 0x00)
	return b
}

// AppendBoolean will append bool to DocumentBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendBoolean(bool bool) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.Boolean, strconv.Itoa(a.key[last]))
	a.key[last]++
	if bool {
		a.bson = append(a.bson, 0x01)
		return a
	}
	a.bson = append(a.bson, 0x00)
	return a
}

// AppendDateTimeElement will append a BSON datetime element using key and dt to DocumentBuilder.bson
// and return the extended buffer.
func (b *DocumentBuilder) AppendDateTimeElement(key string, dt int64) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.DateTime, key)
	b.bson = appendi64(b.bson, dt)
	return b
}

// AppendDateTime will append dt to ArrayBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendDateTime(dt int64) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.DateTime, strconv.Itoa(a.key[last]))
	a.bson = appendi64(a.bson, dt)
	a.key[last]++
	return a
}

// AppendTimeElement will append a BSON datetime element using key and dt to DocumentBuilder.bson
// and return the extended buffer.
func (b *DocumentBuilder) AppendTimeElement(key string, t time.Time) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.DateTime, key)
	b.bson = appendi64(b.bson, t.Unix()*1000+int64(t.Nanosecond()/1e6))
	return b
}

// AppendTime will append time as a BSON DateTime to ArrayBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendTime(t time.Time) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.DateTime, strconv.Itoa(a.key[last]))
	a.bson = appendi64(a.bson, t.Unix()*1000+int64(t.Nanosecond()/1e6))
	a.key[last]++
	return a
}

// AppendNullElement will append a BSON null element using key to DocumentBuilder.bson
// and return the extended buffer.
func (b *DocumentBuilder) AppendNullElement(key string) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.Null, key)
	return b
}

// AppendRegexElement will append a BSON regex element using key, pattern, and
// options to DocumentBuilder.bson and return the extended buffer.
func (b *DocumentBuilder) AppendRegexElement(key, pattern, options string) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.Regex, key)
	b.bson = append(b.bson, pattern+string(0x00)+options+string(0x00)...)
	return b
}

// AppendRegex will append pattern and options to ArrayBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendRegex(pattern, options string) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.Regex, strconv.Itoa(a.key[last]))
	a.bson = append(a.bson, pattern+string(0x00)+options+string(0x00)...)
	a.key[last]++
	return a
}

// AppendDBPointerElement will append a BSON DBPointer element using key, ns,
// and oid to DocumentBuilder.bson and return the extended buffer.
func (b *DocumentBuilder) AppendDBPointerElement(key, ns string, oid primitive.ObjectID) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.DBPointer, key)
	b.bson = appendstring(b.bson, ns)
	b.bson = append(b.bson, oid[:]...)
	return b
}

// AppendDBPointer will append ns and oid to ArrayBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendDBPointer(ns string, oid primitive.ObjectID) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.DBPointer, strconv.Itoa(a.key[last]))
	a.bson = appendstring(a.bson, ns)
	a.bson = append(a.bson, oid[:]...)
	a.key[last]++
	return a
}

// AppendJavaScriptElement will append a BSON JavaScript element using key and
// js to DocumentBuilder.bson and return the extended buffer.
func (b *DocumentBuilder) AppendJavaScriptElement(key, js string) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.JavaScript, key)
	b.bson = appendstring(b.bson, js)
	return b
}

// AppendJavaScript will append js to ArrayBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendJavaScript(js string) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.JavaScript, strconv.Itoa(a.key[last]))
	a.bson = appendstring(a.bson, js)
	a.key[last]++
	return a
}

// AppendSymbolElement will append a BSON symbol element using key and symbol to DocumentBuilder.bson
// and return the extended buffer.
func (b *DocumentBuilder) AppendSymbolElement(key, symbol string) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.Symbol, key)
	b.bson = appendstring(b.bson, symbol)
	return b
}

// AppendSymbol will append symbol to ArrayBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendSymbol(symbol string) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.Symbol, strconv.Itoa(a.key[last]))
	a.bson = appendstring(a.bson, symbol)
	a.key[last]++
	return a
}

// AppendCodeWithScopeElement will append a BSON code with scope element using
// key, code, and scope to DocumentBuilder.bson and return the extended buffer.
func (b *DocumentBuilder) AppendCodeWithScopeElement(key, code string, scope []byte) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.CodeWithScope, key)
	length := int32(4 + 4 + len(code) + 1 + len(scope)) // length of cws, length of code, code, 0x00, scope
	b.bson = appendLength(b.bson, length)
	b.bson = append(appendstring(b.bson, code), scope...)
	return b
}

// AppendCodeWithScope will append code and scope to ArrayBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendCodeWithScope(code string, scope []byte) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.CodeWithScope, strconv.Itoa(a.key[last]))
	length := int32(4 + 4 + len(code) + 1 + len(scope)) // length of cws, length of code, code, 0x00, scope
	a.bson = appendLength(a.bson, length)
	a.bson = append(appendstring(a.bson, code), scope...)
	a.key[last]++
	return a
}

// AppendInt32Element will append a BSON int32 element using key and i32 to DocumentBuilder.bson
// and return the extended buffer.
func (b *DocumentBuilder) AppendInt32Element(key string, i32 int32) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.Int32, key)
	b.bson = appendi32(b.bson, i32)
	return b
}

// AppendInt32 will append i32 to ArrayBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendInt32(i32 int32) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.Int32, strconv.Itoa(a.key[last]))
	a.bson = appendi32(a.bson, i32)
	a.key[last]++
	return a
}

// AppendTimestampElement will append a BSON timestamp element using key, t, and
// i to DocumentBuilder.bson and return the extended buffer.
func (b *DocumentBuilder) AppendTimestampElement(key string, t, i uint32) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.Timestamp, key)
	b.bson = appendu32(appendu32(b.bson, i), t) // i is the lower 4 bytes, t is the higher 4 bytes
	return b
}

// AppendTimestamp will append t and i to ArrayBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendTimestamp(t, i uint32) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.Timestamp, strconv.Itoa(a.key[last]))
	a.bson = appendu32(appendu32(a.bson, i), t) // i is the lower 4 bytes, t is the higher 4 bytes
	a.key[last]++
	return a
}

// AppendInt64Element will append a BSON int64 element using key and i64 to DocumentBuilder.bson
// and return the extended buffer.
func (b *DocumentBuilder) AppendInt64Element(key string, i64 int64) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.Int64, key)
	b.bson = appendi64(b.bson, i64)
	return b
}

// AppendInt64 will append i64 to DocumentBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendInt64(i64 int64) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.Int64, strconv.Itoa(a.key[last]))
	a.bson = appendi64(a.bson, i64)
	a.key[last]++
	return a
}

// AppendDecimal128Element will append a BSON primitive.28 element using key and
// d128 to DocumentBuilder.bson and return the extended buffer.
func (b *DocumentBuilder) AppendDecimal128Element(key string, d128 primitive.Decimal128) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.Decimal128, key)
	high, low := d128.GetBytes()
	b.bson = appendu64(appendu64(b.bson, low), high)
	return b
}

// AppendDecimal128 will append d128 to ArrayBuilder.bson and return the extended buffer.
func (a *ArrayBuilder) AppendDecimal128(d128 primitive.Decimal128) *ArrayBuilder {
	last := len(a.key) - 1
	a.bson = AppendHeader(a.bson, bsontype.Int64, strconv.Itoa(a.key[last]))
	high, low := d128.GetBytes()
	a.bson = appendu64(appendu64(a.bson, low), high)
	a.key[last]++
	return a
}

// AppendMaxKeyElement will append a BSON max key element using key to DocumentBuilder.bson
// and return the extended buffer.
func (b *DocumentBuilder) AppendMaxKeyElement(key string) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.MaxKey, key)
	return b
}

// AppendMinKeyElement will append a BSON min key element using key to DocumentBuilder.bson
// and return the extended buffer.
func (b *DocumentBuilder) AppendMinKeyElement(key string) *DocumentBuilder {
	b.bson = AppendHeader(b.bson, bsontype.MinKey, key)
	return b
}

// Build constructs and returns the resulting BSON bytes
func (b *DocumentBuilder) Build() []byte {
	b = b.AppendDocumentEnd()
	return b.bson
}

// Build constructs and returns the resulting BSON bytes
func (a *ArrayBuilder) Build() []byte {
	a = a.AppendArrayEnd()
	return a.bson
}
