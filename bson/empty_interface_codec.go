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
	// defaultDocumentType specifies the Go type to decode top-level and nested BSON documents into. In particular, the
	// usage for this field is restricted to data typed as "interface{}" or "map[string]interface{}". If DocumentType is
	// set to a type that a BSON document cannot be unmarshaled into (e.g. "string"), unmarshalling will result in an
	// error. DocumentType overrides the Ancestor field.
	defaultDocumentType reflect.Type

	// decodeBinaryAsSlice causes DecodeValue to unmarshal BSON binary field values that are the
	// "Generic" or "Old" BSON binary subtype as a Go byte slice instead of a Binary.
	decodeBinaryAsSlice bool
}

// EncodeValue is the ValueEncoderFunc for interface{}.
func (eic emptyInterfaceCodec) EncodeValue(reg EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tEmpty {
		return ValueEncoderError{Name: "EmptyInterfaceEncodeValue", Types: []reflect.Type{tEmpty}, Received: val}
	}

	if val.IsNil() {
		return vw.WriteNull()
	}
	encoder, err := reg.LookupEncoder(val.Elem().Type())
	if err != nil {
		return err
	}

	return encoder.EncodeValue(reg, vw, val.Elem())
}

func (eic emptyInterfaceCodec) getEmptyInterfaceDecodeType(reg DecoderRegistry, valueType Type, ancestorType reflect.Type) (reflect.Type, error) {
	isDocument := valueType == Type(0) || valueType == TypeEmbeddedDocument
	if isDocument {
		if eic.defaultDocumentType != nil {
			// If the bsontype is an embedded document and the DocumentType is set on the DecodeContext, then return
			// that type.
			return eic.defaultDocumentType, nil
		}
		if ancestorType != nil && ancestorType != tEmpty {
			// Using ancestor information rather than looking up the type map entry forces consistent decoding.
			// If we're decoding into a bson.D, subdocuments should also be decoded as bson.D, even if a type map entry
			// has been registered.
			return ancestorType, nil
		}
	}

	rtype, err := reg.LookupTypeMapEntry(valueType)
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

		rtype, err = reg.LookupTypeMapEntry(lookupType)
		if err == nil {
			return rtype, nil
		}
	}

	return nil, err
}

func (eic emptyInterfaceCodec) decodeType(reg DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	rtype, err := eic.getEmptyInterfaceDecodeType(reg, vr.Type(), t)
	if err != nil {
		switch vr.Type() {
		case TypeNull:
			return reflect.Zero(t), vr.ReadNull()
		default:
			return emptyValue, err
		}
	}

	decoder, err := reg.LookupDecoder(rtype)
	if err != nil {
		return emptyValue, err
	}

	elem, err := decodeTypeOrValueWithInfo(decoder, reg, vr, rtype)
	if err != nil {
		return emptyValue, err
	}
	if elem.Type() != rtype {
		elem = elem.Convert(rtype)
	}

	if eic.decodeBinaryAsSlice && rtype == tBinary {
		binElem := elem.Interface().(Binary)
		if binElem.Subtype == TypeBinaryGeneric || binElem.Subtype == TypeBinaryBinaryOld {
			elem = reflect.ValueOf(binElem.Data)
		}
	}

	return elem, nil
}

// DecodeValue is the ValueDecoderFunc for interface{}.
func (eic emptyInterfaceCodec) DecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tEmpty {
		return ValueDecoderError{Name: "EmptyInterfaceDecodeValue", Types: []reflect.Type{tEmpty}, Received: val}
	}

	elem, err := eic.decodeType(reg, vr, val.Type())
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}
