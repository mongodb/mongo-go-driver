// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package errutil

import (
	"bytes"
	"fmt"
)

// batchErrorsTargetLength is the target length of error messages returned by batch operation
// error types. Try to limit batch error messages to 2kb to prevent problems when printing error
// messages from large batch operations.
const batchErrorsTargetLength = 2000

// JoinBatchErrors appends messages from the given errors to a comma-separated string. If the
// string exceeds 2kb, it stops appending error messages and appends the message "+N more errors..."
// to the end.
//
// Example format:
//
//	"[message 1, message 2, +8 more errors...]"
func JoinBatchErrors(errs []error) string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "[")
	for idx, err := range errs {
		if idx != 0 {
			fmt.Fprint(&buf, ", ")
		}
		// If the error message has exceeded the target error message length, stop appending errors
		// to the message and append the number of remaining errors instead.
		if buf.Len() > batchErrorsTargetLength {
			fmt.Fprintf(&buf, "+%d more errors...", len(errs)-idx)
			break
		}
		fmt.Fprint(&buf, err.Error())
	}
	fmt.Fprint(&buf, "]")

	return buf.String()
}
