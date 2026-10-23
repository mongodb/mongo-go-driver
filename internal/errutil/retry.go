// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package errutil

import (
	"strconv"
	"strings"
)

// RetryError collects all errors encountered while retrying an operation.
//
// The main error is the primary error to surface as described by the Retryable
// Writes specification. It is the only error Unwrap returns, so errors.Is and
// errors.As resolve against it exactly, ignoring all other attempt errors. The
// error message prints the main error first, followed by every attempt error in
// chronological order, so the full retry history is in the error message.
type RetryError struct {
	// mainErr is the error the operation returned. Unwrap returns it.
	mainErr error

	// attemptErrs holds every attempt error in chronological order. The main
	// error is usually, but not always, the last element: some retry paths
	// return an error from an earlier attempt.
	attemptErrs []error
}

// NewRetryError returns an error that aggregates the errors from every attempt
// of a retried operation. The attempts are listed in chronological order.
// mainErr is the primary error to surface as described by the Retryable Writes
// specification.
//
// NewRetryError returns nil if mainErr is nil, and returns mainErr unchanged if
// there was only a single attempt. Nil errors are omitted.
func NewRetryError(attemptErrs []error, mainErr error) error {
	if mainErr == nil {
		return nil
	}

	errs := make([]error, 0, len(attemptErrs))
	for _, err := range attemptErrs {
		if err == nil {
			continue
		}
		errs = append(errs, err)
	}
	if len(errs) <= 1 {
		return mainErr
	}

	return &RetryError{mainErr: mainErr, attemptErrs: errs}
}

// Attempts returns the attempt errors in chronological order.
func (e *RetryError) Attempts() []error {
	attempts := make([]error, len(e.attemptErrs))
	copy(attempts, e.attemptErrs)

	return attempts
}

// Error implements the error interface. It returns the main error's message
// followed by the message from each attempt, one per line.
func (e *RetryError) Error() string {
	var sb strings.Builder
	sb.WriteString(e.mainErr.Error())
	for i, err := range e.attemptErrs {
		sb.WriteString("\nattempt ")
		sb.WriteString(strconv.Itoa(i + 1))
		sb.WriteString(": ")
		sb.WriteString(err.Error())
	}

	return sb.String()
}

// Unwrap returns the main error.
func (e *RetryError) Unwrap() error {
	return e.mainErr
}
