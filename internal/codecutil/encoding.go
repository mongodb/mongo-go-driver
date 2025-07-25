// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package codecutil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

var ErrNilValue = errors.New("value is nil")

// MarshalError is returned when attempting to transform a value into a document
// results in an error.
type MarshalError struct {
	Value any
	Err   error
}

// Error implements the error interface.
func (e MarshalError) Error() string {
	return fmt.Sprintf("cannot marshal type %q to a BSON Document: %v",
		reflect.TypeOf(e.Value), e.Err)
}

func (e MarshalError) Unwrap() error { return e.Err }

// EncoderFn is used to functionally construct an encoder for marshaling values.
type EncoderFn func(io.Writer) *bson.Encoder

// MarshalValue will attempt to encode the value with the encoder returned by
// the encoder function.
func MarshalValue(val any, encFn EncoderFn) (bsoncore.Value, error) {
	// If the val is already a bsoncore.Value, then do nothing.
	if bval, ok := val.(bsoncore.Value); ok {
		return bval, nil
	}

	if val == nil {
		return bsoncore.Value{}, ErrNilValue
	}

	buf := new(bytes.Buffer)

	enc := encFn(buf)

	// Encode the value in a single-element document with an empty key. Use
	// bsoncore to extract the first element and return the BSON value.
	err := enc.Encode(bson.D{{Key: "", Value: val}})
	if err != nil {
		return bsoncore.Value{}, MarshalError{Value: val, Err: err}
	}

	return bsoncore.Document(buf.Bytes()).Index(0).Value(), nil
}
