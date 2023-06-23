// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package internal

import (
	"bytes"
	"fmt"
	"reflect"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

type ErrNilValue struct{}

func (e ErrNilValue) Error() string {
	return "value is nil"
}

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

type EncoderFn func(*bytes.Buffer, *bsoncodec.Registry) (*bson.Encoder, error)

func MarshalValue(val interface{}, registry *bsoncodec.Registry, encFn EncoderFn) (bsoncore.Value, error) {
	if registry == nil {
		registry = bson.DefaultRegistry
	}

	if val == nil {
		return bsoncore.Value{}, ErrNilValue{}
	}

	buf := new(bytes.Buffer)
	enc, err := encFn(buf, registry)
	if err != nil {
		return bsoncore.Value{}, fmt.Errorf("error configuring BSON encoder: %w", err)
	}

	// Encode the value in a single-element document with an empty key. Use bsoncore to extract the
	// first element and return the BSON value.
	err = enc.Encode(bson.D{{Key: "", Value: val}})
	if err != nil {
		return bsoncore.Value{}, MarshalError{Value: val, Err: err}
	}

	return bsoncore.Document(buf.Bytes()).Index(0).Value(), nil
}
