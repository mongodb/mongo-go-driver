// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncodec_test

import (
	"fmt"
	"math"
	"reflect"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

func ExampleRegistry_customEncoder() {
	// Create a custom encoder for an integer type that multiplies the input
	// value by -1 when encoding.
	type negatedInt int

	negatedIntType := reflect.TypeOf(negatedInt(0))

	negatedIntEncoder := func(
		ec bsoncodec.EncodeContext,
		vw bsonrw.ValueWriter,
		val reflect.Value,
	) error {
		// All encoder implementations should check that val is valid and is of
		// the correct type before proceeding.
		if !val.IsValid() || val.Type() != negatedIntType {
			return bsoncodec.ValueEncoderError{
				Name:     "negatedIntEncoder",
				Types:    []reflect.Type{negatedIntType},
				Received: val,
			}
		}

		// Negate val and encode as a BSON int32 if it can fit in 32 bits and a
		// BSON int64 otherwise.
		negatedVal := val.Int() * -1
		if math.MinInt32 <= negatedVal && negatedVal <= math.MaxInt32 {
			return vw.WriteInt32(int32(negatedVal))
		}
		return vw.WriteInt64(negatedVal)
	}

	reg := bson.NewRegistry()
	reg.RegisterTypeEncoder(
		negatedIntType,
		bsoncodec.ValueEncoderFunc(negatedIntEncoder))

	// Define a document that includes both int and negatedInt fields with the
	// same value.
	type myDocument struct {
		Int        int
		NegatedInt negatedInt
	}
	doc := myDocument{
		Int:        1,
		NegatedInt: 1,
	}

	// Marshal the document as BSON. Expect that the int field is encoded to the
	// same value and that the negatedInt field is encoded as the negated value.
	b, err := bson.MarshalWithRegistry(reg, doc)
	if err != nil {
		panic(err)
	}
	fmt.Println(bson.Raw(b).String())
	// Output: {"int": {"$numberInt":"1"},"negatedint": {"$numberInt":"-1"}}
}

func ExampleRegistry_customDecoder() {
	// Create a custom decoder for a boolean type that can be stored in the
	// database as a BSON boolean, int32, or int64. For our custom decoder, BSON
	// int32 or int64 values are considered "true" if they are non-zero.
	type lenientBool bool

	lenientBoolType := reflect.TypeOf(lenientBool(true))

	lenientBoolDecoder := func(
		dc bsoncodec.DecodeContext,
		vr bsonrw.ValueReader,
		val reflect.Value,
	) error {
		// All decoder implementations should check that val is valid, settable,
		// and is of the correct kind before proceeding.
		if !val.IsValid() || !val.CanSet() || val.Type() != lenientBoolType {
			return bsoncodec.ValueDecoderError{
				Name:     "lenientBoolDecoder",
				Types:    []reflect.Type{lenientBoolType},
				Received: val,
			}
		}

		var result bool
		switch vr.Type() {
		case bsontype.Boolean:
			b, err := vr.ReadBoolean()
			if err != nil {
				return err
			}
			result = b
		case bsontype.Int32:
			i32, err := vr.ReadInt32()
			if err != nil {
				return err
			}
			result = i32 != 0
		case bsontype.Int64:
			i64, err := vr.ReadInt64()
			if err != nil {
				return err
			}
			result = i64 != 0
		default:
			return fmt.Errorf(
				"received invalid BSON type to decode into lenientBool: %s",
				vr.Type())
		}

		val.SetBool(result)
		return nil
	}

	reg := bson.NewRegistry()
	reg.RegisterTypeDecoder(
		lenientBoolType,
		bsoncodec.ValueDecoderFunc(lenientBoolDecoder))

	// Marshal a BSON document with a single field "isOK" that is a non-zero
	// integer value.
	b, err := bson.Marshal(bson.M{"isOK": 1})
	if err != nil {
		panic(err)
	}

	// Now try to decode the BSON document to a struct with a field "IsOK" that
	// is type lenientBool. Expect that the non-zero integer value is decoded
	// as boolean true.
	type MyDocument struct {
		IsOK lenientBool `bson:"isOK"`
	}
	var doc MyDocument
	err = bson.UnmarshalWithRegistry(reg, b, &doc)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", doc)
	// Output: {IsOK:true}
}

func ExampleRegistry_RegisterKindEncoder() {
	// Create a custom encoder that writes any Go type that has underlying type
	// int32 as an a BSON int64. To do that, we register the encoder as a "kind"
	// encoder for kind reflect.Int32. That way, even user-defined types with
	// underlying type int32 will be encoded as a BSON int64.
	int32To64Encoder := func(
		ec bsoncodec.EncodeContext,
		vw bsonrw.ValueWriter,
		val reflect.Value,
	) error {
		// All encoder implementations should check that val is valid and is of
		// the correct type or kind before proceeding.
		if !val.IsValid() || val.Kind() != reflect.Int32 {
			return bsoncodec.ValueEncoderError{
				Name:     "int32To64Encoder",
				Kinds:    []reflect.Kind{reflect.Int32},
				Received: val,
			}
		}

		return vw.WriteInt64(val.Int())
	}

	// Create a default registry and register our int32-to-int64 encoder for
	// kind reflect.Int32.
	reg := bson.NewRegistry()
	reg.RegisterKindEncoder(
		reflect.Int32,
		bsoncodec.ValueEncoderFunc(int32To64Encoder))

	// Define a document that includes an int32, an int64, and a user-defined
	// type "myInt" that has underlying type int32.
	type myInt int32
	type myDocument struct {
		MyInt myInt
		Int32 int32
		Int64 int64
	}
	doc := myDocument{
		Int32: 1,
		Int64: 1,
		MyInt: 1,
	}

	// Marshal the document as BSON. Expect that all fields are encoded as BSON
	// int64 (represented as "$numberLong" when encoded as Extended JSON).
	b, err := bson.MarshalWithRegistry(reg, doc)
	if err != nil {
		panic(err)
	}
	fmt.Println(bson.Raw(b).String())
	// Output: {"myint": {"$numberLong":"1"},"int32": {"$numberLong":"1"},"int64": {"$numberLong":"1"}}
}

func ExampleRegistry_RegisterKindDecoder() {
	// Create a custom decoder that can decode any integer value, including
	// integer values encoded as floating point numbers, to any Go type
	// with underlying type int64. To do that, we register the decoder as a
	// "kind" decoder for kind reflect.Int64. That way, we can even decode to
	// user-defined types with underlying type int64.
	flexibleInt64KindDecoder := func(
		dc bsoncodec.DecodeContext,
		vr bsonrw.ValueReader,
		val reflect.Value,
	) error {
		// All decoder implementations should check that val is valid, settable,
		// and is of the correct kind before proceeding.
		if !val.IsValid() || !val.CanSet() || val.Kind() != reflect.Int64 {
			return bsoncodec.ValueDecoderError{
				Name:     "flexibleInt64KindDecoder",
				Kinds:    []reflect.Kind{reflect.Int64},
				Received: val,
			}
		}

		var result int64
		switch vr.Type() {
		case bsontype.Int64:
			i64, err := vr.ReadInt64()
			if err != nil {
				return err
			}
			result = i64
		case bsontype.Int32:
			i32, err := vr.ReadInt32()
			if err != nil {
				return err
			}
			result = int64(i32)
		case bsontype.Double:
			d, err := vr.ReadDouble()
			if err != nil {
				return err
			}
			i64 := int64(d)
			// Make sure the double field is an integer value.
			if d != float64(i64) {
				return fmt.Errorf("double %f is not an integer value", d)
			}
			result = i64
		default:
			return fmt.Errorf(
				"received invalid BSON type to decode into int64: %s",
				vr.Type())
		}

		val.SetInt(result)
		return nil
	}

	reg := bson.NewRegistry()
	reg.RegisterKindDecoder(
		reflect.Int64,
		bsoncodec.ValueDecoderFunc(flexibleInt64KindDecoder))

	// Marshal a BSON document with fields that are mixed numeric types but all
	// hold integer values (i.e. values with no fractional part).
	b, err := bson.Marshal(bson.M{"myInt": float64(8), "int64": int32(9)})
	if err != nil {
		panic(err)
	}

	// Now try to decode the BSON document to a struct with fields
	// that is type int32. Expect that the float value is successfully decoded.
	type myInt int64
	type myDocument struct {
		MyInt myInt
		Int64 int64
	}
	var doc myDocument
	err = bson.UnmarshalWithRegistry(reg, b, &doc)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", doc)
	// Output: {MyInt:8 Int64:9}
}
