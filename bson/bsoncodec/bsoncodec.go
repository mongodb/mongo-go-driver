// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncodec // import "go.mongodb.org/mongo-driver/bson/bsoncodec"

import (
	"fmt"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	emptyValue = reflect.Value{}
)

// Marshaler is an interface implemented by types that can marshal themselves
// into a BSON document represented as bytes. The bytes returned must be a valid
// BSON document if the error is nil.
//
// Deprecated: Use bson.Marshaler instead.
type Marshaler interface {
	MarshalBSON() ([]byte, error)
}

// ValueMarshaler is an interface implemented by types that can marshal
// themselves into a BSON value as bytes. The type must be the valid type for
// the bytes returned. The bytes and byte type together must be valid if the
// error is nil.
//
// Deprecated: Use bson.ValueMarshaler instead.
type ValueMarshaler interface {
	MarshalBSONValue() (bsontype.Type, []byte, error)
}

// Unmarshaler is an interface implemented by types that can unmarshal a BSON
// document representation of themselves. The BSON bytes can be assumed to be
// valid. UnmarshalBSON must copy the BSON bytes if it wishes to retain the data
// after returning.
//
// Deprecated: Use bson.Unmarshaler instead.
type Unmarshaler interface {
	UnmarshalBSON([]byte) error
}

// ValueUnmarshaler is an interface implemented by types that can unmarshal a
// BSON value representation of themselves. The BSON bytes and type can be
// assumed to be valid. UnmarshalBSONValue must copy the BSON value bytes if it
// wishes to retain the data after returning.
//
// Deprecated: Use bson.ValueUnmarshaler instead.
type ValueUnmarshaler interface {
	UnmarshalBSONValue(bsontype.Type, []byte) error
}

// ValueEncoderError is an error returned from a ValueEncoder when the provided value can't be
// encoded by the ValueEncoder.
type ValueEncoderError struct {
	Name     string
	Types    []reflect.Type
	Kinds    []reflect.Kind
	Received reflect.Value
}

func (vee ValueEncoderError) Error() string {
	typeKinds := make([]string, 0, len(vee.Types)+len(vee.Kinds))
	for _, t := range vee.Types {
		typeKinds = append(typeKinds, t.String())
	}
	for _, k := range vee.Kinds {
		if k == reflect.Map {
			typeKinds = append(typeKinds, "map[string]*")
			continue
		}
		typeKinds = append(typeKinds, k.String())
	}
	received := vee.Received.Kind().String()
	if vee.Received.IsValid() {
		received = vee.Received.Type().String()
	}
	return fmt.Sprintf("%s can only encode valid %s, but got %s", vee.Name, strings.Join(typeKinds, ", "), received)
}

// ValueDecoderError is an error returned from a ValueDecoder when the provided value can't be
// decoded by the ValueDecoder.
type ValueDecoderError struct {
	Name     string
	Types    []reflect.Type
	Kinds    []reflect.Kind
	Received reflect.Value
}

func (vde ValueDecoderError) Error() string {
	typeKinds := make([]string, 0, len(vde.Types)+len(vde.Kinds))
	for _, t := range vde.Types {
		typeKinds = append(typeKinds, t.String())
	}
	for _, k := range vde.Kinds {
		if k == reflect.Map {
			typeKinds = append(typeKinds, "map[string]*")
			continue
		}
		typeKinds = append(typeKinds, k.String())
	}
	received := vde.Received.Kind().String()
	if vde.Received.IsValid() {
		received = vde.Received.Type().String()
	}
	return fmt.Sprintf("%s can only decode valid and settable %s, but got %s", vde.Name, strings.Join(typeKinds, ", "), received)
}

