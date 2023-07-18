// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build go1.13
// +build go1.13

package topology

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/mongo/description"
)

var selectNone description.ServerSelectorFunc = func(description.Topology, []description.Server) ([]description.Server, error) {
	return []description.Server{}, nil
}

func TestTopologyErrors(t *testing.T) {
	t.Run("errors are wrapped", func(t *testing.T) {
		t.Run("server selection error", func(t *testing.T) {
			topo, err := New(nil)
			noerr(t, err)

			atomic.StoreInt64(&topo.state, topologyConnected)
			desc := description.Topology{
				Servers: []description.Server{},
			}
			topo.desc.Store(desc)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err = topo.SelectServer(ctx, description.WriteSelector())
			assert.True(t, errors.Is(err, context.Canceled), "expected error %v, got %v", context.Canceled, err)
		})
		t.Run("context deadline error", func(t *testing.T) {
			topo, err := New(nil)
			assert.Nil(t, err, "error creating topology: %v", err)

			var serverSelectionErr error
			callback := func(ctx context.Context) {
				selectServerCtx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
				defer cancel()

				state := newServerSelectionState(selectNone, make(<-chan time.Time))
				subCh := make(<-chan description.Topology)
				_, serverSelectionErr = topo.selectServerFromSubscription(selectServerCtx, subCh, state)
			}
			helpers.AssertSoon(t, callback, 150*time.Millisecond)
			assert.True(t, errors.Is(serverSelectionErr, context.DeadlineExceeded), "expected %v, received %v",
				context.DeadlineExceeded, serverSelectionErr)
		})
	})
}
