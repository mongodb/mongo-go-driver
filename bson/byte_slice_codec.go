// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"fmt"
	"reflect"
)

// byteSliceCodec is the Codec used for []byte values.
type byteSliceCodec struct {
	// encodeNilAsEmpty causes EncodeValue to marshal nil Go byte slices as empty BSON binary values
	// instead of BSON null.
	encodeNilAsEmpty bool
}

// Assert that byteSliceCodec satisfies the typeDecoder interface, which allows it to be
// used by collection type decoders (e.g. map, slice, etc) to set individual values in a
// collection.
var _ typeDecoder = &byteSliceCodec{}

// EncodeValue is the ValueEncoder for []byte.
func (bsc *byteSliceCodec) EncodeValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tByteSlice {
		return ValueEncoderError{Name: "ByteSliceEncodeValue", Types: []reflect.Type{tByteSlice}, Received: val}
	}
	if val.IsNil() && !bsc.encodeNilAsEmpty && !ec.nilByteSliceAsEmpty {
		return vw.WriteNull()
	}
	return vw.WriteBinary(val.Interface().([]byte))
}

func (bsc *byteSliceCodec) decodeType(_ DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tByteSlice {
		return emptyValue, ValueDecoderError{
			Name:     "ByteSliceDecodeValue",
			Types:    []reflect.Type{tByteSlice},
			Received: reflect.Zero(t),
		}
	}

	var data []byte
	var err error
	switch vrType := vr.Type(); vrType {
	case TypeString:
		str, err := vr.ReadString()
		if err != nil {
			return emptyValue, err
		}
		data = []byte(str)
	case TypeSymbol:
		sym, err := vr.ReadSymbol()
		if err != nil {
			return emptyValue, err
		}
		data = []byte(sym)
	case TypeBinary:
		var subtype byte
		data, subtype, err = vr.ReadBinary()
		if err != nil {
			return emptyValue, err
		}
		if subtype != TypeBinaryGeneric && subtype != TypeBinaryBinaryOld {
			return emptyValue, decodeBinaryError{subtype: subtype, typeName: "[]byte"}
		}
	case TypeNull:
		err = vr.ReadNull()
	case TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a []byte", vrType)
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(data), nil
}

// DecodeValue is the ValueDecoder for []byte.
func (bsc *byteSliceCodec) DecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tByteSlice {
		return ValueDecoderError{Name: "ByteSliceDecodeValue", Types: []reflect.Type{tByteSlice}, Received: val}
	}

	elem, err := bsc.decodeType(dc, vr, tByteSlice)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}
