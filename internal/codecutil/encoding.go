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

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var ErrNilValue = errors.New("value is nil")

// ErrMapForOrderedArgument is returned when a map with multiple keys is passed
// to a CRUD method for an ordered parameter
type ErrMapForOrderedArgument struct {
	ParamName string
}

// Error implements the error interface.
func (e ErrMapForOrderedArgument) Error() string {
	return fmt.Sprintf("multi-key map passed in for ordered parameter %v",
		e.ParamName)
}

// MarshalError is returned when attempting to transform a value into a document
// results in an error.
type MarshalError struct {
	Value interface{}
	Err   error
}

// Error implements the error interface.
func (e MarshalError) Error() string {
	return fmt.Sprintf("cannot transform type %s to a BSON Document: %v",
		reflect.TypeOf(e.Value), e.Err)
}

// EncoderFn is used to functionally construct an encoder for marshaling values.
type EncoderFn func(io.Writer) (*bson.Encoder, error)

// defaultEncoderFn will return a function that will construct an encoder with
// a BSON Value writer.
func defaultEncoderFn() EncoderFn {
	return func(w io.Writer) (*bson.Encoder, error) {
		rw, err := bsonrw.NewBSONValueWriter(w)
		if err != nil {
			return nil, err
		}

		enc, err := bson.NewEncoder(rw)
		if err != nil {
			return nil, err
		}

		return enc, nil
	}
}

// newMarshalValueEncoder will attempt to construct an encoder from the provided
// writer and encoder function. If the encoder function is not provided, then
// this function will construct an encoder with a BSON Value writer.
func newMarshalValueEncoder(w io.Writer, encFn EncoderFn) (*bson.Encoder, error) {
	if encFn == nil {
		encFn = defaultEncoderFn()
	}

	enc, err := encFn(w)
	if err != nil {
		return nil, fmt.Errorf("error configuring BSON encoder: %w", err)
	}

	return enc, nil
}

// MarshalValue will attempt to encode the provided value with the registry and
// encoder function. If the encoder function does not exist, then this
func MarshalValue(val interface{}, encFn EncoderFn) (bsoncore.Value, error) {
	// If the val is already a bsoncore.Value, then do nothing.
	if bval, ok := val.(bsoncore.Value); ok {
		return bval, nil
	}

	if val == nil {
		return bsoncore.Value{}, ErrNilValue
	}

	buf := new(bytes.Buffer)

	enc, err := newMarshalValueEncoder(buf, encFn)
	if err != nil {
		return bsoncore.Value{}, err
	}

	// Encode the value in a single-element document with an empty key. Use
	// bsoncore to extract the first element and return the BSON value.
	err = enc.Encode(bson.D{{Key: "", Value: val}})
	if err != nil {
		return bsoncore.Value{}, MarshalError{Value: val, Err: err}
	}

	return bsoncore.Document(buf.Bytes()).Index(0).Value(), nil
}
