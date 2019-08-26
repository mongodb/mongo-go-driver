// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package assert

import (
	"reflect"
	"testing"
)

// Equal compares first and second for equality.
// If the objects are not equal, the test will be failed with an error message containing msg and args.
func Equal(t testing.TB, first, second interface{}, msg string, args ...interface{}) {
	t.Helper()
	if !reflect.DeepEqual(first, second) {
		t.Fatalf(msg, args...)
	}
}

// NotEqual compares first and second for inequality.
func NotEqual(t testing.TB, first, second interface{}, msg string, args ...interface{}) {
	t.Helper()
	if reflect.DeepEqual(first, second) {
		t.Fatalf(msg, args...)
	}
}

// True asserts that the obj parameter is true.
func True(t testing.TB, obj interface{}, msg string, args ...interface{}) {
	t.Helper()
	b, ok := obj.(bool)
	if !ok || !b {
		t.Fatalf(msg, args...)
	}
}

// False asserts that the actual parameter is true.
func False(t testing.TB, obj interface{}, msg string, args ...interface{}) {
	t.Helper()
	b, ok := obj.(bool)
	if !ok || b {
		t.Fatalf(msg, args...)
	}
}

// Nil asserts that the obj parameter is nil.
func Nil(t testing.TB, obj interface{}, msg string, args ...interface{}) {
	t.Helper()
	if !isNil(obj) {
		t.Fatalf(msg, args...)
	}
}

// NotNil asserts that the obj parameter is not nil.
func NotNil(t testing.TB, obj interface{}, msg string, args ...interface{}) {
	t.Helper()
	if isNil(obj) {
		t.Fatalf(msg, args...)
	}
}

func isNil(object interface{}) bool {
	if object == nil {
		return true
	}

	val := reflect.ValueOf(object)
	switch val.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return val.IsNil()
	default:
		return false
	}
}
