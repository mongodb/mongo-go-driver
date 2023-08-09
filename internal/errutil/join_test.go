// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package errutil_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/errutil"
)

func TestJoinReturnsNil(t *testing.T) {
	t.Parallel()

	if err := errutil.Join(); err != nil {
		t.Errorf("errutil.Join() = %v, want nil", err)
	}
	if err := errutil.Join(nil); err != nil {
		t.Errorf("errutil.Join(nil) = %v, want nil", err)
	}
	if err := errutil.Join(nil, nil); err != nil {
		t.Errorf("errutil.Join(nil, nil) = %v, want nil", err)
	}
}

func TestJoin_Error(t *testing.T) {
	t.Parallel()

	err1 := errors.New("err1")
	err2 := errors.New("err2")

	tests := []struct {
		errs []error
		want string
	}{
		{
			errs: []error{err1},
			want: "err1",
		},
		{
			errs: []error{err1, err2},
			want: "err1\nerr2",
		},
		{
			errs: []error{err1, nil, err2},
			want: "err1\nerr2",
		},
	}

	for _, test := range tests {
		test := test // Capture range variable.

		t.Run(fmt.Sprintf("Join(%v)", test.errs), func(t *testing.T) {
			t.Parallel()

			got := errutil.Join(test.errs...).Error()
			assert.Equal(t, test.want, got, "expected and actual error strings are different")
		})
	}
}

func TestJoin_ErrorsIs(t *testing.T) {
	t.Parallel()

	err1 := errors.New("err1")
	err2 := errors.New("err2")

	tests := []struct {
		errs   []error
		target error
		want   bool
	}{
		{
			errs:   []error{err1},
			target: err1,
			want:   true,
		},
		{
			errs:   []error{err1},
			target: err2,
			want:   false,
		},
		{
			errs:   []error{err1, err2},
			target: err2,
			want:   true,
		},
		{
			errs:   []error{err1, nil, context.DeadlineExceeded, err2},
			target: context.DeadlineExceeded,
			want:   true,
		},
	}

	for _, test := range tests {
		test := test // Capture range variable.

		t.Run(fmt.Sprintf("Join(%v)", test.errs), func(t *testing.T) {
			err := errutil.Join(test.errs...)
			got := errors.Is(err, test.target)
			assert.Equal(t, test.want, got, "expected and actual errors.Is result are different")
		})
	}
}

type errType1 struct{}

func (errType1) Error() string { return "" }

type errType2 struct{}

func (errType2) Error() string { return "" }

func TestJoin_ErrorsAs(t *testing.T) {
	t.Parallel()

	err1 := errType1{}
	err2 := errType2{}

	tests := []struct {
		errs   []error
		target interface{}
		want   bool
	}{
		{
			errs:   []error{err1},
			target: &errType1{},
			want:   true,
		},
		{
			errs:   []error{err1},
			target: &errType2{},
			want:   false,
		},
		{
			errs:   []error{err1, err2},
			target: &errType2{},
			want:   true,
		},
		{
			errs:   []error{err1, nil, context.DeadlineExceeded, err2},
			target: &errType2{},
			want:   true,
		},
	}

	for _, test := range tests {
		test := test // Capture range variable.

		t.Run(fmt.Sprintf("Join(%v)", test.errs), func(t *testing.T) {
			err := errutil.Join(test.errs...)
			got := errors.As(err, test.target)
			assert.Equal(t, test.want, got, "expected and actual errors.Is result are different")
		})
	}
}
