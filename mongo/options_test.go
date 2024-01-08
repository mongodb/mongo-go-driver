// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
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
				Skip: ptrOf(int64(1)),
			},
		},
		{
			name: "multiplicity",
			opts: []Options[options.FindArgs]{
				options.Find().SetSkip(1),
				options.Find().SetSkip(2),
			},
			want: &options.FindArgs{
				Skip: ptrOf(int64(2)),
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
				Skip: ptrOf(int64(2)),
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
				Skip: ptrOf(int64(2)),
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
				Skip: ptrOf(int64(2)),
			},
		},
	}

	for _, test := range tests {
		test := test // Capture the range variable

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewArgsFromOptions(test.opts...)
			assert.NoError(t, err, "unexpected merging error")
			assert.Equal(t, test.want, got)
		})
	}
}
