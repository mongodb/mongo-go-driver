// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"reflect"
)

// emptyInterfaceCodec is the Codec used for interface{} values.
type emptyInterfaceCodec struct {
	// decodeBinaryAsSlice causes DecodeValue to unmarshal BSON binary field values that are the
	// "Generic" or "Old" BSON binary subtype as a Go byte slice instead of a Binary.
	decodeBinaryAsSlice bool
}

// Assert that emptyInterfaceCodec satisfies the typeDecoder interface, which allows it
// to be used by collection type decoders (e.g. map, slice, etc) to set individual values in a
// collection.
var _ typeDecoder = &emptyInterfaceCodec{}

// EncodeValue is the ValueEncoderFunc for interface{}.
func (eic *emptyInterfaceCodec) EncodeValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
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

func (eic *emptyInterfaceCodec) getEmptyInterfaceDecodeType(dc DecodeContext, valueType Type) (reflect.Type, error) {
	isDocument := valueType == Type(0) || valueType == TypeEmbeddedDocument
	if isDocument {
		if dc.defaultDocumentType != nil {
			// If the bsontype is an embedded document and the DocumentType is set on the DecodeContext, then return
			// that type.
			return dc.defaultDocumentType, nil
		}
	}

	rtype, err := dc.LookupTypeMapEntry(valueType)
	if err == nil {
		return rtype, nil
	}

	if isDocument {
		// For documents, fallback to looking up a type map entry for Type(0) or TypeEmbeddedDocument,
		// depending on the original valueType.
		var lookupType Type
		switch valueType {
		case Type(0):
			lookupType = TypeEmbeddedDocument
		case TypeEmbeddedDocument:
			lookupType = Type(0)
		}

		rtype, err = dc.LookupTypeMapEntry(lookupType)
		if err == nil {
			return rtype, nil
		}
		// fallback to bson.D
		return tD, nil
	}

	return nil, err
}

func (eic *emptyInterfaceCodec) decodeType(dc DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tEmpty {
		return emptyValue, ValueDecoderError{Name: "EmptyInterfaceDecodeValue", Types: []reflect.Type{tEmpty}, Received: reflect.Zero(t)}
	}

	rtype, err := eic.getEmptyInterfaceDecodeType(dc, vr.Type())
	if err != nil {
		switch vr.Type() {
		case TypeNull:
			return reflect.Zero(t), vr.ReadNull()
		default:
			return emptyValue, err
		}
	}

	decoder, err := dc.LookupDecoder(rtype)
	if err != nil {
		return emptyValue, err
	}

	elem, err := decodeTypeOrValueWithInfo(decoder, dc, vr, rtype)
	if err != nil {
		return emptyValue, err
	}

	if (eic.decodeBinaryAsSlice || dc.binaryAsSlice) && rtype == tBinary {
		binElem := elem.Interface().(Binary)
		if binElem.Subtype == TypeBinaryGeneric || binElem.Subtype == TypeBinaryBinaryOld {
			elem = reflect.ValueOf(binElem.Data)
		}
	}

	return elem, nil
}

// DecodeValue is the ValueDecoderFunc for interface{}.
func (eic *emptyInterfaceCodec) DecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tEmpty {
		return ValueDecoderError{Name: "EmptyInterfaceDecodeValue", Types: []reflect.Type{tEmpty}, Received: val}
	}

	elem, err := eic.decodeType(dc, vr, val.Type())
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}
