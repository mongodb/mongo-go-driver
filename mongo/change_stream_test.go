// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestChangeStream(t *testing.T) {
	t.Run("nil cursor", func(t *testing.T) {
		cs := &ChangeStream{client: &Client{}}

		id := cs.ID()
		assert.Equal(t, int64(0), id, "expected ID 0, got %v", id)
		assert.False(t, cs.Next(bgCtx), "expected Next to return false, got true")
		err := cs.Decode(nil)
		assert.Equal(t, ErrNilCursor, err, "expected error %v, got %v", ErrNilCursor, err)
		err = cs.Err()
		assert.Nil(t, err, "change stream error: %v", err)
		err = cs.Close(bgCtx)
		assert.Nil(t, err, "Close error: %v", err)
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

			cs := &ChangeStream{
				options: &options.ChangeStreamOptions{
					MaxAwaitTime: test.maxAwaitTimeout,
				},
				client: &Client{
					timeout: test.timeout,
				},
			}

			got := validChangeStreamTimeouts(test.parent, cs)
			assert.Equal(t, test.want, got)
		})
	}
}
