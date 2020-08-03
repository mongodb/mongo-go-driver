// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"strconv"

	"go.mongodb.org/mongo-driver/bson/bsontype"
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
	key := strconv.Itoa(a.key[last])
	a.arr = AppendDoubleElement(a.arr, key, f)
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
