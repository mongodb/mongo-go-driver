// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// +build go1.13

package topology

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/x/mongo/driver/description"
)

func TestTopologyErrors(t *testing.T) {
	t.Run("errors are wrapped", func(t *testing.T) {
		t.Run("server selection error", func(t *testing.T) {
			topo, err := New()
			noerr(t, err)

			topo.cfg.cs.HeartbeatInterval = time.Minute
			atomic.StoreInt32(&topo.connectionstate, connected)
			desc := description.Topology{
				Servers: []description.Server{},
			}
			topo.desc.Store(desc)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err = topo.SelectServer(ctx, description.WriteSelector())
			assert.True(t, errors.Is(err, context.Canceled), "expected error %v, got %v", context.Canceled, err)
		})
	})
}
