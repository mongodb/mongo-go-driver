// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"reflect"
	"strconv"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ArrayBuilder builds a bson array
type ArrayBuilder struct {
	arr     []byte
	indexes []int32
	keys    []int
}

// NewArrayBuilder creates a new ArrayBuilder
func NewArrayBuilder() *ArrayBuilder {
	return (&ArrayBuilder{}).startArray()
}

// startArray reserves the array's length and sets the index to where the length begins
func (a *ArrayBuilder) startArray() *ArrayBuilder {
	var index int32
	index, a.arr = AppendArrayStart(a.arr)
	a.indexes = append(a.indexes, index)
	a.keys = append(a.keys, 0)
	return a
}

// append element updates a.keys and calls the given function (fn) with params given
func (a *ArrayBuilder) appendElement(fn interface{}, params []interface{}) *ArrayBuilder {
	last := len(a.keys) - 1
	fun := reflect.ValueOf(fn)
	parameters := make([]reflect.Value, 0, len(params))
	for _, param := range params {
		parameters = append(parameters, reflect.ValueOf(param))
	}
	a.arr = fun.Call(parameters)[0].Interface().([]byte)
	a.keys[last]++
	return a
}

// Build updates the length of the array and index to the beginning of the documents length
// bytes, then returns the array (bson bytes)
func (a *ArrayBuilder) Build() (Array, error) {
	lastIndex := len(a.indexes) - 1
	lastKey := len(a.keys) - 1
	var err error
	a.arr, err = AppendArrayEnd(a.arr, a.indexes[lastIndex])
	if err != nil {
		return nil, err
	}
	a.indexes = a.indexes[:lastIndex]
	a.keys = a.keys[:lastKey]
	return a.arr, nil
}

// AppendInt32 will append i32 to ArrayBuilder.arr
func (a *ArrayBuilder) AppendInt32(i32 int32) *ArrayBuilder {
	a = a.appendElement(AppendInt32Element, []interface{}{a.arr, strconv.Itoa(a.keys[len(a.keys)-1]), i32})
	return a
}

// AppendDocument will append doc to ArrayBuilder.arr
func (a *ArrayBuilder) AppendDocument(doc []byte) *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendDocumentElement(a.arr, strconv.Itoa(a.keys[last]), doc)
	a.keys[last]++
	return a
}

// AppendArray will append arr to ArrayBuilder.arr
func (a *ArrayBuilder) AppendArray(arr []byte) *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendHeader(a.arr, bsontype.Array, strconv.Itoa(a.keys[last]))
	a.arr = AppendArray(a.arr, arr)
	a.keys[last]++
	return a
}

// AppendDouble will append f to ArrayBuilder.doc
func (a *ArrayBuilder) AppendDouble(f float64) *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendDoubleElement(a.arr, strconv.Itoa(a.keys[last]), f)
	a.keys[last]++
	return a
}

// AppendString will append str to ArrayBuilder.doc
func (a *ArrayBuilder) AppendString(str string) *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendStringElement(a.arr, strconv.Itoa(a.keys[last]), str)
	a.keys[last]++
	return a
}

// AppendObjectID will append oid to ArrayBuilder.doc
func (a *ArrayBuilder) AppendObjectID(oid primitive.ObjectID) *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendObjectIDElement(a.arr, strconv.Itoa(a.keys[last]), oid)
	a.keys[last]++
	return a
}

// AppendBinary will append a BSON binary element using subtype, and
// b to a.arr
func (a *ArrayBuilder) AppendBinary(subtype byte, b []byte) *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendBinaryElement(a.arr, strconv.Itoa(a.keys[last]), subtype, b)
	a.keys[last]++
	return a
}

// AppendBoolean will append a boolean element using b to a.arr
func (a *ArrayBuilder) AppendBoolean(b bool) *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendBooleanElement(a.arr, strconv.Itoa(a.keys[last]), b)
	a.keys[last]++
	return a
}

// AppendDateTime will append datetime element dt to a.arr
func (a *ArrayBuilder) AppendDateTime(dt int64) *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendDateTimeElement(a.arr, strconv.Itoa(a.keys[last]), dt)
	a.keys[last]++
	return a
}

// AppendNull will append a null element to a.arr
func (a *ArrayBuilder) AppendNull() *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendNullElement(a.arr, strconv.Itoa(a.keys[last]))
	a.keys[last]++
	return a
}

// AppendRegex will append pattern and options to a.arr
func (a *ArrayBuilder) AppendRegex(pattern, options string) *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendRegexElement(a.arr, strconv.Itoa(a.keys[last]), pattern, options)
	a.keys[last]++
	return a
}

// AppendJavaScript will append js to a.arr
func (a *ArrayBuilder) AppendJavaScript(js string) *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendJavaScriptElement(a.arr, strconv.Itoa(a.keys[last]), js)
	a.keys[last]++
	return a
}

// AppendCodeWithScope will append code and scope to a.arr
func (a *ArrayBuilder) AppendCodeWithScope(code Document, scope []byte) *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendCodeWithScopeElement(a.arr, strconv.Itoa(a.keys[last]), string(code), scope)
	a.keys[last]++
	return a
}

// AppendTimestamp will append t and i to a.arr
func (a *ArrayBuilder) AppendTimestamp(t, i uint32) *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendTimestampElement(a.arr, strconv.Itoa(a.keys[last]), t, i)
	a.keys[last]++
	return a
}

// AppendInt64 will append i64 to a.arr
func (a *ArrayBuilder) AppendInt64(i64 int64) *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendInt64Element(a.arr, strconv.Itoa(a.keys[last]), i64)
	a.keys[last]++
	return a
}

// AppendDecimal128 will append d128 to a.arr
func (a *ArrayBuilder) AppendDecimal128(d128 primitive.Decimal128) *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendDecimal128Element(a.arr, strconv.Itoa(a.keys[last]), d128)
	a.keys[last]++
	return a
}

// AppendMaxKey will append a max key element to a.arr
func (a *ArrayBuilder) AppendMaxKey() *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendMaxKeyElement(a.arr, strconv.Itoa(a.keys[last]))
	a.keys[last]++
	return a
}

// AppendMinKey will append a min key element to a.arr
func (a *ArrayBuilder) AppendMinKey() *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendMinKeyElement(a.arr, strconv.Itoa(a.keys[last]))
	a.keys[last]++
	return a
}

// StartArray starts building an inline Array. After this document is completed,
// the user must call a.FinishArray
func (a *ArrayBuilder) StartArray() *ArrayBuilder {
	last := len(a.keys) - 1
	a.arr = AppendHeader(a.arr, bsontype.Array, strconv.Itoa(a.keys[last]))
	a.keys[last]++
	a.startArray()
	return a
}

// FinishArray builds the most recent array created
func (a *ArrayBuilder) FinishArray() *ArrayBuilder {
	a.arr, _ = a.Build()
	return a
}
