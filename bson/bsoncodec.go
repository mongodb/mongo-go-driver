// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"fmt"
	"reflect"
	"strings"
)

var (
	emptyValue = reflect.Value{}
)

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

// DecodeContext is the contextual information required for a Codec to decode a
// value.
type DecodeContext struct {
	*Registry

	// defaultDocumentType specifies the Go type to decode top-level and nested BSON documents into. In particular, the
	// usage for this field is restricted to data typed as "interface{}" or "map[string]interface{}". If DocumentType is
	// set to a type that a BSON document cannot be unmarshaled into (e.g. "string"), unmarshalling will result in an
	// error. DocumentType overrides the Ancestor field.
	defaultDocumentType reflect.Type

	// truncate, if true, instructs decoders to to truncate the fractional part of BSON "double"
	// values when attempting to unmarshal them into a Go integer (int, int8, int16, int32, int64,
	// uint, uint8, uint16, uint32, or uint64) struct field. The truncation logic does not apply to
	// BSON "decimal128" values.
	truncate bool

	binaryAsSlice       bool
	decodeObjectIDAsHex bool
	useJSONStructTags   bool
	useLocalTimeZone    bool
	zeroMaps            bool
	zeroStructs         bool
}

// EncoderRegistry is an interface provides a ValueEncoder based on the given reflect.Type.
type EncoderRegistry interface {
	LookupEncoder(reflect.Type) (ValueEncoder, error)
}

// ValueEncoder is the interface implemented by types that can encode a provided Go type to BSON.
// The value to encode is provided as a reflect.Value and a bson.ValueWriter is used within the
// EncodeValue method to actually create the BSON representation. For convenience, ValueEncoderFunc
// is provided to allow use of a function with the correct signature as a ValueEncoder. A pointer
// to a Registry instance is provided to allow implementations to lookup further ValueEncoders.
type ValueEncoder interface {
	EncodeValue(EncoderRegistry, ValueWriter, reflect.Value) error
}

// ValueEncoderFunc is an adapter function that allows a function with the correct signature to be
// used as a ValueEncoder.
type ValueEncoderFunc func(EncoderRegistry, ValueWriter, reflect.Value) error

// EncodeValue implements the ValueEncoder interface.
func (fn ValueEncoderFunc) EncodeValue(reg EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	return fn(reg, vw, val)
}

// DecoderRegistry is an interface provides a ValueDecoder based on the given reflect.Type.
type DecoderRegistry interface {
	LookupDecoder(reflect.Type) (ValueDecoder, error)
	LookupTypeMapEntry(Type) (reflect.Type, error)
}

// ValueDecoder is the interface implemented by types that can decode BSON to a provided Go type.
// Implementations should ensure that the value they receive is settable. Similar to ValueEncoderFunc,
// ValueDecoderFunc is provided to allow the use of a function with the correct signature as a
// ValueDecoder. A DecodeContext instance is provided and serves similar functionality to the
// EncodeContext.
type ValueDecoder interface {
	DecodeValue(DecoderRegistry, ValueReader, reflect.Value) error
}

// ValueDecoderFunc is an adapter function that allows a function with the correct signature to be
// used as a ValueDecoder.
type ValueDecoderFunc func(DecoderRegistry, ValueReader, reflect.Value) error

// DecodeValue implements the ValueDecoder interface.
func (fn ValueDecoderFunc) DecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	return fn(reg, vr, val)
}

// typeDecoder is the interface implemented by types that can handle the decoding of a value given its type.
type typeDecoder interface {
	decodeType(DecoderRegistry, ValueReader, reflect.Type) (reflect.Value, error)
}

// typeDecoderFunc is an adapter function that allows a function with the correct signature to be used as a typeDecoder.
type typeDecoderFunc func(DecoderRegistry, ValueReader, reflect.Type) (reflect.Value, error)

func (fn typeDecoderFunc) decodeType(reg DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	return fn(reg, vr, t)
}

// decodeAdapter allows two functions with the correct signatures to be used as both a ValueDecoder and typeDecoder.
type decodeAdapter struct {
	ValueDecoderFunc
	typeDecoderFunc
}

var _ ValueDecoder = decodeAdapter{}
var _ typeDecoder = decodeAdapter{}

func decodeTypeOrValueWithInfo(vd ValueDecoder, reg DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	td, ok := vd.(typeDecoder)
	if ok && td != nil {
		return td.decodeType(reg, vr, t)
	}

	val := reflect.New(t).Elem()
	err := vd.DecodeValue(reg, vr, val)
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
