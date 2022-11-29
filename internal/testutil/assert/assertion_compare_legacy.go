// Copied from https://github.com/stretchr/testify/blob/1333b5d3bda8cf5aedcf3e1aaa95cac28aaab892/assert/assertion_compare_legacy.go

// Copyright 2020 Mat Ryer, Tyler Bunnell and all contributors. All rights reserved.
// Use of this source code is governed by an MIT-style license that can be found in
// the THIRD-PARTY-NOTICES file.

//go:build !go1.17
// +build !go1.17

package assert

import "reflect"

// Older versions of Go does not have the reflect.Value.CanConvert
// method.
func canConvert(value reflect.Value, to reflect.Type) bool {
	return false
}
