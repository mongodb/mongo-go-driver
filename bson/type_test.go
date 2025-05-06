// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"testing"
)

func TestType(t *testing.T) {
	testCases := []struct {
		name string
		t    Type
		want string
	}{
		{"double", TypeDouble, "double"},
		{"string", TypeString, "string"},
		{"embedded document", TypeEmbeddedDocument, "embedded document"},
		{"array", TypeArray, "array"},
		{"binary", TypeBinary, "binary"},
		{"undefined", TypeUndefined, "undefined"},
		{"objectID", TypeObjectID, "objectID"},
		{"boolean", TypeBoolean, "boolean"},
		{"UTC datetime", TypeDateTime, "UTC datetime"},
		{"null", TypeNull, "null"},
		{"regex", TypeRegex, "regex"},
		{"dbPointer", TypeDBPointer, "dbPointer"},
		{"javascript", TypeJavaScript, "javascript"},
		{"symbol", TypeSymbol, "symbol"},
		{"code with scope", TypeCodeWithScope, "code with scope"},
		{"32-bit integer", TypeInt32, "32-bit integer"},
		{"timestamp", TypeTimestamp, "timestamp"},
		{"64-bit integer", TypeInt64, "64-bit integer"},
		{"128-bit decimal", TypeDecimal128, "128-bit decimal"},
		{"max key", TypeMaxKey, "max key"},
		{"min key", TypeMinKey, "min key"},
		{"invalid", (0), "invalid"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.t.String()
			if got != tc.want {
				t.Errorf("String outputs do not match. got %s; want %s", got, tc.want)
			}
		})
	}
}
