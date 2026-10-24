// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package errutil

import (
	"errors"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
)

// codedError is a stand-in for driver.Error: it carries a code and, like
// driver.Error, contains a slice so that it is not comparable with "==".
type codedError struct {
	code   int
	labels []string
}

func (e codedError) Error() string {
	return fmt.Sprintf("coded error %d", e.code)
}

func TestNewRetryError(t *testing.T) {
	errFirst := errors.New("first")
	errFinal := errors.New("final")

	t.Run("nil main returns nil", func(t *testing.T) {
		err := NewRetryError([]error{errFirst, errFinal}, nil)
		assert.Nil(t, err, "expected nil error")
	})

	t.Run("no retry attempts returns main unchanged", func(t *testing.T) {
		assert.Equal(t, errFinal, NewRetryError(nil, errFinal), "expected main error")
		assert.Equal(t, errFinal, NewRetryError([]error{}, errFinal), "expected main error")
		assert.Equal(t, errFinal, NewRetryError([]error{errFinal}, errFinal), "expected main error")
	})

	t.Run("nil attempts are omitted", func(t *testing.T) {
		err := NewRetryError([]error{nil, errFinal, nil}, errFinal)
		assert.Equal(t, errFinal, err, "expected main error")
	})

	t.Run("main error is printed before the attempts", func(t *testing.T) {
		err := NewRetryError([]error{errFirst, errors.New("second"), errFinal}, errFinal)

		want := "final\nattempt 1: first\nattempt 2: second\nattempt 3: final"
		assert.Equal(t, want, err.Error(), "unexpected error message")

		var retryErr RetryError
		assert.True(t, errors.As(err, &retryErr), "expected a RetryError")
		assert.Equal(t, 3, len(retryErr.Attempts()), "expected 3 attempts")
		assert.Equal(t, errFirst, retryErr.Attempts()[0], "expected first attempt first")
		assert.Equal(t, errFinal, retryErr.Attempts()[2], "expected final attempt last")
	})

	t.Run("main error is not always the last attempt", func(t *testing.T) {
		// Operation.Execute returns the previous attempt's error rather than a
		// connection error encountered while retrying.
		errConn := errors.New("connection pool cleared")
		err := NewRetryError([]error{errFirst, errConn}, errFirst)

		want := "first\nattempt 1: first\nattempt 2: connection pool cleared"
		assert.Equal(t, want, err.Error(), "unexpected error message")
		assert.ErrorIs(t, err, errFirst, "expected to match the main error")
		assert.False(t, errors.Is(err, errConn), "expected not to match a non-main attempt")
	})

	t.Run("errors.Is matches only the main error", func(t *testing.T) {
		err := NewRetryError([]error{errFirst, errFinal}, errFinal)

		assert.ErrorIs(t, err, errFinal, "expected to match the main error")
		assert.False(t, errors.Is(err, errFirst), "expected not to match an earlier attempt")
	})

	t.Run("errors.As resolves to the main error", func(t *testing.T) {
		first := codedError{code: 10107} // NotWritablePrimary
		final := codedError{
			code:   11602, // InterruptedDueToReplStateChange
			labels: []string{"RetryableWriteError"},
		}
		err := NewRetryError([]error{first, final}, final)

		var ce codedError
		assert.True(t, errors.As(err, &ce), "expected a codedError")
		assert.Equal(t, final.code, ce.code,
			"expected errors.As to resolve to the main error")
		assert.Equal(t, final.labels, ce.labels,
			"expected errors.As to recover the main error's labels")
	})

	t.Run("errors.As resolves through a wrapping error", func(t *testing.T) {
		first := codedError{code: 10107}
		final := codedError{code: 11602}
		err := fmt.Errorf("wrapped: %w", NewRetryError([]error{first, final}, final))

		var ce codedError
		assert.True(t, errors.As(err, &ce), "expected a codedError")
		assert.Equal(t, final.code, ce.code,
			"expected errors.As to resolve to the main error")
	})
}