// EncodeContext is the contextual information required for a Codec to encode a
// value.
type EncodeContext struct {
	*Registry

	// MinSize, if true, instructs encoders to marshal Go integer values (int, int8, int16,
	// int32, or int64) as the minimum BSON int size (either 32-bit or 64-bit) that can represent
	// the integer value.
	//
	// Deprecated: Use IntMinSize instead.
	MinSize bool

	// AllowUnexportedFields, if true, instructs encoders to marshal values from unexported struct
	// fields.
	AllowUnexportedFields bool

	// ErrorOnInlineDuplicates, if true, instructs encoders to return an error if there is a
	// duplicate field in the marshaled BSON when the "inline" struct tag option is set.
	ErrorOnInlineDuplicates bool

	// IntMinSize, if true, instructs encoders to marshal Go integer values (int, int8, int16,
	// int32, or int64) as the minimum BSON int size (either 32-bit or 64-bit) that can represent
	// the integer value.
	IntMinSize bool

	// MapKeysWithStringer, if true, instructs encoders to convert Go map keys to BSON document
	// field name strings using fmt.Sprintf() instead of the default string conversion logic.
	MapKeysWithStringer bool

	// NilMapAsEmpty, if true, instructs encoders to marshal nil Go maps as empty BSON documents
	// instead of BSON null.
	NilMapAsEmpty bool

	// NilSliceAsEmpty, if true, instructs encoders to marshal nil Go slices as empty BSON arrays
	// instead of BSON null.
	NilSliceAsEmpty bool

	// NilByteSliceAsEmpty, if true, instructs encoders to marshal nil Go byte slices as empty BSON
	// binary values instead of BSON null.
	NilByteSliceAsEmpty bool

	// OmitDefaultStruct, if true, instructs encoders to consider the zero value for a struct (e.g.
	// MyStruct{}) as empty and omit it from the marshaled BSON when the "omitempty" struct tag
	// option is set.
	OmitDefaultStruct bool

	// UseJSONStructTags, if true, instructs encoders to fall back to using the "json" struct tag if
	// a "bson" struct tag is not specified.
	UseJSONStructTags bool
}

// DecodeContext is the contextual information required for a Codec to decode a
// value.
type DecodeContext struct {
	*Registry

	// Truncate allows truncating the fractional part of BSON floating point values when decoding
	// them into a Go integer value. The default is false, which returns an error when attempting to
	// decode BSON floating point values with a fractional part into a Go integer.
	//
	// Deprecated: Use AllowTruncatingFloats instead.
	Truncate bool

	// Ancestor is the type of a containing document. This is mainly used to determine what type
	// should be used when decoding an embedded document into an empty interface. For example, if
	// Ancestor is a bson.M, BSON embedded document values being decoded into an empty interface
	// will be decoded into a bson.M.
	//
	// Deprecated: Use DefaultDocumentM or DefaultDocumentD instead.
	Ancestor reflect.Type

	// defaultDocumentType specifies the Go type to decode top-level and nested BSON documents into. In particular, the
	// usage for this field is restricted to data typed as "interface{}" or "map[string]interface{}". If DocumentType is
	// set to a type that a BSON document cannot be unmarshaled into (e.g. "string"), unmarshalling will result in an
	// error. DocumentType overrides the Ancestor field.
	defaultDocumentType reflect.Type

	// AllowTruncatingDoubles, if true, instructs decoders to truncate the fractional part of BSON
	// "double" values when attempting to unmarshal them into a Go integer struct field. The
	// truncation logic does not apply to BSON "decimal128" values.
	AllowTruncatingDoubles bool

	// AllowUnexportedFields, if true, instructs decoders to unmarshal values into unexported struct fields.
	AllowUnexportedFields bool

	// BinaryAsSlice, if true, instructs decoders to unmarshal BSON binary field values that are the
	// "Generic" or "Old" BSON binary subtype as a Go byte slice instead of a primitive.Binary.
	BinaryAsSlice bool

	// UseJSONStructTags, if true, instructs decoders to fall back to using the "json" struct tag if
	// a "bson" struct tag is not specified.
	UseJSONStructTags bool

	// ZeroMaps, if true, instructs decoders to delete any existing values from Go maps in the
	// destination value passed to Decode before unmarshaling BSON documents into them.
	ZeroMaps bool

	// ZeroStructs, if true, instructs decoders to delete any existing values from Go structs in the
	// destination value passed to Decode before unmarshaling BSON documents into them.
	ZeroStructs bool
}

// DefaultDocumentM will decode empty documents using the primitive.M type. This behavior is restricted to data typed as
// "interface{}" or "map[string]interface{}".
func (dc *DecodeContext) DefaultDocumentM() {
	dc.defaultDocumentType = reflect.TypeOf(primitive.M{})
}

