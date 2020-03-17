// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package session

import (
	"bytes"
	"testing"

	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/x/mongo/driver/description"
)

func TestSessionPool(t *testing.T) {
	t.Run("TestLifo", func(t *testing.T) {
		descChan := make(chan description.Topology)
		p := NewPool(descChan)
		p.timeout = 30 // Set to some arbitrarily high number greater than 1 minute.

		first, err := p.GetSession()
		assert.Nil(t, err, "GetSession error: %v", err)
		firstID := first.SessionID

		second, err := p.GetSession()
		assert.Nil(t, err, "GetSession error: %v", err)
		secondID := second.SessionID

		p.ReturnSession(first)
		p.ReturnSession(second)

		sess, err := p.GetSession()
		assert.Nil(t, err, "GetSession error: %v", err)
		nextSess, err := p.GetSession()
		assert.Nil(t, err, "GetSession error: %v", err)

		assert.True(t, bytes.Equal(sess.SessionID, secondID),
			"first session ID mismatch; expected %s, got %s", secondID, sess.SessionID)
		assert.True(t, bytes.Equal(nextSess.SessionID, firstID),
			"second session ID mismatch; expected %s, got %s", firstID, nextSess.SessionID)
	})

	t.Run("TestExpiredRemoved", func(t *testing.T) {
		descChan := make(chan description.Topology)
		p := NewPool(descChan)
		// New sessions will always become stale when returned
		p.timeout = 0

		first, err := p.GetSession()
		assert.Nil(t, err, "GetSession error: %v", err)
		firstID := first.SessionID

		second, err := p.GetSession()
		assert.Nil(t, err, "GetSession error: %v", err)
		secondID := second.SessionID

		p.ReturnSession(first)
		p.ReturnSession(second)

		sess, err := p.GetSession()
		assert.Nil(t, err, "GetSession error: %v", err)

		assert.False(t, bytes.Equal(sess.SessionID, firstID), "first expired session was not removed")
		assert.False(t, bytes.Equal(sess.SessionID, secondID), "second expired session was not removed")
	})
}
