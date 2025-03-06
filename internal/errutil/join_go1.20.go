// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build go1.20
// +build go1.20

package errutil

import "errors"

// Join calls [errors.Join].
func Join(errs ...error) error {
	return errors.Join(errs...)
}
