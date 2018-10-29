// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncodec

import (
	"fmt"
	"strings"

	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
)

// Marshaler is an interface implemented by types that can marshal themselves
// into a BSON document represented as bytes. The bytes returned must be a valid
// BSON document if the error is nil.
type Marshaler interface {
	MarshalBSON() ([]byte, error)
}

// ValueMarshaler is an interface implemented by types that can marshal
// themselves into a BSON value as bytes. The type must be the valid type for
// the bytes returned. The bytes and byte type together must be valid if the
// error is nil.
type ValueMarshaler interface {
	MarshalBSONValue() (bsontype.Type, []byte, error)
}

// Unmarshaler is an interface implemented by types that can unmarshal a BSON
// document representation of themselves. The BSON bytes can be assumed to be
// valid. UnmarshalBSON must copy the BSON bytes if it wishes to retain the data
// after returning.
type Unmarshaler interface {
	UnmarshalBSON([]byte) error
}

// ValueUnmarshaler is an interface implemented by types that can unmarshal a
// BSON value representaiton of themselves. The BSON bytes and type can be
// assumed to be valid. UnmarshalBSONValue must copy the BSON value bytes if it
// wishes to retain the data after returning.
type ValueUnmarshaler interface {
	UnmarshalBSONValue(bsontype.Type, []byte) error
}

// ValueEncoderError is an error returned from a ValueEncoder when the provided
// value can't be encoded by the ValueEncoder.
type ValueEncoderError struct {
	Name     string
	Types    []interface{}
	Received interface{}
}

func (vee ValueEncoderError) Error() string {
	types := make([]string, 0, len(vee.Types))
	for _, t := range vee.Types {
		types = append(types, fmt.Sprintf("%T", t))
	}
	return fmt.Sprintf("%s can only process %s, but got a %T", vee.Name, strings.Join(types, ", "), vee.Received)
}

// ValueDecoderError is an error returned from a ValueDecoder when the provided
// value can't be decoded by the ValueDecoder.
type ValueDecoderError struct {
	Name     string
	Types    []interface{}
	Received interface{}
}

func (vde ValueDecoderError) Error() string {
	types := make([]string, 0, len(vde.Types))
	for _, t := range vde.Types {
		types = append(types, fmt.Sprintf("%T", t))
	}
	return fmt.Sprintf("%s can only process %s, but got a %T", vde.Name, strings.Join(types, ", "), vde.Received)
}

// EncodeContext is the contextual information required for a Codec to encode a
// value.
type EncodeContext struct {
	*Registry
	MinSize bool
}

// DecodeContext is the contextual information required for a Codec to decode a
// value.
type DecodeContext struct {
	*Registry
	Truncate bool
}

// ValueCodec is the interface that groups the methods to encode and decode
// values.
type ValueCodec interface {
	ValueEncoder
	ValueDecoder
}

// ValueEncoder is the interface implemented by types that can handle the
// encoding of a value. Implementations must handle both values and
// pointers to values.
type ValueEncoder interface {
	EncodeValue(EncodeContext, bsonrw.ValueWriter, interface{}) error
}

// ValueEncoderFunc is an adapter function that allows a function with the
// correct signature to be used as a ValueEncoder.
type ValueEncoderFunc func(EncodeContext, bsonrw.ValueWriter, interface{}) error

// EncodeValue implements the ValueEncoder interface.
func (fn ValueEncoderFunc) EncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, val interface{}) error {
	return fn(ec, vw, val)
}

// ValueDecoder is the interface implemented by types that can handle the
// decoding of a value. Implementations must handle pointers to values,
// including pointers to pointer values. The implementation may create a new
// value and assign it to the pointer if necessary.
type ValueDecoder interface {
	DecodeValue(DecodeContext, bsonrw.ValueReader, interface{}) error
}

// ValueDecoderFunc is an adapter function that allows a function with the
// correct signature to be used as a ValueDecoder.
type ValueDecoderFunc func(DecodeContext, bsonrw.ValueReader, interface{}) error

// DecodeValue implements the ValueDecoder interface.
func (fn ValueDecoderFunc) DecodeValue(dc DecodeContext, vr bsonrw.ValueReader, val interface{}) error {
	return fn(dc, vr, val)
}

// CodecZeroer is the interface implemented by Codecs that can also determine if
// a value of the type that would be encoded is zero.
type CodecZeroer interface {
	IsTypeZero(interface{}) bool
}
