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

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/ptrutil"
)

func newTestContext(t *testing.T, timeout time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)

	return ctx
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
				tolerance := 10 * time.Millisecond

				assert.True(t, delta > -1*tolerance, "expected delta=%d > %d", delta, -1*tolerance)
				assert.True(t, delta <= tolerance, "expected delta=%d <= %d", delta, tolerance)
			}
		})
	}
}

func TestWithTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		parent       context.Context
		timeout      *time.Duration
		wantTimeout  time.Duration
		wantDeadline bool
		wantValues   []any
	}{
		{
			name:         "deadline set with non-zero timeout",
			parent:       newTestContext(t, 1),
			timeout:      ptrutil.Ptr(time.Duration(2)),
			wantTimeout:  1,
			wantDeadline: true,
			wantValues:   []any{},
		},
		{
			name:         "deadline set with zero timeout",
			parent:       newTestContext(t, 1),
			timeout:      ptrutil.Ptr(time.Duration(0)),
			wantTimeout:  1,
			wantDeadline: true,
			wantValues:   []any{},
		},
		{
			name:         "deadline set with nil timeout",
			parent:       newTestContext(t, 1),
			timeout:      nil,
			wantTimeout:  1,
			wantDeadline: true,
			wantValues:   []any{},
		},
		{
			name:         "deadline unset with non-zero timeout",
			parent:       context.Background(),
			timeout:      ptrutil.Ptr(time.Duration(1)),
			wantTimeout:  1,
			wantDeadline: true,
			wantValues:   []any{},
		},
		{
			name:         "deadline unset with zero timeout",
			parent:       context.Background(),
			timeout:      ptrutil.Ptr(time.Duration(0)),
			wantTimeout:  0,
			wantDeadline: false,
			wantValues:   []any{clientLevel{}},
		},
		{
			name:         "deadline unset with nil timeout",
			parent:       context.Background(),
			timeout:      nil,
			wantTimeout:  0,
			wantDeadline: false,
			wantValues:   []any{},
		},
		{
			// If "clientLevel" has been set, but a new timeout is applied
			// to the context, then the constructed context should retain the old
			// timeout. To simplify the code, we assume the first timeout is static.
			name:         "deadline unset with non-zero timeout at clientLevel",
			parent:       context.WithValue(context.Background(), clientLevel{}, true),
			timeout:      ptrutil.Ptr(time.Duration(1)),
			wantTimeout:  0,
			wantDeadline: false,
			wantValues:   []any{},
		},
	}

	for _, test := range tests {
		test := test // Capture the range variable

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := WithTimeout(test.parent, test.timeout)
			t.Cleanup(cancel)

			deadline, gotDeadline := ctx.Deadline()
			assert.Equal(t, test.wantDeadline, gotDeadline)

			if gotDeadline {
				delta := time.Until(deadline) - test.wantTimeout
				tolerance := 10 * time.Millisecond

				assert.True(t, delta > -1*tolerance, "expected delta=%d > %d", delta, -1*tolerance)
				assert.True(t, delta <= tolerance, "expected delta=%d <= %d", delta, tolerance)
			}

			for _, wantValue := range test.wantValues {
				assert.NotNil(t, ctx.Value(wantValue), "expected context to have value %v", wantValue)
			}
		})
	}
}
