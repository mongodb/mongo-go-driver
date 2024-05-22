// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/ptrutil"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestNewArgsFromOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []Options[options.FindArgs]
		want *options.FindArgs
	}{
		{
			name: "nil",
			opts: nil,
			want: &options.FindArgs{},
		},
		{
			name: "empty",
			opts: []Options[options.FindArgs]{},
			want: &options.FindArgs{},
		},
		{
			name: "singleton",
			opts: []Options[options.FindArgs]{
				options.Find().SetSkip(1),
			},
			want: &options.FindArgs{
				Skip: ptrutil.Ptr(int64(1)),
			},
		},
		{
			name: "multiplicity",
			opts: []Options[options.FindArgs]{
				options.Find().SetSkip(1),
				options.Find().SetSkip(2),
			},
			want: &options.FindArgs{
				Skip: ptrutil.Ptr(int64(2)),
			},
		},
		{
			name: "interior null",
			opts: []Options[options.FindArgs]{
				options.Find().SetSkip(1),
				nil,
				options.Find().SetSkip(2),
			},
			want: &options.FindArgs{
				Skip: ptrutil.Ptr(int64(2)),
			},
		},
		{
			name: "start null",
			opts: []Options[options.FindArgs]{
				nil,
				options.Find().SetSkip(1),
				options.Find().SetSkip(2),
			},
			want: &options.FindArgs{
				Skip: ptrutil.Ptr(int64(2)),
			},
		},
		{
			name: "end null",
			opts: []Options[options.FindArgs]{
				options.Find().SetSkip(1),
				options.Find().SetSkip(2),
				nil,
			},
			want: &options.FindArgs{
				Skip: ptrutil.Ptr(int64(2)),
			},
		},
	}

	for _, test := range tests {
		test := test // Capture the range variable

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := newArgsFromOptions(test.opts...)
			assert.NoError(t, err, "unexpected merging error")
			assert.Equal(t, test.want, got)
		})
	}
}

func BenchmarkNewArgsFromOptions(b *testing.B) {
	mockOptions := make([]Options[options.BulkWriteArgs], b.N)
	for i := 0; i < b.N; i = i + 2 {
		// Specifically benchmark the case where a nil value is assigned to the
		// Options interface.
		var bwo *options.BulkWriteOptions

		mockOptions[i] = bwo

		if i+1 < b.N {
			mockOptions[i+1] = options.BulkWrite()
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		_, _ = newArgsFromOptions(mockOptions[i])
	}
}
