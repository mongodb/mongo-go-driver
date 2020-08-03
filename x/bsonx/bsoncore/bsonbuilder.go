// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"strconv"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DocumentBuilder builds a bson document
type DocumentBuilder struct {
	doc     []byte
	indexes []int32
}

// ArrayBuilder builds a bson array
type ArrayBuilder struct {
	arr     []byte
	indexes []int32
	key     []int
}

// startDocument reserves the document's length and set the index to where the length begins
func (db *DocumentBuilder) startDocument() *DocumentBuilder {
	var index int32
	index, db.doc = AppendDocumentStart(db.doc)
	db.indexes = append(db.indexes, index)
	return db
}

// startArray reserves the array's length and sets the index to where the length begins
func (a *ArrayBuilder) startArray() *ArrayBuilder {
	var index int32
	index, a.arr = AppendArrayStart(a.arr)
	a.indexes = append(a.indexes, index)
	a.key = append(a.key, 0)
	return a
}

// NewDocumentBuilder creates a new DocumentBuilder
func NewDocumentBuilder() *DocumentBuilder {
	return (&DocumentBuilder{}).startDocument()
}

// NewArrayBuilder creates a new ArrayBuilder
func NewArrayBuilder() *ArrayBuilder {
	return (&ArrayBuilder{}).startArray()
}

// Build updates the length of the document and index to the beginning of the documents length
// bytes, then returns the document (bson bytes)
func (db *DocumentBuilder) Build() Document {
	last := len(db.indexes) - 1
	db.doc = append(db.doc, 0x00)
	db.doc = UpdateLength(db.doc, db.indexes[last], int32(len(db.doc[db.indexes[last]:])))
	db.indexes = db.indexes[:last]
	return db.doc
}

// Build updates the length of the array and index to the beginning of the documents length
// bytes, then returns the array (bson bytes)
func (a *ArrayBuilder) Build() Array {
	last := len(a.indexes) - 1
	a.arr = append(a.arr, 0x00)
	a.arr = UpdateLength(a.arr, a.indexes[last], int32(len(a.arr[a.indexes[last]:])))
	a.indexes = a.indexes[:last]
	a.key = a.key[:len(a.key)-1]
	return a.arr
}

// AppendInt32 will append an int32 element using key and i32 to DocumentBuilder.doc
func (db *DocumentBuilder) AppendInt32(key string, i32 int32) *DocumentBuilder {
	db.doc = AppendInt32Element(db.doc, key, i32)
	return db
}

// AppendInt32 will append i32 to ArrayBuilder.arr
func (a *ArrayBuilder) AppendInt32(i32 int32) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendInt32Element(a.arr, strconv.Itoa(a.key[last]), i32)
	a.key[last]++
	return a
}

// AppendDocument will append a bson embeded document element using key
// and doc to DocumentBuilder.doc
func (db *DocumentBuilder) AppendDocument(key string, doc []byte) *DocumentBuilder {
	db.doc = AppendDocumentElement(db.doc, key, doc)
	return db
}

// AppendDocument will append doc to ArrayBuilder.arr
func (a *ArrayBuilder) AppendDocument(doc []byte) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendDocumentElement(a.arr, strconv.Itoa(a.key[last]), doc)
	a.key[last]++
	return a
}

// AppendArray will append a bson array using key and arr to DocumentBuilder.doc
func (db *DocumentBuilder) AppendArray(key string, arr []byte) *DocumentBuilder {
	db.doc = AppendHeader(db.doc, bsontype.Array, key)
	db.doc = AppendArray(db.doc, arr)
	return db
}

// AppendArray will append arr to ArrayBuilder.arr
func (a *ArrayBuilder) AppendArray(arr []byte) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendHeader(a.arr, bsontype.Array, strconv.Itoa(a.key[last]))
	a.arr = AppendArray(a.arr, arr)
	a.key[last]++
	return a
}

// AppendDouble will append a double element using key and f to DocumentBuilder.doc
func (db *DocumentBuilder) AppendDouble(key string, f float64) *DocumentBuilder {
	db.doc = AppendDoubleElement(db.doc, key, f)
	return db
}

