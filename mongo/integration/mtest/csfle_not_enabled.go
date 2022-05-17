// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build !cse
// +build !cse

package mtest

// IsCSFLEEnabled returns true if driver is built with Client Side Field Level Encryption support.
// Client Side Field Level Encryption support is enabled with the cse build tag.
func IsCSFLEEnabled() bool {
	return false
}
