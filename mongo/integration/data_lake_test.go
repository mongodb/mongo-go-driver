// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestAtlasDataLake(t *testing.T) {
	// Prose tests against Atlas Data Lake.

	mt := mtest.New(t, mtest.NewOptions().AtlasDataLake(true).CreateClient(false))
	defer mt.Close()
	getMtOpts := func() *mtest.Options {
		return mtest.NewOptions().CollectionName("driverdata")
	}

	mt.RunOpts("killCursors", getMtOpts(), func(mt *mtest.T) {
		// Test that the killCursors sent internally when closing a cursor uses the correct namespace and cursor ID.

		// Run a Find and get the cursor ID and namespace returned by the server. Use a batchSize of 2 to force the
		// server to keep the cursor open.
		cursor, err := mt.Coll.Find(context.Background(), bson.D{}, options.Find().SetBatchSize(2))
		assert.Nil(mt, err, "Find error: %v", err)
		findEvt := mt.GetSucceededEvent()
		assert.Equal(mt, "find", findEvt.CommandName, "expected command name %q, got %q", "find", findEvt.CommandName)
		expectedID := findEvt.Reply.Lookup("cursor", "id").Int64()
		expectedNS := findEvt.Reply.Lookup("cursor", "ns").StringValue()

		// Close the cursor, forcing a killCursors command.
		mt.ClearEvents()
		err = cursor.Close(context.Background())
		assert.Nil(mt, err, "Close error: %v", err)

		// Extract information from the killCursors started event and assert that it sent the right cursor ID and ns.
		killCursorsEvt := mt.GetStartedEvent()
		assert.Equal(mt, "killCursors", killCursorsEvt.CommandName, "expected command name %q, got %q", "killCursors",
			killCursorsEvt.CommandName)
		actualID := killCursorsEvt.Command.Lookup("cursors", "0").Int64()
		killCursorsDB := killCursorsEvt.Command.Lookup("$db").StringValue()
		killCursorsColl := killCursorsEvt.Command.Lookup("killCursors").StringValue()
		actualNS := fmt.Sprintf("%s.%s", killCursorsDB, killCursorsColl)

		assert.Equal(mt, expectedID, actualID, "expected cursor ID %v, got %v; find event %v, killCursors event %v",
			expectedID, actualID, findEvt, killCursorsEvt)
		assert.Equal(mt, expectedNS, actualNS, "expected namespace %q, got %q; find event %v, killCursors event %v",
			expectedNS, actualNS, findEvt, killCursorsEvt)

		// Extract information from the killCursors succeeded event and assert that the right cursor was killed.
		var killCursorsResponse struct {
			CursorsKilled []int64
		}
		err = bson.Unmarshal(mt.GetSucceededEvent().Reply, &killCursorsResponse)
		assert.Nil(mt, err, "error unmarshalling killCursors response: %v", err)
		expectedCursorsKilled := []int64{expectedID}
		assert.Equal(mt, expectedCursorsKilled, killCursorsResponse.CursorsKilled,
			"expected cursorsKilled array %v, got %v", expectedCursorsKilled, killCursorsResponse.CursorsKilled)
	})

	mt.RunOpts("auth settings", noClientOpts, func(mt *mtest.T) {
		// Test connectivity using different auth settings.

		testCases := []struct {
			name          string
			authMechanism string // No auth will be used if this is "".
		}{
			{"no auth", ""},
			{"scram-sha-1", "SCRAM-SHA-1"},
			{"scram-sha-256", "SCRAM-SHA-256"},
		}
		for _, tc := range testCases {
			clientOpts := getBaseClientOptions()
			if tc.authMechanism != "" {
				cred := getBaseCredential(mt)
				cred.AuthMechanism = tc.authMechanism
				clientOpts.SetAuth(cred)
			}
			mtOpts := getMtOpts().ClientOptions(clientOpts)

			mt.RunOpts(tc.name, mtOpts, func(mt *mtest.T) {
				err := mt.Client.Ping(context.Background(), mtest.PrimaryRp)
				assert.Nil(mt, err, "Ping error: %v", err)
			})
		}
	})
}

func getBaseClientOptions() *options.ClientOptions {
	opts := options.Client().ApplyURI(mtest.ClusterURI())
	return options.Client().SetHosts(opts.Hosts)
}

func getBaseCredential(mt *mtest.T) options.Credential {
	mt.Helper()

	cred := options.Client().ApplyURI(mtest.ClusterURI()).Auth
	assert.NotNil(mt, cred, "expected options for URI %q to have a non-nil Auth field", mtest.ClusterURI())
	return *cred
}
