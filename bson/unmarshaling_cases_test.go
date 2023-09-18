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

func unmarshalingTestCases() []unmarshalingTestCase {
	var zeroPtrStruct unmarshalerPtrStruct
	{
		i := myInt64(0)
		m := myMap{}
		b := myBytes{}
		s := myString("")
		zeroPtrStruct = unmarshalerPtrStruct{I: &i, M: &m, B: &b, S: &s}
	}

	var zeroNonPtrStruct unmarshalerNonPtrStruct
	{
		i := myInt64(0)
		m := myMap{}
		b := myBytes{}
		s := myString("")
		zeroNonPtrStruct = unmarshalerNonPtrStruct{I: i, M: m, B: b, S: s}
	}

	var valPtrStruct unmarshalerPtrStruct
	{
		i := myInt64(5)
		m := myMap{"key": "value"}
		b := myBytes{0x00, 0x01}
		s := myString("test")
		valPtrStruct = unmarshalerPtrStruct{I: &i, M: &m, B: &b, S: &s}
	}

	var valNonPtrStruct unmarshalerNonPtrStruct
	{
		i := myInt64(5)
		m := myMap{"key": "value"}
		b := myBytes{0x00, 0x01}
		s := myString("test")
		valNonPtrStruct = unmarshalerNonPtrStruct{I: i, M: m, B: b, S: s}
	}

	type fooBytes struct {
		Foo []byte
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
			name:  "nil pointer fields with UnmarshalBSON function should marshal and unmarshal to the same values",
			sType: reflect.TypeOf(unmarshalerPtrStruct{}),
			want:  &unmarshalerPtrStruct{},
			data:  docToBytes(unmarshalerPtrStruct{}),
		},
		// GODRIVER-2252
		// Test that a struct of pointer types with UnmarshalBSON functions defined marshal and
		// unmarshal to the same Go values when the pointer values are the respective zero values.
		{
			name:  "zero-value pointer fields with UnmarshalBSON function should marshal and unmarshal to the same values",
			sType: reflect.TypeOf(unmarshalerPtrStruct{}),
			want:  &zeroPtrStruct,
			data:  docToBytes(zeroPtrStruct),
		},
		// GODRIVER-2252
		// Test that a struct of pointer types with UnmarshalBSON functions defined marshal and
		// unmarshal to the same Go values when the pointer values are non-zero values.
		{
			name:  "non-zero-value pointer fields with UnmarshalBSON function should marshal and unmarshal to the same values",
			sType: reflect.TypeOf(unmarshalerPtrStruct{}),
			want:  &valPtrStruct,
			data:  docToBytes(valPtrStruct),
		},
		// GODRIVER-2311
		// Test that an unmarshaled struct that has a byte slice value does not reference the same
		// underlying array as the input.
		{
			name:  "struct with byte slice",
			sType: reflect.TypeOf(fooBytes{}),
			want: &fooBytes{
				Foo: []byte{0, 1, 2, 3, 4, 5},
			},
			data: docToBytes(fooBytes{
				Foo: []byte{0, 1, 2, 3, 4, 5},
			}),
		},
		// GODRIVER-2427
		// Test that a struct of non-pointer types with UnmarshalBSON functions defined for the pointer of the field
		// will marshal and unmarshal to the same Go values when the non-pointer values are the respctive zero values.
		{
			name: `zero-value non-pointer fields with pointer UnmarshalBSON function should marshal and unmarshal to
			the same values`,
			sType: reflect.TypeOf(unmarshalerNonPtrStruct{}),
			want:  &zeroNonPtrStruct,
			data:  docToBytes(zeroNonPtrStruct),
		},
		// GODRIVER-2427
		// Test that a struct of non-pointer types with UnmarshalBSON functions defined for the pointer of the field
		// unmarshal to the same Go values when the non-pointer values are non-zero values.
		{
			name: `non-zero-value non-pointer fields with pointer UnmarshalBSON function should marshal and unmarshal
			to the same values`,
			sType: reflect.TypeOf(unmarshalerNonPtrStruct{}),
			want:  &valNonPtrStruct,
			data:  docToBytes(valNonPtrStruct),
		},
	}
}

// unmarshalerPtrStruct contains a collection of fields that are all pointers to custom types that
// implement the bson.Unmarshaler interface. It is used to test the BSON unmarshal behavior for
// pointer types with custom UnmarshalBSON functions.
type unmarshalerPtrStruct struct {
	I *myInt64
	M *myMap
	B *myBytes
	S *myString
}

// unmarshalerNonPtrStruct contains a collection of non-pointer fields that are all to custom types that implement the
// bson.Unmarshaler interface. It is used to test the BSON unmarshal behavior for types with custom UnmarshalBSON
// functions.
type unmarshalerNonPtrStruct struct {
	I myInt64
	M myMap
	B myBytes
	S myString
}

type myInt64 int64

func (mi *myInt64) UnmarshalBSON(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}
	i, err := bsonrw.NewBSONValueReader(bsontype.Int64, bytes).ReadInt64()
	if err != nil {
		return err
	}
	*mi = myInt64(i)
	return nil
}

type myMap map[string]string

func (mm *myMap) UnmarshalBSON(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}
	var m map[string]string
	err := Unmarshal(bytes, &m)
	*mm = myMap(m)
	return err
}

type myBytes []byte

func (mb *myBytes) UnmarshalBSON(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}
	b, _, err := bsonrw.NewBSONValueReader(bsontype.Binary, bytes).ReadBinary()
	if err != nil {
		return err
	}
	*mb = b
	return nil
}

type myString string

func (ms *myString) UnmarshalBSON(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}
	s, err := bsonrw.NewBSONValueReader(bsontype.String, bytes).ReadString()
	if err != nil {
		return err
	}
	*ms = myString(s)
	return nil
}
