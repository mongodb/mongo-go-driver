// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build !go1.20
// +build !go1.20

package errutil

// Join returns an error that wraps the given errors. Any nil error values are
// discarded. Join returns nil if every value in errs is nil. The error formats
// as the concatenation of the strings obtained by calling the Error method of
// each element of errs, with a newline between each string.
//
// A non-nil error returned by Join implements the "Unwrap() error" method.
func Join(errs ...error) error {
	return join(errs...)
}