// AppendDouble will append f to ArrayBuilder.doc
func (a *ArrayBuilder) AppendDouble(f float64) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendDoubleElement(a.arr, strconv.Itoa(a.key[last]), f)
	a.key[last]++
	return a
}

// AppendString will append str to ArrayBuilder.doc with the given key
func (db *DocumentBuilder) AppendString(key string, str string) *DocumentBuilder {
	db.doc = AppendStringElement(db.doc, key, str)
	return db
}

// AppendString will append str to ArrayBuilder.doc
func (a *ArrayBuilder) AppendString(str string) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendStringElement(a.arr, strconv.Itoa(a.key[last]), str)
	a.key[last]++
	return a
}

// AppendObjectID will append oid to DocumentBuilder.doc with the given key
func (db *DocumentBuilder) AppendObjectID(key string, oid primitive.ObjectID) *DocumentBuilder {
	db.doc = AppendObjectIDElement(db.doc, key, oid)
	return db
}

// AppendObjectID will append oid to ArrayBuilder.doc
func (a *ArrayBuilder) AppendObjectID(oid primitive.ObjectID) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendObjectIDElement(a.arr, strconv.Itoa(a.key[last]), oid)
	a.key[last]++
	return a
}

// AppendBinary will append a BSON binary element using key, subtype, and
// b to db.doc
func (db *DocumentBuilder) AppendBinary(key string, subtype byte, b []byte) *DocumentBuilder {
	db.doc = AppendBinaryElement(db.doc, key, subtype, b)
	return db
}

// AppendBinary will append a BSON binary element using subtype, and
// b to a.arr
func (a *ArrayBuilder) AppendBinary(subtype byte, b []byte) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendBinaryElement(a.arr, strconv.Itoa(a.key[last]), subtype, b)
	a.key[last]++
	return a
}

// AppendBoolean will append a boolean element using key and b to db.doc
func (db *DocumentBuilder) AppendBoolean(key string, b bool) *DocumentBuilder {
	db.doc = AppendBooleanElement(db.doc, key, b)
	return db
}

// AppendBoolean will append a boolean element using b to a.arr
func (a *ArrayBuilder) AppendBoolean(b bool) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendBooleanElement(a.arr, strconv.Itoa(a.key[last]), b)
	a.key[last]++
	return a
}

// AppendDateTime will append a datetime element using key and dt to db.doc
func (db *DocumentBuilder) AppendDateTime(key string, dt int64) *DocumentBuilder {
	db.doc = AppendDateTimeElement(db.doc, key, dt)
	return db
}

// AppendDateTime will append datetime element dt to a.arr
func (a *ArrayBuilder) AppendDateTime(dt int64) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendDateTimeElement(a.arr, strconv.Itoa(a.key[last]), dt)
	a.key[last]++
	return a
}

// AppendNull will append a null element using key to db.doc
func (db *DocumentBuilder) AppendNull(key string) *DocumentBuilder {
	db.doc = AppendNullElement(db.doc, key)
	return db
}

// AppendNull will append a null element to a.arr
func (a *ArrayBuilder) AppendNull() *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendNullElement(a.arr, strconv.Itoa(a.key[last]))
	a.key[last]++
	return a
}

// AppendRegex will append pattern and options using key to db.doc
func (db *DocumentBuilder) AppendRegex(key, pattern, options string) *DocumentBuilder {
	db.doc = AppendRegexElement(db.doc, key, pattern, options)
	return db
}

// AppendRegex will append pattern and options to a.arr
func (a *ArrayBuilder) AppendRegex(pattern, options string) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendRegexElement(a.arr, strconv.Itoa(a.key[last]), pattern, options)
	a.key[last]++
	return a
}

// AppendJavaScript will append js using the provided key to db.doc
func (db *DocumentBuilder) AppendJavaScript(key, js string) *DocumentBuilder {
	db.doc = AppendJavaScriptElement(db.doc, key, js)
	return db
}

// AppendJavaScript will append js to a.arr
func (a *ArrayBuilder) AppendJavaScript(js string) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendJavaScriptElement(a.arr, strconv.Itoa(a.key[last]), js)
	a.key[last]++
	return a
}

