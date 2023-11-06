// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package errutil

import "errors"

// join is a Go 1.13-1.19 compatible version of [errors.Join]. It is only called
// by Join in join_go1.19.go. It is included here in a file without build
// constraints only for testing purposes.
//
// It is heavily based on Join from
// https://cs.opensource.google/go/go/+/refs/tags/go1.21.0:src/errors/join.go
func join(errs ...error) error {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	e := &joinError{
		errs: make([]error, 0, n),
	}
	for _, err := range errs {
		if err != nil {
			e.errs = append(e.errs, err)
		}
	}
	return e
}

// joinError is a Go 1.13-1.19 compatible joinable error type. Its error
// message is identical to [errors.Join], but it implements "Unwrap() error"
// instead of "Unwrap() []error".
//
// It is heavily based on the joinError from
// https://cs.opensource.google/go/go/+/refs/tags/go1.21.0:src/errors/join.go
type joinError struct {
	errs []error
}

func (e *joinError) Error() string {
	var b []byte
	for i, err := range e.errs {
		if i > 0 {
			b = append(b, '\n')
		}
		b = append(b, err.Error()...)
	}
	return string(b)
}

// Unwrap returns another joinError with the same errors as the current
// joinError except the first error in the slice. Continuing to call Unwrap
// on each returned error will increment through every error in the slice. The
// resulting behavior when using [errors.Is] and [errors.As] is similar to an
// error created using [errors.Join] in Go 1.20+.
func (e *joinError) Unwrap() error {
	if len(e.errs) == 1 {
		return e.errs[0]
	}
	return &joinError{errs: e.errs[1:]}
}

// Is calls [errors.Is] with the first error in the slice.
func (e *joinError) Is(target error) bool {
	if len(e.errs) == 0 {
		return false
	}
	return errors.Is(e.errs[0], target)
}

// As calls [errors.As] with the first error in the slice.
func (e *joinError) As(target interface{}) bool {
	if len(e.errs) == 0 {
		return false
	}
	return errors.As(e.errs[0], target)
}
