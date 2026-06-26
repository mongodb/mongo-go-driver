// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build !cse

package mtest

import "fmt"

// verifyLibmongocryptVersionConstraints returns an error when a non-empty
// version constraint is requested but the driver was not built with the cse
// tag (and therefore has no access to libmongocrypt). When both bounds are
// empty there is nothing to check and nil is returned.
func verifyLibmongocryptVersionConstraints(min, max string) error {
	if min == "" && max == "" {
		return nil
	}
	return fmt.Errorf("libmongocrypt version constraints require the cse build tag")
}
