// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"errors"
	"reflect"
)

var (
	// ErrMgoSetZero may be returned from a SetBSON method to have the value set to its respective zero value.
	ErrMgoSetZero = errors.New("set to zero")

	tInt            = reflect.TypeOf(int(0))
	tM              = reflect.TypeOf(M{})
	tInterfaceSlice = reflect.TypeOf([]interface{}{})
	tGetter         = reflect.TypeOf((*getter)(nil)).Elem()
	tSetter         = reflect.TypeOf((*setter)(nil)).Elem()
)

// NewMgoRegistry creates a new bson.Registry configured with the default encoders and decoders.
func NewMgoRegistry() *Registry {
	mapCodec := &mapCodec{
		decodeZerosMap:         true,
		encodeNilAsEmpty:       true,
		encodeKeysWithStringer: true,
	}
	structCodec := &structCodec{
		inlineMapEncoder:        mapCodec,
		decodeZeroStruct:        true,
		encodeOmitDefaultStruct: true,
		allowUnexportedFields:   true,
	}
	uintCodec := &uintCodec{encodeToMinSize: true}

	reg := NewRegistry()
	reg.RegisterTypeDecoder(tEmpty, &emptyInterfaceCodec{decodeBinaryAsSlice: true})
	reg.RegisterKindDecoder(reflect.String, ValueDecoderFunc(mgoStringDecodeValue))
	reg.RegisterKindDecoder(reflect.Struct, structCodec)
	reg.RegisterKindDecoder(reflect.Map, mapCodec)
	reg.RegisterTypeEncoder(tByteSlice, &byteSliceCodec{encodeNilAsEmpty: true})
	reg.RegisterKindEncoder(reflect.Struct, structCodec)
	reg.RegisterKindEncoder(reflect.Slice, &sliceCodec{encodeNilAsEmpty: true})
	reg.RegisterKindEncoder(reflect.Map, mapCodec)
	reg.RegisterKindEncoder(reflect.Uint, uintCodec)
	reg.RegisterKindEncoder(reflect.Uint8, uintCodec)
	reg.RegisterKindEncoder(reflect.Uint16, uintCodec)
	reg.RegisterKindEncoder(reflect.Uint32, uintCodec)
	reg.RegisterKindEncoder(reflect.Uint64, uintCodec)
	reg.RegisterTypeMapEntry(TypeInt32, tInt)
	reg.RegisterTypeMapEntry(TypeDateTime, tTime)
	reg.RegisterTypeMapEntry(TypeArray, tInterfaceSlice)
	reg.RegisterTypeMapEntry(Type(0), tM)
	reg.RegisterTypeMapEntry(TypeEmbeddedDocument, tM)
	reg.RegisterInterfaceEncoder(tGetter, ValueEncoderFunc(getterEncodeValue))
	reg.RegisterInterfaceDecoder(tSetter, ValueDecoderFunc(setterDecodeValue))
	return reg
}

// NewRespectNilValuesMgoRegistry creates a new bson.Registry configured to behave like mgo/bson
// with RespectNilValues set to true.
func NewRespectNilValuesMgoRegistry() *Registry {
	mapCodec := &mapCodec{
		decodeZerosMap: true,
	}

	reg := NewMgoRegistry()
	reg.RegisterKindDecoder(reflect.Map, mapCodec)
	reg.RegisterTypeEncoder(tByteSlice, &byteSliceCodec{encodeNilAsEmpty: false})
	reg.RegisterKindEncoder(reflect.Slice, &sliceCodec{})
	reg.RegisterKindEncoder(reflect.Map, mapCodec)
	return reg
}

func mgoStringDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if val.Kind() != reflect.String {
		return ValueDecoderError{
			Name:     "StringDecodeValue",
			Kinds:    []reflect.Kind{reflect.String},
			Received: reflect.Zero(val.Type()),
		}
	}

	if vr.Type() == TypeObjectID {
		oid, err := vr.ReadObjectID()
		if err != nil {
			return err
		}
		if dc.objectIDAsHexString {
			val.SetString(oid.Hex())
		} else {
			val.SetString(string(oid[:]))
		}
		return nil
	}
	return (&stringCodec{}).DecodeValue(dc, vr, val)
}