// AppendCodeWithScope will append code and scope using key to db.doc
func (db *DocumentBuilder) AppendCodeWithScope(key, code string, scope []byte) *DocumentBuilder {
	db.doc = AppendCodeWithScopeElement(db.doc, key, code, scope)
	return db
}

// AppendCodeWithScope will append code and scope to a.arr
func (a *ArrayBuilder) AppendCodeWithScope(code string, scope []byte) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendCodeWithScopeElement(a.arr, strconv.Itoa(a.key[last]), code, scope)
	a.key[last]++
	return a
}

// AppendTimestamp will append t and i to db.doc using provided key
func (db *DocumentBuilder) AppendTimestamp(key string, t, i uint32) *DocumentBuilder {
	db.doc = AppendTimestampElement(db.doc, key, t, i)
	return db
}

// AppendTimestamp will append t and i to a.arr
func (a *ArrayBuilder) AppendTimestamp(t, i uint32) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendTimestampElement(a.arr, strconv.Itoa(a.key[last]), t, i)
	a.key[last]++
	return a
}

// AppendInt64 will append i64 to dst using key to db.doc
func (db *DocumentBuilder) AppendInt64(key string, i64 int64) *DocumentBuilder {
	db.doc = AppendInt64Element(db.doc, key, i64)
	return db
}

// AppendInt64 will append i64 to a.arr
func (a *ArrayBuilder) AppendInt64(i64 int64) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendInt64Element(a.arr, strconv.Itoa(a.key[last]), i64)
	a.key[last]++
	return a
}

// AppendDecimal128 will append d128 to db.doc using provided key
func (db *DocumentBuilder) AppendDecimal128(key string, d128 primitive.Decimal128) *DocumentBuilder {
	db.doc = AppendDecimal128Element(db.doc, key, d128)
	return db
}

// AppendDecimal128 will append d128 to a.arr
func (a *ArrayBuilder) AppendDecimal128(d128 primitive.Decimal128) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendDecimal128Element(a.arr, strconv.Itoa(a.key[last]), d128)
	a.key[last]++
	return a
}

// AppendMaxKey will append a max key element using key to db.doc
func (db *DocumentBuilder) AppendMaxKey(key string) *DocumentBuilder {
	db.doc = AppendMaxKeyElement(db.doc, key)
	return db
}

// AppendMaxKey will append a max key element to a.arr
func (a *ArrayBuilder) AppendMaxKey(key string) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendMaxKeyElement(a.arr, strconv.Itoa(a.key[last]))
	a.key[last]++
	return a
}

// AppendMinKey will append a min key element using key to db.doc
func (db *DocumentBuilder) AppendMinKey(key string) *DocumentBuilder {
	db.doc = AppendMinKeyElement(db.doc, key)
	return db
}

// AppendMinKey will append a min key element to a.arr
func (a *ArrayBuilder) AppendMinKey(key string) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendMinKeyElement(a.arr, strconv.Itoa(a.key[last]))
	a.key[last]++
	return a
}

// AppendInlineDocument starts building a document element with the provided key
// After this document is completed, the user must call buildInlineDocument
func (db *DocumentBuilder) AppendInlineDocument(key string) *DocumentBuilder {
	db.doc = AppendHeader(db.doc, bsontype.EmbeddedDocument, key)
	db = db.startDocument()
	return db
}

// BuildInlineDocument builds the most recent document created
func (db *DocumentBuilder) BuildInlineDocument() *DocumentBuilder {
	db.doc = db.Build()
	return db
}

// AppendInlineArray starts building an Array. After this document is completed,
// the user must call BuildInlineArray
func (a *ArrayBuilder) AppendInlineArray() *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendHeader(a.arr, bsontype.Array, strconv.Itoa(a.key[last]))
	a.key[last]++
	a.startArray()
	return a
}

// BuildInlineArray builds the most recent array created
func (a *ArrayBuilder) BuildInlineArray() *ArrayBuilder {
	a.arr = a.Build()
	return a
}
