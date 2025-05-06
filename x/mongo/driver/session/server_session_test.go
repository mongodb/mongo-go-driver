// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package session

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
)

func TestServerSession(t *testing.T) {
	int64ToPtr := func(i64 int64) *int64 { return &i64 }

	t.Run("Expired", func(t *testing.T) {
		t.Run("non-lb mode", func(t *testing.T) {
			sess, err := newServerSession()
			assert.Nil(t, err, "newServerSession error: %v", err)

			// The session should be expired if timeoutMinutes is 0 or if its last used time is too old.
			assert.True(t, sess.expired(topologyDescription{}), "expected session to be expired when timeoutMinutes=0")
			sess.LastUsed = time.Now().Add(-30 * time.Minute)
			topoDesc := topologyDescription{timeoutMinutes: int64ToPtr(30)}
			assert.True(t, sess.expired(topoDesc), "expected session to be expired when timeoutMinutes=30")
		})
		t.Run("lb mode", func(t *testing.T) {
			sess, err := newServerSession()
			assert.Nil(t, err, "newServerSession error: %v", err)

			// The session should never be considered expired.
			topoDesc := topologyDescription{kind: description.TopologyKindLoadBalanced}
			assert.False(t, sess.expired(topoDesc), "session reported that it was expired in LB mode with timeoutMinutes=0")

			sess.LastUsed = time.Now().Add(-30 * time.Minute)
			topoDesc.timeoutMinutes = int64ToPtr(10)
			assert.False(t, sess.expired(topoDesc), "session reported that it was expired in LB mode with timeoutMinutes=10")
		})
	})
}
