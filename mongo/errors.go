// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"bytes"
	"fmt"

	"github.com/mongodb/mongo-go-driver/core/result"
)

// WriteError is a non-write concern failure that occured as a result of a write
// operation.
type WriteError struct {
	Code    int
	Message string
}

func (we WriteError) Error() string { return we.Message }

// WriteErrors is a group of non-write concern failures that occured as a result
// of a write operation.
type WriteErrors []WriteError

func (we WriteErrors) Error() string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "multiple write errors: [")
	for _, err := range we {
		fmt.Fprintf(&buf, "{%s}", err)
	}
	fmt.Fprint(&buf, "]")
	return buf.String()
}

func writeErrorsFromResult(rwes []result.WriteError) WriteErrors {
	wes := make(WriteErrors, 0, len(rwes))
	for _, err := range rwes {
		wes = append(wes, WriteError{Code: err.Code, Message: err.ErrMsg})
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
