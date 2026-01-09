// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package testutil

import (
	"reflect"
	"unsafe"
)

// getUnexportedField uses reflection and unsafe to read an unexported field
// named fieldName from a struct pointer v, and returns it as interface{}.
func getUnexportedField(v any, fieldName string) any {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		panic("GetUnexportedField: v must be a non-nil pointer to struct")
	}

	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		panic("GetUnexportedField: v must point to a struct")
	}

	field := elem.FieldByName(fieldName)
	if !field.IsValid() {
		panic("GetUnexportedField: no such field: " + fieldName)
	}

	// Rebuild a settable value pointing to this unexported field.
	field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
	return field.Interface()
}

// GetUnexportedFieldAs is a generic wrapper around GetUnexportedField that
// returns the field value with the correct type.
func GetUnexportedFieldAs[T any](v any, fieldName string) T {
	return getUnexportedField(v, fieldName).(T)
}
