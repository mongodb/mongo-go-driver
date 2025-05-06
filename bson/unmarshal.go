// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"fmt"
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
// pointed to by val. If val is nil or not a pointer, Unmarshal returns an
// error.
//
// When unmarshaling BSON, if the BSON value is null and the Go value is a
// pointer, the pointer is set to nil without calling UnmarshalBSONValue.
func Unmarshal(data []byte, val interface{}) error {
	vr := getDocumentReader(bytes.NewReader(data))
	defer putDocumentReader(vr)

	if l, err := vr.peekLength(); err != nil {
		return err
	} else if int(l) != len(data) {
		return fmt.Errorf("invalid document length")
	}
	return unmarshalFromReader(DecodeContext{Registry: defaultRegistry}, vr, val)
}

// UnmarshalValue parses the BSON value of type t with bson.NewRegistry() and
// stores the result in the value pointed to by val. If val is nil or not a pointer,
// UnmarshalValue returns an error.
func UnmarshalValue(t Type, data []byte, val interface{}) error {
	vr := newValueReader(t, bytes.NewReader(data))
	return unmarshalFromReader(DecodeContext{Registry: defaultRegistry}, vr, val)
}

// UnmarshalExtJSON parses the extended JSON-encoded data and stores the result
// in the value pointed to by val. If val is nil or not a pointer, UnmarshalExtJSON
// returns an error.
//
// If canonicalOnly is true, UnmarshalExtJSON returns an error if the Extended
// JSON was not marshaled in canonical mode.
//
// For more information about Extended JSON, see
// https://www.mongodb.com/docs/manual/reference/mongodb-extended-json/
func UnmarshalExtJSON(data []byte, canonicalOnly bool, val interface{}) error {
	ejvr, err := NewExtJSONValueReader(bytes.NewReader(data), canonicalOnly)
	if err != nil {
		return err
	}

	return unmarshalFromReader(DecodeContext{Registry: defaultRegistry}, ejvr, val)
}

func unmarshalFromReader(dc DecodeContext, vr ValueReader, val interface{}) error {
	dec := decPool.Get().(*Decoder)
	defer decPool.Put(dec)

	dec.Reset(vr)
	dec.dc = dc

	return dec.Decode(val)
}
