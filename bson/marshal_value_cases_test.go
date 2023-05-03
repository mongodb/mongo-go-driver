// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"io"
	"testing"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
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

func (mvi marshalValueMarshaler) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.Int32, bsoncore.AppendInt32(nil, int32(mvi.Foo)), nil
}

var _ ValueUnmarshaler = &marshalValueMarshaler{}

func (mvi *marshalValueMarshaler) UnmarshalBSONValue(_ bsontype.Type, b []byte) error {
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
	bsontype bsontype.Type
	bytes    []byte
}

func newMarshalValueTestCases(t *testing.T) []marshalValueTestCase {
	t.Helper()

	var (
		oid           = primitive.NewObjectID()
		regex         = primitive.Regex{Pattern: "pattern", Options: "imx"}
		dbPointer     = primitive.DBPointer{DB: "db", Pointer: primitive.NewObjectID()}
		codeWithScope = primitive.CodeWithScope{Code: "code", Scope: D{{"a", "b"}}}
		decimal128    = primitive.NewDecimal128(5, 10)
		structTest    = marshalValueStruct{Foo: 10}
	)
	idx, scopeCore := bsoncore.AppendDocumentStart(nil)
	scopeCore = bsoncore.AppendStringElement(scopeCore, "a", "b")
	scopeCore, err := bsoncore.AppendDocumentEnd(scopeCore, idx)
	assert.Nil(t, err, "Document error: %v", err)
	structCore, err := Marshal(structTest)
	assert.Nil(t, err, "Marshal error: %v", err)

	return []marshalValueTestCase{
		{"double", 3.14, bsontype.Double, bsoncore.AppendDouble(nil, 3.14)},
		{"string", "hello world", bsontype.String, bsoncore.AppendString(nil, "hello world")},
		{"binary", primitive.Binary{1, []byte{1, 2}}, bsontype.Binary, bsoncore.AppendBinary(nil, 1, []byte{1, 2})},
		{"undefined", primitive.Undefined{}, bsontype.Undefined, []byte{}},
		{"object id", oid, bsontype.ObjectID, bsoncore.AppendObjectID(nil, oid)},
		{"boolean", true, bsontype.Boolean, bsoncore.AppendBoolean(nil, true)},
		{"datetime", primitive.DateTime(5), bsontype.DateTime, bsoncore.AppendDateTime(nil, 5)},
		{"null", primitive.Null{}, bsontype.Null, []byte{}},
		{"regex", regex, bsontype.Regex, bsoncore.AppendRegex(nil, regex.Pattern, regex.Options)},
		{"dbpointer", dbPointer, bsontype.DBPointer, bsoncore.AppendDBPointer(nil, dbPointer.DB, dbPointer.Pointer)},
		{"javascript", primitive.JavaScript("js"), bsontype.JavaScript, bsoncore.AppendJavaScript(nil, "js")},
		{"symbol", primitive.Symbol("symbol"), bsontype.Symbol, bsoncore.AppendSymbol(nil, "symbol")},
		{"code with scope", codeWithScope, bsontype.CodeWithScope, bsoncore.AppendCodeWithScope(nil, "code", scopeCore)},
		{"int32", 5, bsontype.Int32, bsoncore.AppendInt32(nil, 5)},
		{"int64", int64(5), bsontype.Int64, bsoncore.AppendInt64(nil, 5)},
		{"timestamp", primitive.Timestamp{T: 1, I: 5}, bsontype.Timestamp, bsoncore.AppendTimestamp(nil, 1, 5)},
		{"decimal128", decimal128, bsontype.Decimal128, bsoncore.AppendDecimal128(nil, decimal128)},
		{"min key", primitive.MinKey{}, bsontype.MinKey, []byte{}},
		{"max key", primitive.MaxKey{}, bsontype.MaxKey, []byte{}},
		{"struct", structTest, bsontype.EmbeddedDocument, structCore},
		{"D", D{{"foo", int32(10)}}, bsontype.EmbeddedDocument, structCore},
		{"M", M{"foo": int32(10)}, bsontype.EmbeddedDocument, structCore},
		{"ValueMarshaler", marshalValueMarshaler{Foo: 10}, bsontype.Int32, bsoncore.AppendInt32(nil, 10)},
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
		marshalValueTestCase{"interface", interfaceTest, bsontype.EmbeddedDocument, interfaceCore},
	)

	return marshalValueTestCases
}
