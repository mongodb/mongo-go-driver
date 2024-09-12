// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncoreutil

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
)

func TestTruncate(t *testing.T) {
	t.Parallel()

	for _, tcase := range []struct {
		name     string
		arg      string
		width    int
		expected string
	}{
		{
			name:     "empty",
			arg:      "",
			width:    0,
			expected: "",
		},
		{
			name:     "short",
			arg:      "foo",
			width:    1000,
			expected: "foo",
		},
		{
			name:     "long",
			arg:      "foo bar baz",
			width:    9,
			expected: "foo bar b",
		},
		{
			name:     "multi-byte",
			arg:      "你好",
			width:    4,
			expected: "你",
		},
	} {
		tcase := tcase

		t.Run(tcase.name, func(t *testing.T) {
			t.Parallel()

			actual := Truncate(tcase.arg, tcase.width)
			assert.Equal(t, tcase.expected, actual)
		})
	}

}
