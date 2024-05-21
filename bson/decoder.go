// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// ErrDecodeToNil is the error returned when trying to decode to a nil value
var ErrDecodeToNil = errors.New("cannot Decode to nil value")

// This pool is used to keep the allocations of Decoders down. This is only used for the Marshal*
// methods and is not consumable from outside of this package. The Decoders retrieved from this pool
// must have both Reset and SetRegistry called on them.
var decPool = sync.Pool{
	New: func() interface{} {
		return new(Decoder)
	},
}

// A Decoder reads and decodes BSON documents from a stream. It reads from a ValueReader as
// the source of BSON data.
type Decoder struct {
	reg *Registry
	vr  ValueReader
}

// NewDecoder returns a new decoder that uses the default registry to read from vr.
func NewDecoder(vr ValueReader) *Decoder {
	return &Decoder{
		reg: NewRegistryBuilder().Build(),
		vr:  vr,
	}
}

// Decode reads the next BSON document from the stream and decodes it into the
// value pointed to by val.
//
// See [Unmarshal] for details about BSON unmarshaling behavior.
func (d *Decoder) Decode(val interface{}) error {
	if unmarshaler, ok := val.(Unmarshaler); ok {
		// TODO(skriptble): Reuse a []byte here and use the AppendDocumentBytes method.
		buf, err := copyDocumentToBytes(d.vr)
		if err != nil {
			return err
		}
		return unmarshaler.UnmarshalBSON(buf)
	}

	rval := reflect.ValueOf(val)
	switch rval.Kind() {
	case reflect.Ptr:
		if rval.IsNil() {
			return ErrDecodeToNil
		}
		rval = rval.Elem()
	case reflect.Map:
		if rval.IsNil() {
			return ErrDecodeToNil
		}
	default:
		return fmt.Errorf("argument to Decode must be a pointer or a map, but got %v", rval)
	}
	decoder, err := d.reg.LookupDecoder(rval.Type())
	if err != nil {
		return err
	}

	return decoder.DecodeValue(d.reg, d.vr, rval)
}

// Reset will reset the state of the decoder, using the same *DecodeContext used in
// the original construction but using vr for reading.
func (d *Decoder) Reset(vr ValueReader) {
	d.vr = vr
}

// SetRegistry replaces the current registry of the decoder with r.
func (d *Decoder) SetRegistry(r *Registry) {
	d.reg = r
}

// DefaultDocumentM causes the Decoder to always unmarshal documents into the primitive.M type. This
// behavior is restricted to data typed as "interface{}" or "map[string]interface{}".
func (d *Decoder) DefaultDocumentM() {
	t := reflect.TypeOf((*emptyInterfaceCodec)(nil))
	if v, ok := d.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*emptyInterfaceCodec).defaultDocumentType = reflect.TypeOf(M{})
		}
	}
}

// DefaultDocumentD causes the Decoder to always unmarshal documents into the primitive.D type. This
// behavior is restricted to data typed as "interface{}" or "map[string]interface{}".
func (d *Decoder) DefaultDocumentD() {
	t := reflect.TypeOf((*emptyInterfaceCodec)(nil))
	if v, ok := d.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*emptyInterfaceCodec).defaultDocumentType = reflect.TypeOf(D{})
		}
	}
}

// AllowTruncatingDoubles causes the Decoder to truncate the fractional part of BSON "double" values
// when attempting to unmarshal them into a Go integer (int, int8, int16, int32, or int64) struct
// field. The truncation logic does not apply to BSON "decimal128" values.
func (d *Decoder) AllowTruncatingDoubles() {
	t := reflect.TypeOf((*intCodec)(nil))
	if v, ok := d.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*intCodec).truncate = true
		}
	}
	// TODO floatCodec
}

// BinaryAsSlice causes the Decoder to unmarshal BSON binary field values that are the "Generic" or
// "Old" BSON binary subtype as a Go byte slice instead of a primitive.Binary.
func (d *Decoder) BinaryAsSlice() {
	t := reflect.TypeOf((*emptyInterfaceCodec)(nil))
	if v, ok := d.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*emptyInterfaceCodec).decodeBinaryAsSlice = true
		}
	}
}

// DecodeObjectIDAsHex causes the Decoder to unmarshal BSON ObjectID as a hexadecimal string.
func (d *Decoder) DecodeObjectIDAsHex() {
	t := reflect.TypeOf((*stringCodec)(nil))
	if v, ok := d.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*stringCodec).decodeObjectIDAsHex = true
		}
	}
}

// UseJSONStructTags causes the Decoder to fall back to using the "json" struct tag if a "bson"
// struct tag is not specified.
func (d *Decoder) UseJSONStructTags() {
	t := reflect.TypeOf((*structCodec)(nil))
	if v, ok := d.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*structCodec).useJSONStructTags = true
		}
	}
}

// UseLocalTimeZone causes the Decoder to unmarshal time.Time values in the local timezone instead
// of the UTC timezone.
func (d *Decoder) UseLocalTimeZone() {
	t := reflect.TypeOf((*timeCodec)(nil))
	if v, ok := d.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*timeCodec).useLocalTimeZone = true
		}
	}
}

// ZeroMaps causes the Decoder to delete any existing values from Go maps in the destination value
// passed to Decode before unmarshaling BSON documents into them.
func (d *Decoder) ZeroMaps() {
	t := reflect.TypeOf((*mapCodec)(nil))
	if v, ok := d.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*mapCodec).decodeZerosMap = true
		}
	}
}

// ZeroStructs causes the Decoder to delete any existing values from Go structs in the destination
// value passed to Decode before unmarshaling BSON documents into them.
func (d *Decoder) ZeroStructs() {
	t := reflect.TypeOf((*structCodec)(nil))
	if v, ok := d.reg.codecTypeMap[t]; ok && v != nil {
		for i := range v {
			v[i].(*structCodec).decodeZeroStruct = true
		}
	}
}
