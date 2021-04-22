// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package session

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/description"
)

func TestServerSession(t *testing.T) {
	t.Run("Expired", func(t *testing.T) {
		t.Run("non-lb mode", func(t *testing.T) {
			sess, err := newServerSession()
			assert.Nil(t, err, "newServerSession error: %v", err)

			// The session should be expired if timeoutMinutes is 0 or if its last used time is too old.
			assert.True(t, sess.expired(topologyDescription{}), "expected session to be expired when timeoutMinutes=0")
			sess.LastUsed = time.Now().Add(-30 * time.Minute)
			topoDesc := topologyDescription{timeoutMinutes: 30}
			assert.True(t, sess.expired(topoDesc), "expected session to be expired when timeoutMinutes=30")
		})
		t.Run("lb mode", func(t *testing.T) {
			sess, err := newServerSession()
			assert.Nil(t, err, "newServerSession error: %v", err)

			// The session should never be considered expired.
			topoDesc := topologyDescription{kind: description.LoadBalanced}
			assert.False(t, sess.expired(topoDesc), "session reported that it was expired in LB mode with timeoutMinutes=0")

			sess.LastUsed = time.Now().Add(-30 * time.Minute)
			topoDesc.timeoutMinutes = 10
			assert.False(t, sess.expired(topoDesc), "session reported that it was expired in LB mode with timeoutMinutes=10")
		})
	})
}
