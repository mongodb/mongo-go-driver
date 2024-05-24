// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"reflect"
)

// RegistryOpt is used to configure a Registry.
type RegistryOpt struct {
	typ reflect.Type
	fn  reflect.Value
}

// NewRegistryOpt creates a *RegistryOpt from a setter function.
// For example:
//
//	opt := NewRegistryOpt(func(c *Codec) {
//	    c.attr = value
//	})
//
// reg := NewRegistryBuilder().Build()
// reg.SetCodecOptions(opt)
//
// The "attr" field in the registered Codec can be set to "value".
func NewRegistryOpt[T any](fn func(T) error) *RegistryOpt {
	var zero [0]T
	return &RegistryOpt{
		typ: reflect.TypeOf(zero).Elem(),
		fn:  reflect.ValueOf(fn),
	}
}

// NilByteSliceAsEmpty causes the Encoder to marshal nil Go byte slices as empty BSON binary values
// instead of BSON null.
var NilByteSliceAsEmpty = NewRegistryOpt(func(c *byteSliceCodec) error {
	c.encodeNilAsEmpty = true
	return nil
})

// BinaryAsSlice causes the Decoder to unmarshal BSON binary field values that are the "Generic" or
// "Old" BSON binary subtype as a Go byte slice instead of a primitive.Binary.
var BinaryAsSlice = NewRegistryOpt(func(c *emptyInterfaceCodec) error {
	c.decodeBinaryAsSlice = true
	return nil
})

// DefaultDocumentM causes the Decoder to always unmarshal documents into the primitive.M type. This
// behavior is restricted to data typed as "interface{}" or "map[string]interface{}".
var DefaultDocumentM = NewRegistryOpt(func(c *emptyInterfaceCodec) error {
	c.defaultDocumentType = reflect.TypeOf(M{})
	return nil
})

// DefaultDocumentD causes the Decoder to always unmarshal documents into the primitive.D type. This
// behavior is restricted to data typed as "interface{}" or "map[string]interface{}".
var DefaultDocumentD = NewRegistryOpt(func(c *emptyInterfaceCodec) error {
	c.defaultDocumentType = reflect.TypeOf(D{})
	return nil
})

// NilMapAsEmpty causes the Encoder to marshal nil Go maps as empty BSON documents instead of BSON
// null.
var NilMapAsEmpty = NewRegistryOpt(func(c *mapCodec) error {
	c.encodeNilAsEmpty = true
	return nil
})

// StringifyMapKeysWithFmt causes the Encoder to convert Go map keys to BSON document field name
// strings using fmt.Sprint instead of the default string conversion logic.
var StringifyMapKeysWithFmt = NewRegistryOpt(func(c *mapCodec) error {
	c.encodeKeysWithStringer = true
	return nil
})

// ZeroMaps causes the Decoder to delete any existing values from Go maps in the destination value
// passed to Decode before unmarshaling BSON documents into them.
var ZeroMaps = NewRegistryOpt(func(c *mapCodec) error {
	c.decodeZerosMap = true
	return nil
})

// AllowTruncatingDoubles causes the Decoder to truncate the fractional part of BSON "double" values
// when attempting to unmarshal them into a Go integer (int, int8, int16, int32, or int64) struct
// field. The truncation logic does not apply to BSON "decimal128" values.
var AllowTruncatingDoubles = NewRegistryOpt(func(c *numCodec) error {
	c.truncate = true
	return nil
})

// IntMinSize causes the Encoder to marshal Go integer values (int, int8, int16, int32, int64, uint,
// uint8, uint16, uint32, or uint64) as the minimum BSON int size (either 32 or 64 bits) that can
// represent the integer value.
var IntMinSize = NewRegistryOpt(func(c *numCodec) error {
	c.minSize = true
	return nil
})

// NilSliceAsEmpty causes the Encoder to marshal nil Go slices as empty BSON arrays instead of BSON
// null.
var NilSliceAsEmpty = NewRegistryOpt(func(c *sliceCodec) error {
	c.encodeNilAsEmpty = true
	return nil
})

// DecodeObjectIDAsHex causes the Decoder to unmarshal BSON ObjectID as a hexadecimal string.
var DecodeObjectIDAsHex = NewRegistryOpt(func(c *stringCodec) error {
	c.decodeObjectIDAsHex = true
	return nil
})

// ErrorOnInlineDuplicates causes the Encoder to return an error if there is a duplicate field in
// the marshaled BSON when the "inline" struct tag option is set.
var ErrorOnInlineDuplicates = NewRegistryOpt(func(c *structCodec) error {
	c.overwriteDuplicatedInlinedFields = false
	return nil
})

// TODO(GODRIVER-2820): Update the description to remove the note about only examining exported
// TODO struct fields once the logic is updated to also inspect private struct fields.

// OmitZeroStruct causes the Encoder to consider the zero value for a struct (e.g. MyStruct{})
// as empty and omit it from the marshaled BSON when the "omitempty" struct tag option is set.
//
// Note that the Encoder only examines exported struct fields when determining if a struct is the
// zero value. It considers pointers to a zero struct value (e.g. &MyStruct{}) not empty.
var OmitZeroStruct = NewRegistryOpt(func(c *structCodec) error {
	c.encodeOmitDefaultStruct = true
	return nil
})

// UseJSONStructTags causes the Encoder and Decoder to fall back to using the "json" struct tag if
// a "bson" struct tag is not specified.
var UseJSONStructTags = NewRegistryOpt(func(c *structCodec) error {
	c.useJSONStructTags = true
	return nil
})

// ZeroStructs causes the Decoder to delete any existing values from Go structs in the destination
// value passed to Decode before unmarshaling BSON documents into them.
var ZeroStructs = NewRegistryOpt(func(c *structCodec) error {
	c.decodeZeroStruct = true
	return nil
})

// UseLocalTimeZone causes the Decoder to unmarshal time.Time values in the local timezone instead
// of the UTC timezone.
var UseLocalTimeZone = NewRegistryOpt(func(c *timeCodec) error {
	c.useLocalTimeZone = true
	return nil
})
