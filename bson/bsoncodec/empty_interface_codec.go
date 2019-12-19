// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncodec

import (
	"reflect"

	"go.mongodb.org/mongo-driver/bson/bsonoptions"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var defaultEmptyInterfaceCodec = NewEmptyInterfaceCodec()

// EmptyInterfaceCodec is the Codec used for interface{} values.
type EmptyInterfaceCodec struct {
	DecodeAsMap         bool
	DecodeBinaryAsSlice bool
}

var _ ValueCodec = &EmptyInterfaceCodec{}

// NewEmptyInterfaceCodec returns a EmptyInterfaceCodec with options opts.
func NewEmptyInterfaceCodec(opts ...*bsonoptions.EmptyInterfaceCodecOptions) *EmptyInterfaceCodec {
	interfaceOpt := bsonoptions.MergeEmptyInterfaceCodecOptions(opts...)

	codec := EmptyInterfaceCodec{}
	if interfaceOpt.DecodeAsMap != nil {
		codec.DecodeAsMap = *interfaceOpt.DecodeAsMap
	}
	if interfaceOpt.DecodeBinaryAsSlice != nil {
		codec.DecodeBinaryAsSlice = *interfaceOpt.DecodeBinaryAsSlice
	}
	return &codec
}

// EncodeValue is the ValueEncoderFunc for interface{}.
func (eic EmptyInterfaceCodec) EncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tEmpty {
		return ValueEncoderError{Name: "EmptyInterfaceEncodeValue", Types: []reflect.Type{tEmpty}, Received: val}
	}

	if val.IsNil() {
		return vw.WriteNull()
	}
	encoder, err := ec.LookupEncoder(val.Elem().Type())
	if err != nil {
		return err
	}

	return encoder.EncodeValue(ec, vw, val.Elem())
}

// DecodeValue is the ValueDecoderFunc for interface{}.
func (eic EmptyInterfaceCodec) DecodeValue(dc DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tEmpty {
		return ValueDecoderError{Name: "EmptyInterfaceDecodeValue", Types: []reflect.Type{tEmpty}, Received: val}
	}

	rtype, err := dc.LookupTypeMapEntry(vr.Type())
	if err != nil {
		switch vr.Type() {
		case bsontype.EmbeddedDocument:
			if dc.Ancestor != nil {
				rtype = dc.Ancestor
				break
			}
			rtype = tD
			if eic.DecodeAsMap {
				rtype = tM
			}
		case bsontype.Null:
			val.Set(reflect.Zero(val.Type()))
			return vr.ReadNull()
		default:
			return err
		}
	}

	decoder, err := dc.LookupDecoder(rtype)
	if err != nil {
		return err
	}

	elem := reflect.New(rtype).Elem()
	err = decoder.DecodeValue(dc, vr, elem)
	if err != nil {
		return err
	}
	if eic.DecodeBinaryAsSlice && rtype == tBinary {
		binElem := elem.Interface().(primitive.Binary)
		if binElem.Subtype == bsontype.BinaryGeneric || binElem.Subtype == bsontype.BinaryBinaryOld {
			elem = reflect.ValueOf(binElem.Data)
		}
	}

	val.Set(elem)
	return nil
}
