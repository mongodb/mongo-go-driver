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
	"go.mongodb.org/mongo-driver/v2/internal/ptrutil"
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
	tests := []struct {
		name        string
		ctxTimeout  *time.Duration
		timeout     time.Duration
		wantTimeout time.Duration
		want        bool
	}{
		{
			name:       "Timeout shorter than context deadline",
			ctxTimeout: ptrutil.Ptr(10 * time.Second),
			timeout:    1 * time.Second,
			want:       true,
		},
		{
			name:       "Timeout equal to context deadline",
			ctxTimeout: ptrutil.Ptr(1 * time.Second),
			timeout:    1 * time.Second,
			want:       false,
		},
		{
			name:       "Timeout greater than context deadline",
			ctxTimeout: ptrutil.Ptr(1 * time.Second),
			timeout:    10 * time.Second,
			want:       false,
		},
		{
			name:       "Context deadline already expired",
			ctxTimeout: ptrutil.Ptr(-1 * time.Second),
			timeout:    1 * time.Second,
			want:       true, // *timeout <= 0 branch in code
		},
		{
			name:       "Timeout is zero, context deadline in future",
			ctxTimeout: ptrutil.Ptr(10 * time.Second),
			timeout:    0 * time.Second,
			want:       true,
		},
		{
			name:       "Timeout is negative, context deadline in future",
			ctxTimeout: ptrutil.Ptr(10 * time.Second),
			timeout:    -1 * time.Second,
			want:       true,
		},
		{
			name:       "Timeout provided, context has no deadline",
			ctxTimeout: nil,
			timeout:    1 * time.Second,
			want:       true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			if test.ctxTimeout != nil {
				var cancel context.CancelFunc

				ctx, cancel = context.WithTimeout(ctx, *test.ctxTimeout)
				defer cancel()
			}

			got := TimeoutWithinContext(ctx, test.timeout)
			assert.Equal(t, test.want, got)
		})
	}
}
