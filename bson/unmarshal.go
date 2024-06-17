// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
)

// Unmarshaler is the interface implemented by types that can unmarshal a BSON
// document representation of themselves. The input can be assumed to be a valid
// encoding of a BSON document. UnmarshalBSON must copy the JSON data if it
// wishes to retain the data after returning.
//
// Unmarshaler is only used to unmarshal full BSON documents. To create custom
// BSON unmarshaling behavior for individual values in a BSON document,
// implement the ValueUnmarshaler interface instead.
type Unmarshaler interface {
	UnmarshalBSON([]byte) error
}

// ValueUnmarshaler is the interface implemented by types that can unmarshal a
// BSON value representation of themselves. The input can be assumed to be a
// valid encoding of a BSON value. UnmarshalBSONValue must copy the BSON value
// bytes if it wishes to retain the data after returning.
//
// ValueUnmarshaler is only used to unmarshal individual values in a BSON
// document. To create custom BSON unmarshaling behavior for an entire BSON
// document, implement the Unmarshaler interface instead.
type ValueUnmarshaler interface {
	UnmarshalBSONValue(typ byte, data []byte) error
}

// Unmarshal parses the BSON-encoded data and stores the result in the value
// pointed to by val. If val is nil or not a pointer, Unmarshal returns
// InvalidUnmarshalError.
func Unmarshal(data []byte, val interface{}) error {
	return UnmarshalWithRegistry(DefaultRegistry, data, val)
}

// UnmarshalWithRegistry parses the BSON-encoded data using Registry r and
// stores the result in the value pointed to by val. If val is nil or not
// a pointer, UnmarshalWithRegistry returns InvalidUnmarshalError.
//
// Deprecated: Use [NewDecoder] and specify the Registry by calling [Decoder.SetRegistry] instead:
//
//	dec, err := bson.NewDecoder(NewBSONDocumentReader(data))
//	if err != nil {
//		panic(err)
//	}
//	dec.SetRegistry(reg)
//
// See [Decoder] for more examples.
func UnmarshalWithRegistry(r *Registry, data []byte, val interface{}) error {
	vr := NewValueReader(data)
	return unmarshalFromReader(DecodeContext{Registry: r}, vr, val)
}

// UnmarshalWithContext parses the BSON-encoded data using DecodeContext dc and
// stores the result in the value pointed to by val. If val is nil or not
// a pointer, UnmarshalWithRegistry returns InvalidUnmarshalError.
//
// Deprecated: Use [NewDecoder] and use the Decoder configuration methods to set the desired unmarshal
// behavior instead:
//
//	dec, err := bson.NewDecoder(NewBSONDocumentReader(data))
//	if err != nil {
//		panic(err)
//	}
//	dec.DefaultDocumentM()
//
// See [Decoder] for more examples.
func UnmarshalWithContext(dc DecodeContext, data []byte, val interface{}) error {
	vr := NewValueReader(data)
	return unmarshalFromReader(dc, vr, val)
}

// UnmarshalValue parses the BSON value of type t with bson.DefaultRegistry and
// stores the result in the value pointed to by val. If val is nil or not a pointer,
// UnmarshalValue returns an error.
func UnmarshalValue(t Type, data []byte, val interface{}) error {
	return UnmarshalValueWithRegistry(DefaultRegistry, t, data, val)
}

// UnmarshalValueWithRegistry parses the BSON value of type t with registry r and
// stores the result in the value pointed to by val. If val is nil or not a pointer,
// UnmarshalValue returns an error.
//
// Deprecated: Using a custom registry to unmarshal individual BSON values will not be supported in
// Go Driver 2.0.
func UnmarshalValueWithRegistry(r *Registry, t Type, data []byte, val interface{}) error {
	vr := NewBSONValueReader(t, data)
	return unmarshalFromReader(DecodeContext{Registry: r}, vr, val)
}

// UnmarshalExtJSON parses the extended JSON-encoded data and stores the result
// in the value pointed to by val. If val is nil or not a pointer, Unmarshal
// returns InvalidUnmarshalError.
func UnmarshalExtJSON(data []byte, canonical bool, val interface{}) error {
	return UnmarshalExtJSONWithRegistry(DefaultRegistry, data, canonical, val)
}

// UnmarshalExtJSONWithRegistry parses the extended JSON-encoded data using
// Registry r and stores the result in the value pointed to by val. If val is
// nil or not a pointer, UnmarshalWithRegistry returns InvalidUnmarshalError.
//
// Deprecated: Use [NewDecoder] and specify the Registry by calling [Decoder.SetRegistry] instead:
//
//	vr, err := NewExtJSONValueReader(bytes.NewReader(data), true)
//	if err != nil {
//		panic(err)
//	}
//	dec, err := bson.NewDecoder(vr)
//	if err != nil {
//		panic(err)
//	}
//	dec.SetRegistry(reg)
//
// See [Decoder] for more examples.
func UnmarshalExtJSONWithRegistry(r *Registry, data []byte, canonical bool, val interface{}) error {
	ejvr, err := NewExtJSONValueReader(bytes.NewReader(data), canonical)
	if err != nil {
		return err
	}

	return unmarshalFromReader(DecodeContext{Registry: r}, ejvr, val)
}

// UnmarshalExtJSONWithContext parses the extended JSON-encoded data using
// DecodeContext dc and stores the result in the value pointed to by val. If val is
// nil or not a pointer, UnmarshalWithRegistry returns InvalidUnmarshalError.
//
// Deprecated: Use [NewDecoder] and use the Decoder configuration methods to set the desired unmarshal
// behavior instead:
//
//	vr, err := NewExtJSONValueReader(bytes.NewReader(data), true)
//	if err != nil {
//		panic(err)
//	}
//	dec, err := bson.NewDecoder(vr)
//	if err != nil {
//		panic(err)
//	}
//	dec.DefaultDocumentM()
//
// See [Decoder] for more examples.
func UnmarshalExtJSONWithContext(dc DecodeContext, data []byte, canonical bool, val interface{}) error {
	ejvr, err := NewExtJSONValueReader(bytes.NewReader(data), canonical)
	if err != nil {
		return err
	}

	return unmarshalFromReader(dc, ejvr, val)
}

func unmarshalFromReader(dc DecodeContext, vr ValueReader, val interface{}) error {
	dec := decPool.Get().(*Decoder)
	defer decPool.Put(dec)

	dec.Reset(vr)
	dec.dc = dc

	return dec.Decode(val)
}