// setter interface: a value implementing the bson.Setter interface will receive the BSON
// value via the SetBSON method during unmarshaling, and the object
// itself will not be changed as usual.
//
// If setting the value works, the method should return nil or alternatively
// ErrMgoSetZero to set the respective field to its zero value (nil for
// pointer types). If SetBSON returns a non-nil error, the unmarshalling
// procedure will stop and error out with the provided value.
//
// This interface is generally useful in pointer receivers, since the method
// will want to change the receiver. A type field that implements the Setter
// interface doesn't have to be a pointer, though.
//
// For example:
//
//	type MyString string
//
//	func (s *MyString) SetBSON(raw bson.RawValue) error {
//	    return raw.Unmarshal(s)
//	}
type setter interface {
	SetBSON(raw RawValue) error
}

// getter interface: a value implementing the bson.Getter interface will have its GetBSON
// method called when the given value has to be marshalled, and the result
// of this method will be marshaled in place of the actual object.
//
// If GetBSON returns return a non-nil error, the marshalling procedure
// will stop and error out with the provided value.
type getter interface {
	GetBSON() (interface{}, error)
}

// setterDecodeValue is the ValueDecoderFunc for Setter types.
func setterDecodeValue(_ DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.IsValid() || (!val.Type().Implements(tSetter) && !reflect.PtrTo(val.Type()).Implements(tSetter)) {
		return ValueDecoderError{Name: "SetterDecodeValue", Types: []reflect.Type{tSetter}, Received: val}
	}

	if val.Kind() == reflect.Ptr && val.IsNil() {
		if !val.CanSet() {
			return ValueDecoderError{Name: "SetterDecodeValue", Types: []reflect.Type{tSetter}, Received: val}
		}
		val.Set(reflect.New(val.Type().Elem()))
	}

	if !val.Type().Implements(tSetter) {
		if !val.CanAddr() {
			return ValueDecoderError{Name: "ValueUnmarshalerDecodeValue", Types: []reflect.Type{tSetter}, Received: val}
		}
		val = val.Addr() // If the type doesn't implement the interface, a pointer to it must.
	}

	t, src, err := copyValueToBytes(vr)
	if err != nil {
		return err
	}

	m, ok := val.Interface().(setter)
	if !ok {
		return ValueDecoderError{Name: "SetterDecodeValue", Types: []reflect.Type{tSetter}, Received: val}
	}
	if err := m.SetBSON(RawValue{Type: t, Value: src}); err != nil {
		if !errors.Is(err, ErrMgoSetZero) {
			return err
		}
		val.Set(reflect.Zero(val.Type()))
	}
	return nil
}

// getterEncodeValue is the ValueEncoderFunc for Getter types.
func getterEncodeValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
	// Either val or a pointer to val must implement Getter
	switch {
	case !val.IsValid():
		return ValueEncoderError{Name: "GetterEncodeValue", Types: []reflect.Type{tGetter}, Received: val}
	case val.Type().Implements(tGetter):
		// If Getter is implemented on a concrete type, make sure that val isn't a nil pointer
		if isImplementationNil(val, tGetter) {
			return vw.WriteNull()
		}
	case reflect.PtrTo(val.Type()).Implements(tGetter) && val.CanAddr():
		val = val.Addr()
	default:
		return ValueEncoderError{Name: "GetterEncodeValue", Types: []reflect.Type{tGetter}, Received: val}
	}

	m, ok := val.Interface().(getter)
	if !ok {
		return vw.WriteNull()
	}
	x, err := m.GetBSON()
	if err != nil {
		return err
	}
	if x == nil {
		return vw.WriteNull()
	}
	vv := reflect.ValueOf(x)
	encoder, err := ec.Registry.LookupEncoder(vv.Type())
	if err != nil {
		return err
	}
	return encoder.EncodeValue(ec, vw, vv)
}
