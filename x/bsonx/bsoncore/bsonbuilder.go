// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package bsoncore contains functions that can be used to encode and decode bson
// elements and values to or from a slice of bytes. These functions are aimed at
// allowing low level manipulation of bson and can be used to build a higher
// level bson library.
package bsoncore

import (
	"strconv"

	"go.mongodb.org/mongo-driver/bson/bsontype"
)

// DocumentBuilder builds a bson document
type DocumentBuilder struct {
	doc   []byte
	index []int32
}

// ArrayBuilder builds a bson array
type ArrayBuilder struct {
	arr   []byte
	index []int32
	key   []int
}

// startDocument reserves the document's length and set the index to where the length begins
func (b *DocumentBuilder) startDocument() *DocumentBuilder {
	var index int32
	index, b.doc = AppendDocumentStart(b.doc)
	b.index = append(b.index, index)
	return b
}

// startArray reserves the array's length and sets the index to where the length begins
func (a *ArrayBuilder) startArray() *ArrayBuilder {
	var index int32
	index, a.arr = AppendArrayStart(a.arr)
	a.index = append(a.index, index)
	a.key = append(a.key, 0)
	return a
}

// NewDocumentBuilder creates a new DocumentBuilder
func NewDocumentBuilder() *DocumentBuilder {
	b := (&DocumentBuilder{}).startDocument()
	return b
}

// NewArrayBuilder creates a new ArrayBuilder
func NewArrayBuilder() *ArrayBuilder {
	a := (&ArrayBuilder{}).startArray()
	return a
}

// Build updates the length of the document and index to the beginning of the documents length
// bytes, then returns the document (bson bytes)
func (b *DocumentBuilder) Build() []byte {
	last := len(b.index) - 1
	b.doc = append(b.doc, 0x00)
	b.doc = UpdateLength(b.doc, b.index[last], int32(len(b.doc[b.index[last]:])))
	b.index = b.index[:last]
	return b.doc
}

// Build updates the length of the array and index to the beginning of the documents length
// bytes, then returns the array (bson bytes)
func (a *ArrayBuilder) Build() []byte {
	last := len(a.index) - 1
	a.arr = append(a.arr, 0x00)
	a.arr = UpdateLength(a.arr, a.index[last], int32(len(a.arr[a.index[last]:])))
	a.index = a.index[:last]
	a.key = a.key[:len(a.key)-1]
	return a.arr
}

// AppendInt32 will append a bson int32 element using key and i32 to DocumentBuilder.doc
func (b *DocumentBuilder) AppendInt32(key string, i32 int32) *DocumentBuilder {
	b.doc = AppendHeader(b.doc, bsontype.Int32, key)
	b.doc = AppendInt32(b.doc, i32)
	return b
}

// AppendInt32 will append i32 to ArrayBuilder.arr
func (a *ArrayBuilder) AppendInt32(i32 int32) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendHeader(a.arr, bsontype.Int32, strconv.Itoa(a.key[last]))
	a.arr = AppendInt32(a.arr, i32)
	a.key[last]++
	return a
}

// AppendDocument will append a bson embeded document element using key
// and doc to DocumentBuilder.doc
func (b *DocumentBuilder) AppendDocument(key string, doc []byte) *DocumentBuilder {
	b.doc = AppendHeader(b.doc, bsontype.EmbeddedDocument, key)
	b.doc = AppendDocument(b.doc, doc)
	return b
}

// AppendDocument will append doc to ArrayBuilder.arr
func (a *ArrayBuilder) AppendDocument(doc []byte) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendHeader(a.arr, bsontype.EmbeddedDocument, strconv.Itoa(a.key[last]))
	a.arr = AppendDocument(a.arr, doc)
	a.key[last]++
	return a
}

// AppendArray will append a bson array using key and arr to DocumentBuilder.doc
//
func (b *DocumentBuilder) AppendArray(key string, arr []byte) *DocumentBuilder {
	b.doc = AppendHeader(b.doc, bsontype.Array, key)
	b.doc = AppendArray(b.doc, arr)
	return b
}

// AppendArray will append arr to ArrayBuilder.arr
func (a *ArrayBuilder) AppendArray(arr []byte) *ArrayBuilder {
	last := len(a.key) - 1
	a.arr = AppendHeader(a.arr, bsontype.Array, strconv.Itoa(a.key[last]))
	a.arr = AppendArray(a.arr, arr)
	a.key[last]++
	return a
}
