// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongoutil

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func BenchmarkNewOptions(b *testing.B) {
	b.Run("reflect.ValueOf is always called", func(b *testing.B) {
		opts := make([]options.Lister[options.FindOptions], b.N)

		// Create a huge string to see if we can force reflect.ValueOf to use heap
		// over stack.
		size := 16 * 1024 * 1024
		str := strings.Repeat("a", size)

		for i := 0; i < b.N; i++ {
			opts[i] = options.Find().SetComment(str).SetHint("y").SetMin(1).SetMax(2)
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = NewOptions[options.FindOptions](opts...)
		}
	})
}

func TestValidChangeStreamTimeouts(t *testing.T) {
	t.Parallel()

	newDurPtr := func(dur time.Duration) *time.Duration {
		return &dur
	}

	tests := []struct {
		name                     string
		parent                   context.Context
		maxAwaitTimeout, timeout *time.Duration
		wantTimeout              time.Duration
		want                     bool
	}{
		{
			name:            "no context deadline and no timeouts",
			parent:          context.Background(),
			maxAwaitTimeout: nil,
			timeout:         nil,
			wantTimeout:     0,
			want:            true,
		},
		{
			name:            "no context deadline and maxAwaitTimeout",
			parent:          context.Background(),
			maxAwaitTimeout: newDurPtr(1),
			timeout:         nil,
			wantTimeout:     0,
			want:            true,
		},
		{
			name:            "no context deadline and timeout",
			parent:          context.Background(),
			maxAwaitTimeout: nil,
			timeout:         newDurPtr(1),
			wantTimeout:     0,
			want:            true,
		},
		{
			name:            "no context deadline and maxAwaitTime gt timeout",
			parent:          context.Background(),
			maxAwaitTimeout: newDurPtr(2),
			timeout:         newDurPtr(1),
			wantTimeout:     0,
			want:            false,
		},
		{
			name:            "no context deadline and maxAwaitTime lt timeout",
			parent:          context.Background(),
			maxAwaitTimeout: newDurPtr(1),
			timeout:         newDurPtr(2),
			wantTimeout:     0,
			want:            true,
		},
		{
			name:            "no context deadline and maxAwaitTime eq timeout",
			parent:          context.Background(),
			maxAwaitTimeout: newDurPtr(1),
			timeout:         newDurPtr(1),
			wantTimeout:     0,
			want:            false,
		},
		{
			name:            "no context deadline and maxAwaitTime with negative timeout",
			parent:          context.Background(),
			maxAwaitTimeout: newDurPtr(1),
			timeout:         newDurPtr(-1),
			wantTimeout:     0,
			want:            true,
		},
	}

	for _, test := range tests {
		test := test // Capture the range variable

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := ValidMaxAwaitTimeMS(test.parent, test.timeout, test.maxAwaitTimeout)
			assert.Equal(t, test.want, got)
		})
	}
}
