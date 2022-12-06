// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
package driver

import (
	"errors"
	"testing"
)

func TestErrorIs(t *testing.T) {
	t.Parallel()

	for _, tcase := range []struct {
		name string
		want bool
		err1 error
		err2 error
	}{
		{
			"Error with same codes",
			true,
			Error{Code: 1},
			Error{Code: 1},
		},
		{
			"Error with different codes",
			false,
			Error{Code: 1},
			Error{Code: 2},
		},
		{
			"Error with different types",
			false,
			Error{Code: 1},
			errors.New("foo"),
		},
		{
			"WriteError with same codes",
			true,
			WriteError{Code: 1},
			WriteError{Code: 1},
		},
		{
			"WriteError with different codes",
			false,
			WriteError{Code: 1},
			WriteError{Code: 2},
		},
		{
			"WriteError with different types",
			false,
			WriteError{Code: 1},
			errors.New("foo"),
		},
		{
			"WriteErrors with same codes",
			true,
			WriteConcernError{Code: 1},
			WriteConcernError{Code: 1},
		},
		{
			"WriteErrors with different codes",
			false,
			WriteConcernError{Code: 1},
			WriteConcernError{Code: 2},
		},
		{
			"WriteErrors with different types",
			false,
			WriteConcernError{Code: 1},
			errors.New("foo"),
		},
		{
			"WriteCommandError with same WriteConcernError and nil WriteErrors",
			true,
			WriteCommandError{WriteConcernError: &WriteConcernError{Code: 1}},
			WriteCommandError{WriteConcernError: &WriteConcernError{Code: 1}},
		},
		{
			"WriteException with different WriteConcernError and nil WriteErrors",
			false,
			WriteCommandError{WriteConcernError: &WriteConcernError{Code: 1}},
			WriteCommandError{WriteConcernError: &WriteConcernError{Code: 2}},
		},
		{
			"WriteCommandError with same WriteConcernError and same WriteError",
			true,
			WriteCommandError{
				WriteConcernError: &WriteConcernError{Code: 1},
				WriteErrors:       []WriteError{{Code: 1}},
			},
			WriteCommandError{
				WriteConcernError: &WriteConcernError{Code: 1},
				WriteErrors:       []WriteError{{Code: 1}},
			},
		},
		{
			"WriteCommandError with different WriteConcernErrors and same WriteErrors",
			false,
			WriteCommandError{
				WriteConcernError: &WriteConcernError{Code: 1},
				WriteErrors:       []WriteError{{Code: 1}},
			},
			WriteCommandError{
				WriteConcernError: &WriteConcernError{Code: 2},
				WriteErrors:       []WriteError{{Code: 1}},
			},
		},
		{
			"WriteCommandError with same WriteConcernErrors and different WriteErrors",
			false,
			WriteCommandError{
				WriteConcernError: &WriteConcernError{Code: 1},
				WriteErrors:       []WriteError{{Code: 1}},
			},
			WriteCommandError{
				WriteConcernError: &WriteConcernError{Code: 1},
				WriteErrors:       []WriteError{{Code: 2}},
			},
		},
		{
			"WriteCommandError with different WriteConcernErrors and different WriteErrors",
			false,
			WriteCommandError{
				WriteConcernError: &WriteConcernError{Code: 1},
				WriteErrors:       []WriteError{{Code: 1}},
			},
			WriteCommandError{
				WriteConcernError: &WriteConcernError{Code: 2},
				WriteErrors:       []WriteError{{Code: 2}},
			},
		},
		{
			"WriteCommandError with nil WriteConcernError and same WriteErrors",
			true,
			WriteCommandError{
				WriteErrors: []WriteError{{Code: 1}},
			},
			WriteCommandError{
				WriteErrors: []WriteError{{Code: 1}},
			},
		},
		{
			"WriteCommandError with different types",
			false,
			WriteCommandError{WriteConcernError: &WriteConcernError{Code: 1}},
			errors.New("foo"),
		},
	} {
		tcase := tcase

		t.Run(tcase.name, func(t *testing.T) {
			t.Parallel()

			if got := errors.Is(tcase.err1, tcase.err2); got != tcase.want {
				t.Errorf("Expected %v, got %v", tcase.want, got)
			}
		})
	}
}
