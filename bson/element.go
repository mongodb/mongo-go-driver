// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/go-stack/stack"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
)

const validateMaxDepthDefault = 2048

// ErrTooSmall indicates that a slice provided to write into is not large enough to fit the data.
type ErrTooSmall struct {
	Stack stack.CallStack
}

// NewErrTooSmall creates a new ErrTooSmall with the given message and the current stack.
func NewErrTooSmall() ErrTooSmall {
	return ErrTooSmall{Stack: stack.Trace().TrimRuntime()}
}

// Error implements the error interface.
func (e ErrTooSmall) Error() string {
	return "too small"
}

// ErrorStack returns a string representing the stack at the point where the error occurred.
func (e ErrTooSmall) ErrorStack() string {
	s := bytes.NewBufferString("too small: [")

	for i, call := range e.Stack {
		if i != 0 {
			s.WriteString(", ")
		}

		// go vet doesn't like %k even though it's part of stack's API, so we move the format
		// string so it doesn't complain. (We also can't make it a constant, or go vet still
		// complains.)
		callFormat := "%k.%n %v"

		s.WriteString(fmt.Sprintf(callFormat, call, call, call))
	}

	s.WriteRune(']')

	return s.String()
}

// Equals checks that err2 also is an ErrTooSmall.
func (e ErrTooSmall) Equals(err2 error) bool {
	switch err2.(type) {
	case ErrTooSmall:
		return true
	default:
		return false
	}
}

// ErrUninitializedElement is returned whenever any method is invoked on an uninitialized Element.
var ErrUninitializedElement = errors.New("bson/ast/compact: Method call on uninitialized Element")

// ErrInvalidWriter indicates that a type that can't be written into was passed to a writer method.
var ErrInvalidWriter = errors.New("bson: invalid writer provided")

// ErrInvalidString indicates that a BSON string value had an incorrect length.
var ErrInvalidString = errors.New("invalid string value")

// ErrInvalidBinarySubtype indicates that a BSON binary value had an undefined subtype.
var ErrInvalidBinarySubtype = errors.New("invalid BSON binary Subtype")

// ErrInvalidBooleanType indicates that a BSON boolean value had an incorrect byte.
var ErrInvalidBooleanType = errors.New("invalid value for BSON Boolean Type")

// ErrStringLargerThanContainer indicates that the code portion of a BSON JavaScript code with scope
// value is larger than the specified length of the entire value.
var ErrStringLargerThanContainer = errors.New("string size is larger than the JavaScript code with scope container")

// ErrInvalidElement indicates that a bson.Element had invalid underlying BSON.
var ErrInvalidElement = errors.New("invalid Element")

// ElementTypeError specifies that a method to obtain a BSON value an incorrect type was called on a bson.Value.
//
// TODO: rename this ValueTypeError.
type ElementTypeError struct {
	Method string
	Type   bsontype.Type
}

// Error implements the error interface.
func (ete ElementTypeError) Error() string {
	return "Call of " + ete.Method + " on " + ete.Type.String() + " type"
}
