// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package csot

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal/assert"
)

func newTestContext(t *testing.T, timeout time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)

	return ctx
}

func newDurPtr(dur time.Duration) *time.Duration {
	return &dur
}

func TestWithServerSelectionTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		parent                 context.Context
		serverSelectionTimeout time.Duration
		wantTimeout            time.Duration
		wantOk                 bool
	}{
		{
			name:                   "no context deadine and ssto is zero",
			parent:                 context.Background(),
			serverSelectionTimeout: 0,
			wantTimeout:            0,
			wantOk:                 false,
		},
		{
			name:                   "no context deadline and ssto is positive",
			parent:                 context.Background(),
			serverSelectionTimeout: 1,
			wantTimeout:            1,
			wantOk:                 true,
		},
		{
			name:                   "no context deadline and ssto is negative",
			parent:                 context.Background(),
			serverSelectionTimeout: -1,
			wantTimeout:            0,
			wantOk:                 false,
		},
		{
			name:                   "context deadline is zero and ssto is positive",
			parent:                 newTestContext(t, 0),
			serverSelectionTimeout: 1,
			wantTimeout:            1,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is zero and ssto is negative",
			parent:                 newTestContext(t, 0),
			serverSelectionTimeout: -1,
			wantTimeout:            0,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is negative and ssto is zero",
			parent:                 newTestContext(t, -1),
			serverSelectionTimeout: 0,
			wantTimeout:            -1,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is negative and ssto is positive",
			parent:                 newTestContext(t, -1),
			serverSelectionTimeout: 1,
			wantTimeout:            1,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is negative and ssto is negative",
			parent:                 newTestContext(t, -1),
			serverSelectionTimeout: -1,
			wantTimeout:            -1,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is positive and ssto is zero",
			parent:                 newTestContext(t, 1),
			serverSelectionTimeout: 0,
			wantTimeout:            1,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is positive and equal to ssto",
			parent:                 newTestContext(t, 1),
			serverSelectionTimeout: 1,
			wantTimeout:            1,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is positive lt ssto",
			parent:                 newTestContext(t, 1),
			serverSelectionTimeout: 2,
			wantTimeout:            2,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is positive gt ssto",
			parent:                 newTestContext(t, 2),
			serverSelectionTimeout: 1,
			wantTimeout:            2,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is positive and ssto is negative",
			parent:                 newTestContext(t, -1),
			serverSelectionTimeout: -1,
			wantTimeout:            1,
			wantOk:                 true,
		},
	}

	for _, test := range tests {
		test := test // Capture the range variable

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := WithServerSelectionTimeout(test.parent, test.serverSelectionTimeout)
			t.Cleanup(cancel)

			deadline, gotOk := ctx.Deadline()
			assert.Equal(t, test.wantOk, gotOk)

			if gotOk {
				delta := time.Until(deadline) - test.wantTimeout
				tolerance := 5 * time.Millisecond

				assert.True(t, delta > -1*tolerance, "expected delta=%d > %d", delta, -1*tolerance)
				assert.True(t, delta <= tolerance, "expected delta=%d <= %d", delta, tolerance)
			}
		})
	}
}

func TestWithChangeStreamNextContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		parent      context.Context
		timeout     *time.Duration
		wantTimeout time.Duration
		wantOk      bool
	}{
		{
			name:        "no context deadline and nil timeout",
			parent:      context.Background(),
			timeout:     nil,
			wantTimeout: 0,
			wantOk:      false,
		},
		{
			name:        "no context deadline and zero timeout",
			parent:      context.Background(),
			timeout:     newDurPtr(0),
			wantTimeout: 0,
			wantOk:      false,
		},
		{
			name:        "no context deadline and positive timeout",
			parent:      context.Background(),
			timeout:     newDurPtr(1),
			wantTimeout: 1,
			wantOk:      true,
		},
		{
			name:        "no context deadline and negative timeout",
			parent:      context.Background(),
			timeout:     newDurPtr(-1),
			wantTimeout: 0,
			wantOk:      false,
		},
		{
			name:        "context deadline and nil timeout",
			parent:      newTestContext(t, 1),
			timeout:     nil,
			wantTimeout: 1,
			wantOk:      true,
		},
		{
			name:        "context deadline and positive timeout",
			parent:      newTestContext(t, 1),
			timeout:     newDurPtr(2),
			wantTimeout: 1,
			wantOk:      true,
		},
	}

	for _, test := range tests {
		test := test // Capture the range variable

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := WithChangeStreamNextContext(test.parent, test.timeout)
			t.Cleanup(cancel)

			deadline, gotOk := ctx.Deadline()
			assert.Equal(t, test.wantOk, gotOk)

			if gotOk {
				delta := time.Until(deadline) - test.wantTimeout
				tolerance := 5 * time.Millisecond

				assert.True(t, delta > -1*tolerance, "expected delta=%d > %d", delta, -1*tolerance)
				assert.True(t, delta <= tolerance, "expected delta=%d <= %d", delta, tolerance)
			}

			assert.True(t, IsWithoutMaxTime(ctx))
		})
	}
}

func TestValidChangeStreamTimeouts(t *testing.T) {
	t.Parallel()

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
		{
			name:            "no context deadline and maxAwaitTime with zero timeout",
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

			got := ValidChangeStreamTimeouts(test.parent, test.maxAwaitTimeout, test.timeout)
			assert.Equal(t, test.want, got)
		})
	}
}
