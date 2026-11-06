// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mtest

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/require"
)

func TestCheckedOutSessions(t *testing.T) {
	mt := New(t, NewOptions().ClientType(Mock))
	mt.Setup()

	// A returned session is not counted, before or after a client reset.
	sess, err := mt.Client.StartSession()
	require.NoError(t, err, "starting a session should not error")
	sess.EndSession(context.Background())

	require.Equal(t, 0, mt.checkedOutSessions(), "returned session should not be counted")

	mt.ResetClient(nil)

	require.Equal(t, 0, mt.sampledSessions, "returned session should not be sampled")
	require.Equal(t, 0, mt.checkedOutSessions(), "returned session should not be counted after reset")

	// A session leaked across a client reset is counted.
	_, err = mt.Client.StartSession()
	require.NoError(t, err, "starting a session should not error")

	mt.ResetClient(nil)

	require.Equal(t, 1, mt.sampledSessions, "leaked session should be sampled")
	require.Equal(t, 1, mt.checkedOutSessions(), "leaked session should be counted")

	// Sessions on the current client are summed with the sampled sessions.
	live, err := mt.Client.StartSession()
	require.NoError(t, err, "starting a session should not error")

	require.Equal(t, 2, mt.checkedOutSessions(), "live and sampled sessions should be summed")

	live.EndSession(context.Background())
	require.Equal(t, 1, mt.checkedOutSessions(), "ending a session should decrement the count")

	// Zero the sampled sessions so the teardown assertion passes.
	mt.sampledSessions = 0
}
