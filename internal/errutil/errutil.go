// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package errutil

import (
	"fmt"
)

// WrappedError represents an error that contains another error.
type WrappedError interface {
	// Message gets the basic message of the error.
	Message() string
	// Inner gets the inner error if one exists.
	Inner() error
}

// rolledUpErrorMessage gets a flattened error message.
func rolledUpErrorMessage(err error) string {
	if wrappedErr, ok := err.(WrappedError); ok {
		inner := wrappedErr.Inner()
		if inner != nil {
			return fmt.Sprintf("%s: %s", wrappedErr.Message(), rolledUpErrorMessage(inner))
		}

		return wrappedErr.Message()
	}

	return err.Error()
}

// UnwrapError attempts to unwrap the error down to its root cause.
func UnwrapError(err error) error {

	switch tErr := err.(type) {
	case WrappedError:
		return UnwrapError(tErr.Inner())
	}

	return err
}

// WrapError wraps an error with a message.
func WrapError(inner error, message string) error {
	return &wrappedError{message, inner}
}

// WrapErrorf wraps an error with a message.
func WrapErrorf(inner error, format string, args ...interface{}) error {
	return &wrappedError{fmt.Sprintf(format, args...), inner}
}

type wrappedError struct {
	message string
	inner   error
}

func (e *wrappedError) Message() string {
	return e.message
}

func (e *wrappedError) Error() string {
	return rolledUpErrorMessage(e)
}

func (e *wrappedError) Inner() error {
	return e.inner
}

func (e *wrappedError) Unwrap() error {
	return e.inner
}
