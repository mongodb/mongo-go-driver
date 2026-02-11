// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"fmt"
	"reflect"
	"strings"
)

type unmarshalingTestCase struct {
	name  string
	sType reflect.Type
	want  any
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

	longKey := strings.Repeat("k", 16_000_000)
	tLongKey := reflect.StructOf([]reflect.StructField{
		{
			Name: "Foo",
			Type: reflect.TypeOf(false),
			Tag:  reflect.StructTag(fmt.Sprintf(`bson:"%s"`, longKey)),
		},
	})

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
		{
			name:  "nil pointer and non-pointer type with literal null BSON",
			sType: reflect.TypeOf(unmarshalBehaviorTestCase{}),
			want: &unmarshalBehaviorTestCase{
				BSONValueTracker: unmarshalBSONValueCallTracker{
					called: true,
				},
				BSONValuePtrTracker: nil,
				BSONTracker: unmarshalBSONCallTracker{
					called: true,
				},
				BSONPtrTracker: nil,
			},
			data: docToBytes(D{
				{Key: "bv_tracker", Value: nil},
				{Key: "bv_ptr_tracker", Value: nil},
				{Key: "b_tracker", Value: nil},
				{Key: "b_ptr_tracker", Value: nil},
			}),
		},
		{
			name:  "nil pointer and non-pointer type with BSON minkey",
			sType: reflect.TypeOf(unmarshalBehaviorTestCase{}),
			want: &unmarshalBehaviorTestCase{
				BSONValueTracker: unmarshalBSONValueCallTracker{
					called: true,
				},
				BSONValuePtrTracker: &unmarshalBSONValueCallTracker{
					called: true,
				},
				BSONTracker: unmarshalBSONCallTracker{
					called: true,
				},
				BSONPtrTracker: nil,
			},
			data: docToBytes(D{
				{Key: "bv_tracker", Value: MinKey{}},
				{Key: "bv_ptr_tracker", Value: MinKey{}},
				{Key: "b_tracker", Value: MinKey{}},
				{Key: "b_ptr_tracker", Value: MinKey{}},
			}),
		},
		{
			name:  "nil pointer and non-pointer type with BSON maxkey",
			sType: reflect.TypeOf(unmarshalBehaviorTestCase{}),
			want: &unmarshalBehaviorTestCase{
				BSONValueTracker: unmarshalBSONValueCallTracker{
					called: true,
				},
				BSONValuePtrTracker: &unmarshalBSONValueCallTracker{
					called: true,
				},
				BSONTracker: unmarshalBSONCallTracker{
					called: true,
				},
				BSONPtrTracker: nil,
			},
			data: docToBytes(D{
				{Key: "bv_tracker", Value: MaxKey{}},
				{Key: "bv_ptr_tracker", Value: MaxKey{}},
				{Key: "b_tracker", Value: MaxKey{}},
				{Key: "b_ptr_tracker", Value: MaxKey{}},
			}),
		},
		{
			name:  "long key",
			sType: tLongKey,
			want: func() any {
				vLongKey := reflect.New(tLongKey)
				vLongKey.Elem().Field(0).SetBool(true)
				return vLongKey.Interface()
			}(),
			data: docToBytes(D{{longKey, true}}),
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

var _ ValueUnmarshaler = (*myInt64)(nil)

func (mi *myInt64) UnmarshalBSONValue(t byte, b []byte) error {
	if len(b) == 0 {
		return nil
	}

	if Type(t) == TypeInt64 {
		i, err := newBufferedValueReader(TypeInt64, b).ReadInt64()
		if err != nil {
			return err
		}

		*mi = myInt64(i)
	}

	return nil
}

func (mi *myInt64) UnmarshalBSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	i, err := newBufferedValueReader(TypeInt64, b).ReadInt64()
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

func (mb *myBytes) UnmarshalBSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	b, _, err := newBufferedValueReader(TypeBinary, b).ReadBinary()
	if err != nil {
		return err
	}
	*mb = b
	return nil
}

type myString string

func (ms *myString) UnmarshalBSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	s, err := newBufferedValueReader(TypeString, b).ReadString()
	if err != nil {
		return err
	}
	*ms = myString(s)
	return nil
}

// unmarshalBSONValueCallTracker is a test struct that tracks whether the
// UnmarshalBSONValue method has been called.
type unmarshalBSONValueCallTracker struct {
	called bool // called is set to true when UnmarshalBSONValue is invoked.
}

var _ ValueUnmarshaler = &unmarshalBSONValueCallTracker{}

// unmarshalBSONCallTracker is a test struct that tracks whether the
// UnmarshalBSON method has been called.
type unmarshalBSONCallTracker struct {
	called bool // called is set to true when UnmarshalBSON is invoked.
}

// Ensure unmarshalBSONCallTracker implements the Unmarshaler interface.
var _ Unmarshaler = &unmarshalBSONCallTracker{}

// unmarshalBehaviorTestCase holds instances of call trackers for testing BSON
// unmarshaling behavior.
type unmarshalBehaviorTestCase struct {
	BSONValueTracker    unmarshalBSONValueCallTracker  `bson:"bv_tracker"`     // BSON value unmarshaling by value.
	BSONValuePtrTracker *unmarshalBSONValueCallTracker `bson:"bv_ptr_tracker"` // BSON value unmarshaling by pointer.
	BSONTracker         unmarshalBSONCallTracker       `bson:"b_tracker"`      // BSON unmarshaling by value.
	BSONPtrTracker      *unmarshalBSONCallTracker      `bson:"b_ptr_tracker"`  // BSON unmarshaling by pointer.
}

func (tracker *unmarshalBSONValueCallTracker) UnmarshalBSONValue(byte, []byte) error {
	tracker.called = true
	return nil
}

func (tracker *unmarshalBSONCallTracker) UnmarshalBSON([]byte) error {
	tracker.called = true
	return nil
}
