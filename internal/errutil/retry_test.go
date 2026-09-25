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
	t.Parallel()

	errFirst := errors.New("first")
	errFinal := errors.New("final")

	t.Run("nil final returns nil", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, NewRetryError([]error{errFirst}, nil), "expected nil error")
	})

	t.Run("no previous errors returns final unchanged", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, errFinal, NewRetryError(nil, errFinal), "expected final error")
		assert.Equal(t, errFinal, NewRetryError([]error{}, errFinal), "expected final error")
	})

	t.Run("nil previous errors are omitted", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, errFinal, NewRetryError([]error{nil, nil}, errFinal),
			"expected final error")
	})

	t.Run("previous errors equal to final are omitted", func(t *testing.T) {
		t.Parallel()

		// Retry paths like Operation.Execute may return an error from an
		// earlier attempt as the final error. It must not be listed twice.
		assert.Equal(t, errFinal, NewRetryError([]error{errFinal}, errFinal),
			"expected final error")

		// Uncomparable error types must not panic and must still dedup.
		uncomparable := codedError{code: 11600, labels: []string{"RetryableWriteError"}}
		assert.Equal(t,
			error(uncomparable),
			NewRetryError([]error{codedError{code: 11600, labels: []string{"RetryableWriteError"}}}, uncomparable),
			"expected final error")
	})

	t.Run("attempts are listed chronologically", func(t *testing.T) {
		t.Parallel()

		err := NewRetryError([]error{errFirst, errors.New("second")}, errFinal)

		want := "attempt 1: first\nattempt 2: second\nattempt 3 (final): final"
		assert.Equal(t, want, err.Error(), "unexpected error message")

		var retryErr *RetryError
		assert.True(t, errors.As(err, &retryErr), "expected a *RetryError")
		assert.Equal(t, 3, len(retryErr.Attempts()), "expected 3 attempts")
		assert.Equal(t, errFirst, retryErr.Attempts()[0], "expected first attempt first")
		assert.Equal(t, errFinal, retryErr.Attempts()[2], "expected final attempt last")
	})

	t.Run("errors.Is matches any attempt", func(t *testing.T) {
		t.Parallel()

		err := NewRetryError([]error{errFirst}, errFinal)

		assert.ErrorIs(t, err, errFirst, "expected to match the first attempt")
		assert.ErrorIs(t, err, errFinal, "expected to match the final attempt")
	})

	t.Run("errors.As resolves to the final attempt", func(t *testing.T) {
		t.Parallel()

		first := codedError{code: 10107} // NotWritablePrimary
		final := codedError{code: 11602} // InterruptedDueToReplStateChange
		err := NewRetryError([]error{first}, final)

		var ce codedError
		assert.True(t, errors.As(err, &ce), "expected a codedError")
		assert.Equal(t, final.code, ce.code,
			"expected errors.As to resolve to the final attempt")
	})

	t.Run("errors.As resolves through a wrapping error", func(t *testing.T) {
		t.Parallel()

		final := codedError{code: 11602}
		err := fmt.Errorf("wrapped: %w", NewRetryError([]error{codedError{code: 10107}}, final))

		var ce codedError
		assert.True(t, errors.As(err, &ce), "expected a codedError")
		assert.Equal(t, final.code, ce.code,
			"expected errors.As to resolve to the final attempt")
	})
}
