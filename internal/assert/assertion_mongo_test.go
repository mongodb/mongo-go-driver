// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package assert

import (
	"testing"
)

func TestDifferentAddressRanges(t *testing.T) {
	t.Parallel()

	slice := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	testCases := []struct {
		name string
		a    []byte
		b    []byte
		want bool
	}{
		{
			name: "distinct byte slices",
			a:    []byte{0, 1, 2, 3},
			b:    []byte{0, 1, 2, 3},
			want: true,
		},
		{
			name: "same byte slice",
			a:    slice,
			b:    slice,
			want: false,
		},
		{
			name: "whole and subslice",
			a:    slice,
			b:    slice[:4],
			want: false,
		},
		{
			name: "two subslices",
			a:    slice[1:2],
			b:    slice[3:4],
			want: false,
		},
		{
			name: "empty",
			a:    []byte{0, 1, 2, 3},
			b:    []byte{},
			want: true,
		},
		{
			name: "nil",
			a:    []byte{0, 1, 2, 3},
			b:    nil,
			want: true,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := DifferentAddressRanges(new(testing.T), tc.a, tc.b)
			if got != tc.want {
				t.Errorf("DifferentAddressRanges(%p, %p) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
