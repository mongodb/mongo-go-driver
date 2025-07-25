// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package errutil

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
)

// TestJoin_Nil asserts that join returns a nil error for the same inputs that
// [errors.Join] returns a nil error.
func TestJoin_Nil(t *testing.T) {
	t.Parallel()

	assert.Equal(t, errors.Join(), join(), "errors.Join() != join()")
	assert.Equal(t, errors.Join(nil), join(nil), "errors.Join(nil) != join(nil)")
	assert.Equal(t, errors.Join(nil, nil), join(nil, nil), "errors.Join(nil, nil) != join(nil, nil)")
}

// TestJoin_Error asserts that join returns an error with the same error message
// as the error returned by [errors.Join].
func TestJoin_Error(t *testing.T) {
	t.Parallel()

	err1 := errors.New("err1")
	err2 := errors.New("err2")

	tests := []struct {
		desc string
		errs []error
	}{{
		desc: "single error",
		errs: []error{err1},
	}, {
		desc: "two errors",
		errs: []error{err1, err2},
	}, {
		desc: "two errors and a nil value",
		errs: []error{err1, nil, err2},
	}}

	for _, test := range tests {
		test := test // Capture range variable.

		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			want := errors.Join(test.errs...).Error()
			got := join(test.errs...).Error()
			assert.Equal(t,
				want,
				got,
				"errors.Join().Error() != join().Error() for input %v",
				test.errs)
		})
	}
}

// TestJoin_ErrorsIs asserts that join returns an error that behaves identically
// to the error returned by [errors.Join] when passed to [errors.Is].
func TestJoin_ErrorsIs(t *testing.T) {
	t.Parallel()

	err1 := errors.New("err1")
	err2 := errors.New("err2")

	tests := []struct {
		desc   string
		errs   []error
		target error
	}{{
		desc:   "one error with a matching target",
		errs:   []error{err1},
		target: err1,
	}, {
		desc:   "one error with a non-matching target",
		errs:   []error{err1},
		target: err2,
	}, {
		desc:   "nil error",
		errs:   []error{nil},
		target: err1,
	}, {
		desc:   "no errors",
		errs:   []error{},
		target: err1,
	}, {
		desc:   "two different errors with a matching target",
		errs:   []error{err1, err2},
		target: err2,
	}, {
		desc:   "two identical errors with a matching target",
		errs:   []error{err1, err1},
		target: err1,
	}, {
		desc:   "wrapped error with a matching target",
		errs:   []error{fmt.Errorf("error: %w", err1)},
		target: err1,
	}, {
		desc:   "nested joined error with a matching target",
		errs:   []error{err1, join(err2, errors.New("nope"))},
		target: err2,
	}, {
		desc:   "nested joined error with no matching targets",
		errs:   []error{err1, join(errors.New("nope"), errors.New("nope 2"))},
		target: err2,
	}, {
		desc:   "nested joined error with a wrapped matching target",
		errs:   []error{join(fmt.Errorf("error: %w", err1), errors.New("nope")), err2},
		target: err1,
	}, {
		desc:   "context.DeadlineExceeded",
		errs:   []error{err1, nil, context.DeadlineExceeded, err2},
		target: context.DeadlineExceeded,
	}, {
		desc:   "wrapped context.DeadlineExceeded",
		errs:   []error{err1, nil, fmt.Errorf("error: %w", context.DeadlineExceeded), err2},
		target: context.DeadlineExceeded,
	}}

	for _, test := range tests {
		test := test // Capture range variable.

		t.Run(test.desc, func(t *testing.T) {
			// Assert that top-level errors returned by errors.Join and join
			// behave the same with errors.Is.
			want := errors.Join(test.errs...)
			got := join(test.errs...)
			assert.Equal(t,
				errors.Is(want, test.target),
				errors.Is(got, test.target),
				"errors.Join() and join() behave differently with errors.Is")

			// Assert that wrapped errors returned by errors.Join and join
			// behave the same with errors.Is.
			want = fmt.Errorf("error: %w", errors.Join(test.errs...))
			got = fmt.Errorf("error: %w", join(test.errs...))
			assert.Equal(t,
				errors.Is(want, test.target),
				errors.Is(got, test.target),
				"errors.Join() and join(), when wrapped, behave differently with errors.Is")
		})
	}
}

type errType1 struct{}

func (errType1) Error() string { return "" }

type errType2 struct{}

func (errType2) Error() string { return "" }

// TestJoin_ErrorsIs asserts that join returns an error that behaves identically
// to the error returned by [errors.Join] when passed to [errors.As].
func TestJoin_ErrorsAs(t *testing.T) {
	t.Parallel()

	err1 := errType1{}
	err2 := errType2{}

	tests := []struct {
		desc   string
		errs   []error
		target any
	}{{
		desc:   "one error with a matching target",
		errs:   []error{err1},
		target: &errType1{},
	}, {
		desc:   "one error with a non-matching target",
		errs:   []error{err1},
		target: &errType2{},
	}, {
		desc:   "nil error",
		errs:   []error{nil},
		target: &errType1{},
	}, {
		desc:   "no errors",
		errs:   []error{},
		target: &errType1{},
	}, {
		desc:   "two different errors with a matching target",
		errs:   []error{err1, err2},
		target: &errType2{},
	}, {
		desc:   "two identical errors with a matching target",
		errs:   []error{err1, err1},
		target: &errType1{},
	}, {
		desc:   "wrapped error with a matching target",
		errs:   []error{fmt.Errorf("error: %w", err1)},
		target: &errType1{},
	}, {
		desc:   "nested joined error with a matching target",
		errs:   []error{err1, join(err2, errors.New("nope"))},
		target: &errType2{},
	}, {
		desc:   "nested joined error with no matching targets",
		errs:   []error{err1, join(errors.New("nope"), errors.New("nope 2"))},
		target: &errType2{},
	}, {
		desc:   "nested joined error with a wrapped matching target",
		errs:   []error{join(fmt.Errorf("error: %w", err1), errors.New("nope")), err2},
		target: &errType1{},
	}, {
		desc:   "context.DeadlineExceeded",
		errs:   []error{err1, nil, context.DeadlineExceeded, err2},
		target: &errType2{},
	}}

	for _, test := range tests {
		test := test // Capture range variable.

		t.Run(test.desc, func(t *testing.T) {
			// Assert that top-level errors returned by errors.Join and join
			// behave the same with errors.As.
			want := errors.Join(test.errs...)
			got := join(test.errs...)
			assert.Equal(t,
				errors.As(want, test.target),
				errors.As(got, test.target),
				"errors.Join() and join() behave differently with errors.As")

			// Assert that wrapped errors returned by errors.Join and join
			// behave the same with errors.As.
			want = fmt.Errorf("error: %w", errors.Join(test.errs...))
			got = fmt.Errorf("error: %w", join(test.errs...))
			assert.Equal(t,
				errors.As(want, test.target),
				errors.As(got, test.target),
				"errors.Join() and join(), when wrapped, behave differently with errors.As")
		})
	}
}
