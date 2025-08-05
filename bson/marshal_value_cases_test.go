// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"io"

	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

var marshalValueTestCases = []marshalValueTestCase{
	{
		name:     "double",
		val:      3.14,
		bsontype: TypeDouble,
		bytes:    bsoncore.AppendDouble(nil, 3.14),
	},
	{
		name:     "string",
		val:      "hello world",
		bsontype: TypeString,
		bytes:    bsoncore.AppendString(nil, "hello world"),
	},
	{
		name:     "binary",
		val:      Binary{1, []byte{1, 2}},
		bsontype: TypeBinary,
		bytes:    bsoncore.AppendBinary(nil, 1, []byte{1, 2}),
	},
	{
		name:     "undefined",
		val:      Undefined{},
		bsontype: TypeUndefined,
		bytes:    []byte{},
	},
	{
		name:     "object id",
		val:      ObjectID{103, 116, 166, 161, 70, 33, 67, 139, 164, 144, 255, 112},
		bsontype: TypeObjectID,
		bytes:    bsoncore.AppendObjectID(nil, ObjectID{103, 116, 166, 161, 70, 33, 67, 139, 164, 144, 255, 112}),
	},
	{
		name:     "boolean",
		val:      true,
		bsontype: TypeBoolean,
		bytes:    bsoncore.AppendBoolean(nil, true),
	},
	{
		name:     "datetime",
		val:      DateTime(5),
		bsontype: TypeDateTime,
		bytes:    bsoncore.AppendDateTime(nil, 5),
	},
	{
		name:     "null",
		val:      Null{},
		bsontype: TypeNull,
		bytes:    []byte{},
	},
	{
		name:     "regex",
		val:      Regex{Pattern: "pattern", Options: "imx"},
		bsontype: TypeRegex,
		bytes:    bsoncore.AppendRegex(nil, "pattern", "imx"),
	},
	{
		name: "dbpointer",
		val: DBPointer{
			DB:      "db",
			Pointer: ObjectID{103, 116, 166, 161, 70, 33, 67, 139, 164, 144, 255, 112},
		},
		bsontype: TypeDBPointer,
		bytes: bsoncore.AppendDBPointer(
			nil,
			"db",
			ObjectID{103, 116, 166, 161, 70, 33, 67, 139, 164, 144, 255, 112},
		),
	},
	{
		name:     "javascript",
		val:      JavaScript("js"),
		bsontype: TypeJavaScript,
		bytes:    bsoncore.AppendJavaScript(nil, "js"),
	},
	{
		name:     "symbol",
		val:      Symbol("symbol"),
		bsontype: TypeSymbol,
		bytes:    bsoncore.AppendSymbol(nil, "symbol"),
	},
	{
		name:     "code with scope",
		val:      CodeWithScope{Code: "code", Scope: D{{"a", "b"}}},
		bsontype: TypeCodeWithScope,
		bytes: bsoncore.AppendCodeWithScope(
			nil,
			"code",
			bsoncore.NewDocumentBuilder().
				AppendString("a", "b").
				Build()),
	},
	{
		name:     "int32",
		val:      5,
		bsontype: TypeInt32,
		bytes:    bsoncore.AppendInt32(nil, 5),
	},
	{
		name:     "int64",
		val:      int64(5),
		bsontype: TypeInt64,
		bytes:    bsoncore.AppendInt64(nil, 5),
	},
	{
		name:     "timestamp",
		val:      Timestamp{T: 1, I: 5},
		bsontype: TypeTimestamp,
		bytes:    bsoncore.AppendTimestamp(nil, 1, 5),
	},
	{
		name:     "decimal128",
		val:      NewDecimal128(5, 10),
		bsontype: TypeDecimal128,
		bytes:    bsoncore.AppendDecimal128(nil, 5, 10),
	},
	{
		name:     "min key",
		val:      MinKey{},
		bsontype: TypeMinKey,
		bytes:    []byte{},
	},
	{
		name:     "max key",
		val:      MaxKey{},
		bsontype: TypeMaxKey,
		bytes:    []byte{},
	},
	{
		name:     "struct",
		val:      marshalValueStruct{Foo: 10},
		bsontype: TypeEmbeddedDocument,
		bytes: bsoncore.NewDocumentBuilder().
			AppendInt32("foo", 10).
			Build(),
	},
	{
		name:     "D",
		val:      D{{"foo", int32(10)}},
		bsontype: TypeEmbeddedDocument,
		bytes: bsoncore.NewDocumentBuilder().
			AppendInt32("foo", 10).
			Build(),
	},
	{
		name:     "M",
		val:      M{"foo": int32(10)},
		bsontype: TypeEmbeddedDocument,
		bytes: bsoncore.NewDocumentBuilder().
			AppendInt32("foo", 10).
			Build(),
	},
	{
		name:     "ValueMarshaler",
		val:      marshalValueMarshaler{Foo: 10},
		bsontype: TypeInt32,
		bytes:    bsoncore.AppendInt32(nil, 10),
	},
}

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
	val      any
	bsontype Type
	bytes    []byte
}
