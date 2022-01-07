// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"reflect"

	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

type unmarshalingTestCase struct {
	name  string
	sType reflect.Type
	want  interface{}
	data  []byte
}

var unmarshalingTestCases = func() []unmarshalingTestCase {
	var zeroMyStruct myStruct
	{
		i := myInt64(0)
		m := myMap{}
		b := myBytes{}
		s := myString("")
		zeroMyStruct = myStruct{I: &i, M: &m, B: &b, S: &s}
	}

	var valMyStruct myStruct
	{
		i := myInt64(5)
		m := myMap{"key": "value"}
		b := myBytes{0x00, 0x01}
		s := myString("test")
		valMyStruct = myStruct{I: &i, M: &m, B: &b, S: &s}
	}

	return []unmarshalingTestCase{
		{
			name: "small struct",
			sType: reflect.TypeOf(struct {
				Foo bool
			}{}),
			want: &struct {
				Foo bool
			}{Foo: true},
			data: docToBytes(D{{"foo", true}}),
		},
		{
			name: "nested document",
			sType: reflect.TypeOf(struct {
				Foo struct {
					Bar bool
				}
			}{}),
			want: &struct {
				Foo struct {
					Bar bool
				}
			}{
				Foo: struct {
					Bar bool
				}{Bar: true},
			},
			data: docToBytes(D{{"foo", D{{"bar", true}}}}),
		},
		{
			name: "simple array",
			sType: reflect.TypeOf(struct {
				Foo []bool
			}{}),
			want: &struct {
				Foo []bool
			}{
				Foo: []bool{true},
			},
			data: docToBytes(D{{"foo", A{true}}}),
		},
		{
			name: "struct with mixed case fields",
			sType: reflect.TypeOf(struct {
				FooBar int32
			}{}),
			want: &struct {
				FooBar int32
			}{
				FooBar: 10,
			},
			data: docToBytes(D{{"fooBar", int32(10)}}),
		},
		// GODRIVER-2252
		// Test that a struct of pointer types with UnmarshalBSON functions defined marshal and
		// unmarshal to the same Go values when the pointer values are "nil".
		{
			name:  "fields with UnmarshalBSON function should marshal and unmarshal to the same values",
			sType: reflect.TypeOf(myStruct{}),
			want:  &myStruct{},
			data:  docToBytes(myStruct{}),
		},
		// GODRIVER-2252
		// Test that a struct of pointer types with UnmarshalBSON functions defined marshal and
		// unmarshal to the same Go values when the pointer values are the respective zero values.
		{
			name:  "TODO",
			sType: reflect.TypeOf(myStruct{}),
			want:  &zeroMyStruct,
			data:  docToBytes(zeroMyStruct),
		},
		// GODRIVER-2252
		// Test that a struct of pointer types with UnmarshalBSON functions defined marshal and
		// unmarshal to the same Go values when the pointer values are non-zero values.
		{
			name:  "TODO",
			sType: reflect.TypeOf(myStruct{}),
			want:  &valMyStruct,
			data:  docToBytes(valMyStruct),
		},
	}
}()

type myStruct struct {
	I *myInt64
	M *myMap
	B *myBytes
	S *myString
}

type myInt64 int64

func (mi *myInt64) UnmarshalBSON(bytes []byte) error {
	i, err := bsonrw.NewBSONValueReader(bsontype.Int64, bytes).ReadInt64()
	if err != nil {
		return err
	}
	*mi = myInt64(i)
	return nil
}

type myMap map[string]string

func (mm *myMap) UnmarshalBSON(bytes []byte) error {
	var m map[string]string
	err := Unmarshal(bytes, &m)
	*mm = myMap(m)
	return err
}

type myBytes []byte

func (mb *myBytes) UnmarshalBSON(bytes []byte) error {
	b, _, err := bsonrw.NewBSONValueReader(bsontype.Binary, bytes).ReadBinary()
	if err != nil {
		return err
	}
	*mb = b
	return nil
}

type myString string

func (ms *myString) UnmarshalBSON(bytes []byte) error {
	s, err := bsonrw.NewBSONValueReader(bsontype.String, bytes).ReadString()
	if err != nil {
		return err
	}
	*ms = myString(s)
	return nil
}