// DefaultDocumentD will decode empty documents using the primitive.D type. This behavior is restricted to data typed as
// "interface{}" or "map[string]interface{}".
func (dc *DecodeContext) DefaultDocumentD() {
	dc.defaultDocumentType = reflect.TypeOf(primitive.D{})
}

// ValueCodec is an interface for encoding and decoding a reflect.Value.
// values.
//
// Deprecated: Use ValueEncoder and ValueDecoder instead.
type ValueCodec interface {
	ValueEncoder
	ValueDecoder
}

// ValueEncoder is the interface implemented by types that can handle the encoding of a value.
type ValueEncoder interface {
	EncodeValue(EncodeContext, bsonrw.ValueWriter, reflect.Value) error
}

// ValueEncoderFunc is an adapter function that allows a function with the correct signature to be
// used as a ValueEncoder.
type ValueEncoderFunc func(EncodeContext, bsonrw.ValueWriter, reflect.Value) error

// EncodeValue implements the ValueEncoder interface.
func (fn ValueEncoderFunc) EncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	return fn(ec, vw, val)
}

// ValueDecoder is the interface implemented by types that can handle the decoding of a value.
type ValueDecoder interface {
	DecodeValue(DecodeContext, bsonrw.ValueReader, reflect.Value) error
}

// ValueDecoderFunc is an adapter function that allows a function with the correct signature to be
// used as a ValueDecoder.
type ValueDecoderFunc func(DecodeContext, bsonrw.ValueReader, reflect.Value) error

// DecodeValue implements the ValueDecoder interface.
func (fn ValueDecoderFunc) DecodeValue(dc DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	return fn(dc, vr, val)
}

// typeDecoder is the interface implemented by types that can handle the decoding of a value given its type.
type typeDecoder interface {
	decodeType(DecodeContext, bsonrw.ValueReader, reflect.Type) (reflect.Value, error)
}

// typeDecoderFunc is an adapter function that allows a function with the correct signature to be used as a typeDecoder.
type typeDecoderFunc func(DecodeContext, bsonrw.ValueReader, reflect.Type) (reflect.Value, error)

func (fn typeDecoderFunc) decodeType(dc DecodeContext, vr bsonrw.ValueReader, t reflect.Type) (reflect.Value, error) {
	return fn(dc, vr, t)
}

// decodeAdapter allows two functions with the correct signatures to be used as both a ValueDecoder and typeDecoder.
type decodeAdapter struct {
	ValueDecoderFunc
	typeDecoderFunc
}

var _ ValueDecoder = decodeAdapter{}
var _ typeDecoder = decodeAdapter{}

// decodeTypeOrValue calls decoder.decodeType is decoder is a typeDecoder. Otherwise, it allocates a new element of type
// t and calls decoder.DecodeValue on it.
func decodeTypeOrValue(decoder ValueDecoder, dc DecodeContext, vr bsonrw.ValueReader, t reflect.Type) (reflect.Value, error) {
	td, _ := decoder.(typeDecoder)
	return decodeTypeOrValueWithInfo(decoder, td, dc, vr, t, true)
}

func decodeTypeOrValueWithInfo(vd ValueDecoder, td typeDecoder, dc DecodeContext, vr bsonrw.ValueReader, t reflect.Type, convert bool) (reflect.Value, error) {
	if td != nil {
		val, err := td.decodeType(dc, vr, t)
		if err == nil && convert && val.Type() != t {
			// This conversion step is necessary for slices and maps. If a user declares variables like:
			//
			// type myBool bool
			// var m map[string]myBool
			//
			// and tries to decode BSON bytes into the map, the decoding will fail if this conversion is not present
			// because we'll try to assign a value of type bool to one of type myBool.
			val = val.Convert(t)
		}
		return val, err
	}

	val := reflect.New(t).Elem()
	err := vd.DecodeValue(dc, vr, val)
	return val, err
}

// CodecZeroer is the interface implemented by Codecs that can also determine if
// a value of the type that would be encoded is zero.
//
// Deprecated: Defining custom rules for the zero/empty value will not be supported in Go Driver
// 2.0. Users who want to omit empty complex values should use a pointer field and set the value to
// nil instead.
type CodecZeroer interface {
	IsTypeZero(interface{}) bool
}
