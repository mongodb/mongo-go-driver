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

func TestWithServerSelectionTimeout(t *testing.T) {
	t.Parallel()

	newContext := func(t *testing.T, timeout time.Duration) context.Context {
		t.Helper()

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		t.Cleanup(cancel)

		return ctx
	}

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
			parent:                 newContext(t, 0),
			serverSelectionTimeout: 1,
			wantTimeout:            1,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is zero and ssto is negative",
			parent:                 newContext(t, 0),
			serverSelectionTimeout: -1,
			wantTimeout:            0,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is negative and ssto is zero",
			parent:                 newContext(t, -1),
			serverSelectionTimeout: 0,
			wantTimeout:            -1,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is negative and ssto is positive",
			parent:                 newContext(t, -1),
			serverSelectionTimeout: 1,
			wantTimeout:            1,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is negative and ssto is negative",
			parent:                 newContext(t, -1),
			serverSelectionTimeout: -1,
			wantTimeout:            -1,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is positive and ssto is zero",
			parent:                 newContext(t, 1),
			serverSelectionTimeout: 0,
			wantTimeout:            1,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is positive and equal to ssto",
			parent:                 newContext(t, 1),
			serverSelectionTimeout: 1,
			wantTimeout:            1,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is positive lt ssto",
			parent:                 newContext(t, 1),
			serverSelectionTimeout: 2,
			wantTimeout:            2,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is positive gt ssto",
			parent:                 newContext(t, 2),
			serverSelectionTimeout: 1,
			wantTimeout:            2,
			wantOk:                 true,
		},
		{
			name:                   "context deadline is positive and ssto is negative",
			parent:                 newContext(t, -1),
			serverSelectionTimeout: -1,
			wantTimeout:            1,
			wantOk:                 true,
		},
	}

	for _, test := range tests {
		test := test

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
