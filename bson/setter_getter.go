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

// Setter interface: a value implementing the bson.Setter interface will receive the BSON
// value via the SetBSON method during unmarshaling, and the object
// itself will not be changed as usual.
//
// If setting the value works, the method should return nil or alternatively
// mgocompat.ErrSetZero to set the respective field to its zero value (nil for
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
type Setter interface {
	SetBSON(raw RawValue) error
}

// Getter interface: a value implementing the bson.Getter interface will have its GetBSON
// method called when the given value has to be marshalled, and the result
// of this method will be marshaled in place of the actual object.
//
// If GetBSON returns return a non-nil error, the marshalling procedure
// will stop and error out with the provided value.
type Getter interface {
	GetBSON() (interface{}, error)
}

// SetterDecodeValue is the ValueDecoderFunc for Setter types.
func SetterDecodeValue(_ DecoderRegistry, vr ValueReader, val reflect.Value) error {
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

	t, src, err := CopyValueToBytes(vr)
	if err != nil {
		return err
	}

	m, ok := val.Interface().(Setter)
	if !ok {
		return ValueDecoderError{Name: "SetterDecodeValue", Types: []reflect.Type{tSetter}, Received: val}
	}
	if err := m.SetBSON(RawValue{Type: t, Value: src}); err != nil {
		if !errors.Is(err, ErrSetZero) {
			return err
		}
		val.Set(reflect.Zero(val.Type()))
	}
	return nil
}

// GetterEncodeValue is the ValueEncoderFunc for Getter types.
func GetterEncodeValue(reg EncoderRegistry, vw ValueWriter, val reflect.Value) error {
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

	m, ok := val.Interface().(Getter)
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
	encoder, err := reg.LookupEncoder(vv.Type())
	if err != nil {
		return err
	}
	return encoder.EncodeValue(reg, vw, vv)
}
