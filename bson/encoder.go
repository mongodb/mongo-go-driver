// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"reflect"
	"sync"
)

// This pool is used to keep the allocations of Encoders down. This is only used for the Marshal*
// methods and is not consumable from outside of this package. The Encoders retrieved from this pool
// must have both Reset and SetRegistry called on them.
var encPool = sync.Pool{
	New: func() interface{} {
		return new(Encoder)
	},
}

// An Encoder writes a serialization format to an output stream. It writes to a ValueWriter
// as the destination of BSON data.
type Encoder struct {
	reg *Registry
	vw  ValueWriter
}

// NewEncoder returns a new encoder that uses the DefaultRegistry to write to vw.
func NewEncoder(vw ValueWriter) *Encoder {
	return &Encoder{
		reg: DefaultRegistry,
		vw:  vw,
	}
}

// Encode writes the BSON encoding of val to the stream.
//
// See [Marshal] for details about BSON marshaling behavior.
func (e *Encoder) Encode(val interface{}) error {
	if marshaler, ok := val.(Marshaler); ok {
		// TODO(skriptble): Should we have a MarshalAppender interface so that we can have []byte reuse?
		buf, err := marshaler.MarshalBSON()
		if err != nil {
			return err
		}
		return copyDocumentFromBytes(e.vw, buf)
	}

	encoder, err := e.reg.LookupEncoder(reflect.TypeOf(val))
	if err != nil {
		return err
	}

	return encoder.EncodeValue(e.reg, e.vw, reflect.ValueOf(val))
}

// Reset will reset the state of the Encoder, using the same *EncodeContext used in
// the original construction but using vw.
func (e *Encoder) Reset(vw ValueWriter) {
	e.vw = vw
}

// SetRegistry replaces the current registry of the Encoder with r.
func (e *Encoder) SetRegistry(r *Registry) {
	e.reg = r
}

// ErrorOnInlineDuplicates causes the Encoder to return an error if there is a duplicate field in
// the marshaled BSON when the "inline" struct tag option is set.
func (e *Encoder) ErrorOnInlineDuplicates() {
	t := reflect.TypeOf((*structCodec)(nil))
	if v, ok := e.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*structCodec).overwriteDuplicatedInlinedFields = false
		}
	}
}

// IntMinSize causes the Encoder to marshal Go integer values (int, int8, int16, int32, int64, uint,
// uint8, uint16, uint32, or uint64) as the minimum BSON int size (either 32 or 64 bits) that can
// represent the integer value.
func (e *Encoder) IntMinSize() {
	// if v, ok := e.reg.kindEncoders.Load(reflect.Int); ok {
	// 	if enc, ok := v.(*intCodec); ok {
	// 		enc.encodeToMinSize = true
	// 	}
	// }
	// if v, ok := e.reg.kindEncoders.Load(reflect.Uint); ok {
	// 	if enc, ok := v.(*uintCodec); ok {
	// 		enc.encodeToMinSize = true
	// 	}
	// }
	t := reflect.TypeOf((*intCodec)(nil))
	if v, ok := e.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*intCodec).encodeToMinSize = true
		}
	}
}

// StringifyMapKeysWithFmt causes the Encoder to convert Go map keys to BSON document field name
// strings using fmt.Sprint instead of the default string conversion logic.
func (e *Encoder) StringifyMapKeysWithFmt() {
	t := reflect.TypeOf((*mapCodec)(nil))
	if v, ok := e.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*mapCodec).encodeKeysWithStringer = true
		}
	}
}

// NilMapAsEmpty causes the Encoder to marshal nil Go maps as empty BSON documents instead of BSON
// null.
func (e *Encoder) NilMapAsEmpty() {
	t := reflect.TypeOf((*mapCodec)(nil))
	if v, ok := e.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*mapCodec).encodeNilAsEmpty = true
		}
	}
}

// NilSliceAsEmpty causes the Encoder to marshal nil Go slices as empty BSON arrays instead of BSON
// null.
func (e *Encoder) NilSliceAsEmpty() {
	t := reflect.TypeOf((*sliceCodec)(nil))
	if v, ok := e.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*sliceCodec).encodeNilAsEmpty = true
		}
	}
}

// NilByteSliceAsEmpty causes the Encoder to marshal nil Go byte slices as empty BSON binary values
// instead of BSON null.
func (e *Encoder) NilByteSliceAsEmpty() {
	// if v, ok := e.reg.typeEncoders.Load(tByteSlice); ok {
	// 	if enc, ok := v.(*byteSliceCodec); ok {
	// 		enc.encodeNilAsEmpty = true
	// 	}
	// }
	t := reflect.TypeOf((*byteSliceCodec)(nil))
	if v, ok := e.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*byteSliceCodec).encodeNilAsEmpty = true
		}
	}
}

// TODO(GODRIVER-2820): Update the description to remove the note about only examining exported
// TODO struct fields once the logic is updated to also inspect private struct fields.

// OmitZeroStruct causes the Encoder to consider the zero value for a struct (e.g. MyStruct{})
// as empty and omit it from the marshaled BSON when the "omitempty" struct tag option is set.
//
// Note that the Encoder only examines exported struct fields when determining if a struct is the
// zero value. It considers pointers to a zero struct value (e.g. &MyStruct{}) not empty.
func (e *Encoder) OmitZeroStruct() {
	t := reflect.TypeOf((*structCodec)(nil))
	if v, ok := e.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*structCodec).encodeOmitDefaultStruct = true
		}
	}
}

// UseJSONStructTags causes the Encoder to fall back to using the "json" struct tag if a "bson"
// struct tag is not specified.
func (e *Encoder) UseJSONStructTags() {
	t := reflect.TypeOf((*structCodec)(nil))
	if v, ok := e.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*structCodec).useJSONStructTags = true
		}
	}
}
