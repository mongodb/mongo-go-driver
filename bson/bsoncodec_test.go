// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"fmt"
	"reflect"
	"testing"
)

func ExampleValueEncoder() {
	var _ ValueEncoderFunc = func(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
		if val.Kind() != reflect.String {
			return ValueEncoderError{Name: "StringEncodeValue", Kinds: []reflect.Kind{reflect.String}, Received: val}
		}

		return vw.WriteString(val.String())
	}
}

func ExampleValueDecoder() {
	var _ ValueDecoderFunc = func(_ DecodeContext, vr ValueReader, val reflect.Value) error {
		if !val.CanSet() || val.Kind() != reflect.String {
			return ValueDecoderError{Name: "StringDecodeValue", Kinds: []reflect.Kind{reflect.String}, Received: val}
		}

		if vr.Type() != TypeString {
			return fmt.Errorf("cannot decode %v into a string type", vr.Type())
		}

		str, err := vr.ReadString()
		if err != nil {
			return err
		}
		val.SetString(str)
		return nil
	}
}

type llCodec struct {
	t         *testing.T
	decodeval interface{}
	encodeval interface{}
	err       error
}

func (llc *llCodec) EncodeValue(_ EncodeContext, _ ValueWriter, i interface{}) error {
	if llc.err != nil {
		return llc.err
	}

	llc.encodeval = i
	return nil
}

func (llc *llCodec) DecodeValue(_ DecodeContext, _ ValueReader, val reflect.Value) error {
	if llc.err != nil {
		return llc.err
	}

	if !reflect.TypeOf(llc.decodeval).AssignableTo(val.Type()) {
		llc.t.Errorf("decodeval must be assignable to val provided to DecodeValue, but is not. decodeval %T; val %T", llc.decodeval, val)
		return nil
	}

	val.Set(reflect.ValueOf(llc.decodeval))
	return nil
}
