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
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestSDAMErrorHandling(t *testing.T) {
	mt := mtest.New(t, noClientOpts)
	baseClientOpts := func() *options.ClientOptions {
		return options.Client().
			ApplyURI(mt.ConnString()).
			SetRetryWrites(false).
			SetPoolMonitor(poolMonitor).
			SetWriteConcern(mtest.MajorityWc)
	}
	baseMtOpts := func() *mtest.Options {
		mtOpts := mtest.NewOptions().
			Topologies(mtest.ReplicaSet). // Don't run on sharded clusters to avoid complexity of sharded failpoints.
			MinServerVersion("4.0").      // 4.0+ is required to use failpoints on replica sets.
			ClientOptions(baseClientOpts())

		if mt.TopologyKind() == mtest.Sharded {
			// Pin to a single mongos because the tests use failpoints.
			mtOpts.ClientType(mtest.Pinned)
		}
		return mtOpts
	}

	// Set min server version of 4.4 because the during-handshake tests use failpoint features introduced in 4.4 like
	// blockConnection and appName.
	mt.RunOpts("before handshake completes", baseMtOpts().Auth(true).MinServerVersion("4.4"), func(mt *mtest.T) {
		mt.RunOpts("network errors", noClientOpts, func(mt *mtest.T) {
			mt.Run("pool cleared on network timeout", func(mt *mtest.T) {
				// Assert that the pool is cleared when a connection created by an application operation thread
				// encounters a network timeout during handshaking. Unlike the non-timeout test below, we only test
				// connections created in the foreground for timeouts because connections created by the pool
				// maintenance routine can't be timed out using a context.

				appName := "authNetworkTimeoutTest"
				// Set failpoint on saslContinue instead of saslStart because saslStart isn't done when using
				// speculative auth.
				mt.SetFailPoint(mtest.FailPoint{
					ConfigureFailPoint: "failCommand",
					Mode: mtest.FailPointMode{
						Times: 1,
					},
					Data: mtest.FailPointData{
						FailCommands:    []string{"saslContinue"},
						BlockConnection: true,
						BlockTimeMS:     150,
						AppName:         appName,
					},
				})

				// Reset the client with the appName specified in the failpoint.
				clientOpts := options.Client().
					SetAppName(appName).
					SetRetryWrites(false).
					SetPoolMonitor(poolMonitor)
				mt.ResetClient(clientOpts)
				clearPoolChan()

				// The saslContinue blocks for 150ms so run the InsertOne with a 100ms context to cause a network
				// timeout during auth and assert that the pool was cleared.
				timeoutCtx, cancel := context.WithTimeout(mtest.Background, 100*time.Millisecond)
				defer cancel()
				_, err := mt.Coll.InsertOne(timeoutCtx, bson.D{{"test", 1}})
				assert.NotNil(mt, err, "expected InsertOne error, got nil")
				assert.True(mt, isPoolCleared(), "expected pool to be cleared but was not")
			})
			mt.RunOpts("pool cleared on non-timeout network error", noClientOpts, func(mt *mtest.T) {
				mt.Run("background", func(mt *mtest.T) {
					// Assert that the pool is cleared when a connection created by the background pool maintenance
					// routine encounters a non-timeout network error during handshaking.
					appName := "authNetworkErrorTestBackground"

					mt.SetFailPoint(mtest.FailPoint{
						ConfigureFailPoint: "failCommand",
						Mode: mtest.FailPointMode{
							Times: 1,
						},
						Data: mtest.FailPointData{
							FailCommands:    []string{"saslContinue"},
							CloseConnection: true,
							AppName:         appName,
						},
					})

					clientOpts := options.Client().
						SetAppName(appName).
						SetMinPoolSize(5).
						SetPoolMonitor(poolMonitor)
					mt.ResetClient(clientOpts)
					clearPoolChan()

					time.Sleep(200 * time.Millisecond)
					assert.True(mt, isPoolCleared(), "expected pool to be cleared but was not")
				})
				mt.Run("foreground", func(mt *mtest.T) {
					// Assert that the pool is cleared when a connection created by an application thread connection
					// checkout encounters a non-timeout network error during handshaking.
					appName := "authNetworkErrorTestForeground"

					mt.SetFailPoint(mtest.FailPoint{
						ConfigureFailPoint: "failCommand",
						Mode: mtest.FailPointMode{
							Times: 1,
						},
						Data: mtest.FailPointData{
							FailCommands:    []string{"saslContinue"},
							CloseConnection: true,
							AppName:         appName,
						},
					})

					clientOpts := options.Client().
						SetAppName(appName).
						SetPoolMonitor(poolMonitor)
					mt.ResetClient(clientOpts)
					clearPoolChan()

					_, err := mt.Coll.InsertOne(mtest.Background, bson.D{{"x", 1}})
					assert.NotNil(mt, err, "expected InsertOne error, got nil")
					assert.True(mt, isPoolCleared(), "expected pool to be cleared but was not")
				})
			})
		})
	})
	mt.RunOpts("after handshake completes", baseMtOpts(), func(mt *mtest.T) {
		mt.RunOpts("network errors", noClientOpts, func(mt *mtest.T) {
			mt.Run("pool cleared on non-timeout network error", func(mt *mtest.T) {
				clearPoolChan()
				mt.SetFailPoint(mtest.FailPoint{
					ConfigureFailPoint: "failCommand",
					Mode: mtest.FailPointMode{
						Times: 1,
					},
					Data: mtest.FailPointData{
						FailCommands:    []string{"insert"},
						CloseConnection: true,
					},
				})

				_, err := mt.Coll.InsertOne(mtest.Background, bson.D{{"test", 1}})
				assert.NotNil(mt, err, "expected InsertOne error, got nil")
				assert.True(mt, isPoolCleared(), "expected pool to be cleared but was not")
			})
			mt.Run("pool not cleared on timeout network error", func(mt *mtest.T) {
				clearPoolChan()

				_, err := mt.Coll.InsertOne(mtest.Background, bson.D{{"x", 1}})
				assert.Nil(mt, err, "InsertOne error: %v", err)

				filter := bson.M{
					"$where": "function() { sleep(1000); return false; }",
				}
				timeoutCtx, cancel := context.WithTimeout(mtest.Background, 100*time.Millisecond)
				defer cancel()
				_, err = mt.Coll.Find(timeoutCtx, filter)
				assert.NotNil(mt, err, "expected Find error, got %v", err)

				assert.False(mt, isPoolCleared(), "expected pool to not be cleared but was")
			})
		})
		mt.RunOpts("server errors", noClientOpts, func(mt *mtest.T) {
			// Integration tests for the SDAM error handling code path for errors in server response documents. These
			// errors can be part of the top-level document in ok:0 responses or in a nested writeConcernError document.

			// On 4.4, some state change errors include a topologyVersion field. Because we're triggering these errors
			// via failCommand, the topologyVersion does not actually change as it would in an actual state change.
			// This causes the SDAM error handling code path to think we've already handled this state change and
			// ignore the error because it's stale. To avoid this altogether, we cap the test to <= 4.2.
			serverErrorsMtOpts := baseMtOpts().
				MinServerVersion("4.0"). // failCommand support
				MaxServerVersion("4.2").
				ClientOptions(baseClientOpts().SetRetryWrites(false))

			testCases := []struct {
				name      string
				errorCode int32

				// For shutdown errors, the pool is always cleared. For non-shutdown errors, the pool is only cleared
				// for pre-4.2 servers.
				isShutdownError bool
			}{
				// "node is recovering" errors
				{"InterruptedAtShutdown", 11600, true},
				{"InterruptedDueToReplStateChange, not shutdown", 11602, false},
				{"NotMasterOrSecondary", 13436, false},
				{"PrimarySteppedDown", 189, false},
				{"ShutdownInProgress", 91, true},

				// "not master" errors
				{"NotMaster", 10107, false},
				{"NotMasterNoSlaveOk", 13435, false},
			}
			for _, tc := range testCases {
				mt.RunOpts(fmt.Sprintf("command error - %s", tc.name), serverErrorsMtOpts, func(mt *mtest.T) {
					clearPoolChan()

					// Cause the next insert to fail with an ok:0 response.
					fp := mtest.FailPoint{
						ConfigureFailPoint: "failCommand",
						Mode: mtest.FailPointMode{
							Times: 1,
						},
						Data: mtest.FailPointData{
							FailCommands: []string{"insert"},
							ErrorCode:    tc.errorCode,
						},
					}
					mt.SetFailPoint(fp)

					runServerErrorsTest(mt, tc.isShutdownError)
				})
				mt.RunOpts(fmt.Sprintf("write concern error - %s", tc.name), serverErrorsMtOpts, func(mt *mtest.T) {
					clearPoolChan()

					// Cause the next insert to fail with a write concern error.
					fp := mtest.FailPoint{
						ConfigureFailPoint: "failCommand",
						Mode: mtest.FailPointMode{
							Times: 1,
						},
						Data: mtest.FailPointData{
							FailCommands: []string{"insert"},
							WriteConcernError: &mtest.WriteConcernErrorData{
								Code: tc.errorCode,
							},
						},
					}
					mt.SetFailPoint(fp)

					runServerErrorsTest(mt, tc.isShutdownError)
				})
			}
		})
	})
}

func runServerErrorsTest(mt *mtest.T, isShutdownError bool) {
	mt.Helper()

	_, err := mt.Coll.InsertOne(mtest.Background, bson.D{{"x", 1}})
	assert.NotNil(mt, err, "expected InsertOne error, got nil")

	// The pool should always be cleared for shutdown errors, regardless of server version.
	if isShutdownError {
		assert.True(mt, isPoolCleared(), "expected pool to be cleared, but was not")
		return
	}

	// For non-shutdown errors, the pool is only cleared if the error is from a pre-4.2 server.
	wantCleared := mtest.CompareServerVersions(mt.ServerVersion(), "4.2") < 0
	gotCleared := isPoolCleared()
	assert.Equal(mt, wantCleared, gotCleared, "expected pool to be cleared: %v; pool was cleared: %v",
		wantCleared, gotCleared)
}
