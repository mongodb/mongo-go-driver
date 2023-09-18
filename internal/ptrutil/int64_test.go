// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ptrutil

import (
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
)

func TestCompareInt64(t *testing.T) {
	t.Parallel()

	int64ToPtr := func(i64 int64) *int64 { return &i64 }
	int64Ptr := int64ToPtr(1)

	tests := []struct {
		name       string
		ptr1, ptr2 *int64
		want       int
	}{
		{
			name: "empty",
			want: 0,
		},
		{
			name: "ptr1 nil",
			ptr2: int64ToPtr(1),
			want: -2,
		},
		{
			name: "ptr2 nil",
			ptr1: int64ToPtr(1),
			want: 2,
		},
		{
			name: "ptr1 and ptr2 have same value, different address",
			ptr1: int64ToPtr(1),
			ptr2: int64ToPtr(1),
			want: 0,
		},
		{
			name: "ptr1 and ptr2 have the same address",
			ptr1: int64Ptr,
			ptr2: int64Ptr,
			want: 0,
		},
		{
			name: "ptr1 GT ptr2",
			ptr1: int64ToPtr(1),
			ptr2: int64ToPtr(0),
			want: 1,
		},
		{
			name: "ptr1 LT ptr2",
			ptr1: int64ToPtr(0),
			ptr2: int64ToPtr(1),
			want: -1,
		},
	}

	for _, test := range tests {
		test := test // capture the range variable

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := CompareInt64(test.ptr1, test.ptr2)
			assert.Equal(t, test.want, got, "compareInt64() = %v, wanted %v", got, test.want)
		})
	}
}
