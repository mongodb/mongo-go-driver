// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package errutil

import (
	"reflect"
	"strconv"
	"strings"
)

// RetryError aggregates the errors from every attempt of a retried operation.
//
// The error message lists the attempts in chronological order, so it reads as a
// timeline of the failure. Unwrap, on the other hand, returns the attempts with
// the final attempt first, so that errors.Is and errors.As resolve against the
// error the operation actually failed with. That preserves the inspection
// behavior of the single error that was returned before retry errors were
// joined, while still making the earlier errors reachable.
type RetryError struct {
	// attempts holds the attempt errors in chronological order. The last
	// element is the error from the final attempt.
	attempts []error

	// unwrapped holds the same errors with the final attempt first. It is
	// precomputed because errors.Is and errors.As call Unwrap repeatedly and
	// must not mutate the returned slice.
	unwrapped []error
}

// NewRetryError returns an error that aggregates the errors from the previous
// attempts of a retried operation with the error from its final attempt.
//
// It returns nil if final is nil, and returns final unchanged if there are no
// distinct previous errors. Previous errors that are nil, or that are equal to
// the final error, are omitted: some retry paths deliberately return an error
// from an earlier attempt as the final error, and repeating it in the message
// would be noise.
func NewRetryError(prev []error, final error) error {
	if final == nil {
		return nil
	}

	attempts := make([]error, 0, len(prev)+1)
	for _, err := range prev {
		if err == nil || reflect.DeepEqual(err, final) {
			continue
		}
		attempts = append(attempts, err)
	}
	if len(attempts) == 0 {
		return final
	}
	attempts = append(attempts, final)

	unwrapped := make([]error, 0, len(attempts))
	for i := len(attempts) - 1; i >= 0; i-- {
		unwrapped = append(unwrapped, attempts[i])
	}

	return &RetryError{attempts: attempts, unwrapped: unwrapped}
}

// Attempts returns the attempt errors in chronological order. The last element
// is the error from the final attempt.
func (e *RetryError) Attempts() []error {
	attempts := make([]error, len(e.attempts))
	copy(attempts, e.attempts)

	return attempts
}

// Error implements the error interface. It lists the attempt errors in
// chronological order, one per line.
func (e *RetryError) Error() string {
	var sb strings.Builder
	for i, err := range e.attempts {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString("attempt ")
		sb.WriteString(strconv.Itoa(i + 1))
		if i == len(e.attempts)-1 {
			sb.WriteString(" (final)")
		}
		sb.WriteString(": ")
		sb.WriteString(err.Error())
	}

	return sb.String()
}

// Unwrap returns the attempt errors with the final attempt first, so that
// errors.Is and errors.As match the final attempt before any earlier one.
func (e *RetryError) Unwrap() []error {
	return e.unwrapped
}
