// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"math"
	"strings"
	"testing"
)

func FuzzDecodeValue(f *testing.F) {
	// Seed the fuzz corpus with all BSON values from the MarshalValue test
	// cases.
	for _, tc := range marshalValueTestCases {
		f.Add(byte(tc.bsontype), tc.bytes)
	}

	// Also seed the fuzz corpus with special values that we want to test.
	values := []any{
		// int32, including max and min values.
		int32(0), int32(math.MaxInt32), int32(math.MinInt32),
		// int64, including max and min values.
		int64(0), int64(math.MaxInt64), int64(math.MinInt64),
		// string, including empty and large string.
		"", strings.Repeat("z", 10_000),
		// map
		map[string]any{"nested": []any{1, "two", map[string]any{"three": 3}}},
		// array
		[]any{1, 2, 3, "four"},
	}

	for _, v := range values {
		typ, b, err := MarshalValue(v)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(byte(typ), b)
	}

	f.Fuzz(func(t *testing.T, bsonType byte, data []byte) {
		var v any
		if err := UnmarshalValue(Type(bsonType), data, &v); err != nil {
			return
		}

		// There is no value encoder for Go "nil" (nil values handled
		// differently by each type encoder), so skip anything that unmarshals
		// to "nil". It's not clear if MarshalValue should support "nil", but
		// for now we skip it.
		if v == nil {
			t.Logf("data unmarshaled to nil: %v", data)
			return
		}

		typ, encoded, err := MarshalValue(v)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var v2 any
		if err := UnmarshalValue(typ, encoded, &v2); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
	})
}
