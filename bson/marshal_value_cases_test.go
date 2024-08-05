// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"io"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

// helper type for testing MarshalValue that implements io.Reader
type marshalValueInterfaceInner struct {
	Foo int
}

var _ io.Reader = marshalValueInterfaceInner{}

func (marshalValueInterfaceInner) Read([]byte) (int, error) {
	return 0, nil
}

// helper type for testing MarshalValue that contains an interface
type marshalValueInterfaceOuter struct {
	Reader io.Reader
}

// helper type for testing MarshalValue that implements ValueMarshaler
type marshalValueMarshaler struct {
	Foo int
}

var _ ValueMarshaler = marshalValueMarshaler{}

func (mvi marshalValueMarshaler) MarshalBSONValue() (byte, []byte, error) {
	return byte(TypeInt32), bsoncore.AppendInt32(nil, int32(mvi.Foo)), nil
}

var _ ValueUnmarshaler = &marshalValueMarshaler{}

func (mvi *marshalValueMarshaler) UnmarshalBSONValue(_ byte, b []byte) error {
	v, _, _ := bsoncore.ReadInt32(b)
	mvi.Foo = int(v)
	return nil
}

type marshalValueStruct struct {
	Foo int
}

type marshalValueTestCase struct {
	name     string
	val      interface{}
	bsontype Type
	bytes    []byte
}

func newMarshalValueTestCases(t *testing.T) []marshalValueTestCase {
	t.Helper()

	var (
		oid                      = NewObjectID()
		regex                    = Regex{Pattern: "pattern", Options: "imx"}
		dbPointer                = DBPointer{DB: "db", Pointer: NewObjectID()}
		codeWithScope            = CodeWithScope{Code: "code", Scope: D{{"a", "b"}}}
		decimal128h, decimal128l = NewDecimal128(5, 10).GetBytes()
		structTest               = marshalValueStruct{Foo: 10}
	)
	idx, scopeCore := bsoncore.AppendDocumentStart(nil)
	scopeCore = bsoncore.AppendStringElement(scopeCore, "a", "b")
	scopeCore, err := bsoncore.AppendDocumentEnd(scopeCore, idx)
	assert.Nil(t, err, "Document error: %v", err)
	structCore, err := Marshal(structTest)
	assert.Nil(t, err, "Marshal error: %v", err)

	return []marshalValueTestCase{
		{"double", 3.14, TypeDouble, bsoncore.AppendDouble(nil, 3.14)},
		{"string", "hello world", TypeString, bsoncore.AppendString(nil, "hello world")},
		{"binary", Binary{1, []byte{1, 2}}, TypeBinary, bsoncore.AppendBinary(nil, 1, []byte{1, 2})},
		{"undefined", Undefined{}, TypeUndefined, []byte{}},
		{"object id", oid, TypeObjectID, bsoncore.AppendObjectID(nil, oid)},
		{"boolean", true, TypeBoolean, bsoncore.AppendBoolean(nil, true)},
		{"datetime", DateTime(5), TypeDateTime, bsoncore.AppendDateTime(nil, 5)},
		{"null", Null{}, TypeNull, []byte{}},
		{"regex", regex, TypeRegex, bsoncore.AppendRegex(nil, regex.Pattern, regex.Options)},
		{"dbpointer", dbPointer, TypeDBPointer, bsoncore.AppendDBPointer(nil, dbPointer.DB, dbPointer.Pointer)},
		{"javascript", JavaScript("js"), TypeJavaScript, bsoncore.AppendJavaScript(nil, "js")},
		{"symbol", Symbol("symbol"), TypeSymbol, bsoncore.AppendSymbol(nil, "symbol")},
		{"code with scope", codeWithScope, TypeCodeWithScope, bsoncore.AppendCodeWithScope(nil, "code", scopeCore)},
		{"int32", 5, TypeInt32, bsoncore.AppendInt32(nil, 5)},
		{"int64", int64(5), TypeInt64, bsoncore.AppendInt64(nil, 5)},
		{"timestamp", Timestamp{T: 1, I: 5}, TypeTimestamp, bsoncore.AppendTimestamp(nil, 1, 5)},
		{"decimal128", NewDecimal128(decimal128h, decimal128l), TypeDecimal128, bsoncore.AppendDecimal128(nil, decimal128h, decimal128l)},
		{"min key", MinKey{}, TypeMinKey, []byte{}},
		{"max key", MaxKey{}, TypeMaxKey, []byte{}},
		{"struct", structTest, TypeEmbeddedDocument, structCore},
		{"D", D{{"foo", int32(10)}}, TypeEmbeddedDocument, structCore},
		{"M", M{"foo": int32(10)}, TypeEmbeddedDocument, structCore},
		{"ValueMarshaler", marshalValueMarshaler{Foo: 10}, TypeInt32, bsoncore.AppendInt32(nil, 10)},
	}

}

func newMarshalValueTestCasesWithInterfaceCore(t *testing.T) []marshalValueTestCase {
	t.Helper()

	marshalValueTestCases := newMarshalValueTestCases(t)

	interfaceTest := marshalValueInterfaceOuter{
		Reader: marshalValueInterfaceInner{
			Foo: 10,
		},
	}
	interfaceCore, err := Marshal(interfaceTest)
	assert.Nil(t, err, "Marshal error: %v", err)

	marshalValueTestCases = append(
		marshalValueTestCases,
		marshalValueTestCase{"interface", interfaceTest, TypeEmbeddedDocument, interfaceCore},
	)

	return marshalValueTestCases
}
