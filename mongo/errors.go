// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/mongodb/mongo-go-driver/core/dispatch"
	"github.com/mongodb/mongo-go-driver/core/result"
)

// ErrUnacknowledgedWrite is returned from functions that have an unacknowledged
// write concern.
var ErrUnacknowledgedWrite = errors.New("unacknowledged write")

// WriteError is a non-write concern failure that occured as a result of a write
// operation.
type WriteError struct {
	Index   int
	Code    int
	Message string
}

func (we WriteError) Error() string { return we.Message }

// WriteErrors is a group of non-write concern failures that occured as a result
// of a write operation.
type WriteErrors []WriteError

func (we WriteErrors) Error() string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "write errors: [")
	for idx, err := range we {
		if idx != 0 {
			fmt.Fprintf(&buf, ", ")
		}
		fmt.Fprintf(&buf, "{%s}", err)
	}
	fmt.Fprint(&buf, "]")
	return buf.String()
}

func writeErrorsFromResult(rwes []result.WriteError) WriteErrors {
	wes := make(WriteErrors, 0, len(rwes))
	for _, err := range rwes {
		wes = append(wes, WriteError{Index: err.Index, Code: err.Code, Message: err.ErrMsg})
	}
	return wes
}

// WriteConcernError is a write concern failure that occured as a result of a
// write operation.
type WriteConcernError struct {
	Code    int
	Message string
}

func (wce WriteConcernError) Error() string { return wce.Message }

// returnResult is used to determine if a function calling processWriteError should return
// the result or return nil. Since the processWriteError function is used by many different
// methods, both *One and *Many, we need a way to differentiate if the method should return
// the result and the error.
type returnResult int

const (
	rrNone returnResult = 1 << iota // None means do not return the result ever.
	rrOne                           // One means return the result if this was called by a *One method.
	rrMany                          // Many means return the result is this was called by a *Many method.

	rrAll returnResult = rrOne | rrMany // All means always return the result.
)

// processWriteError handles processing the result of a write operation. If the returned boolean is true,
// the result of the write operation should be returned. Currently this will only be true when the error
// is an UnacknowledgedWrite. This function will wrap the errors from other packages and return them
// as errors from this package.
//
// WriteErrors will be returned over WriteConcernErrors if both are present.
func processWriteError(wce *result.WriteConcernError, wes []result.WriteError, err error) (returnResult, error) {
	switch {
	case err == dispatch.ErrUnacknowledgedWrite:
		return rrAll, ErrUnacknowledgedWrite
	case err != nil:
		return rrNone, err
	case len(wes) > 0:
		return rrMany, writeErrorsFromResult(wes)
	case wce != nil:
		return rrMany, WriteConcernError{Code: wce.Code, Message: wce.ErrMsg}
	default:
		return rrAll, nil
	}
}
