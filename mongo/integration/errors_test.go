// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// +build go1.13

package integration

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestErrors(t *testing.T) {
	mt := mtest.New(t, noClientOpts)
	defer mt.Close()

	mt.RunOpts("errors are wrapped", noClientOpts, func(mt *mtest.T) {
		mt.Run("network error during application operation", func(mt *mtest.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := mt.Client.Ping(ctx, mtest.PrimaryRp)
			assert.True(mt, errors.Is(err, context.Canceled), "expected error %v, got %v", context.Canceled, err)
		})

		authOpts := mtest.NewOptions().Auth(true).Topologies(mtest.ReplicaSet, mtest.Single).MinServerVersion("4.0")
		mt.RunOpts("network error during auth", authOpts, func(mt *mtest.T) {
			mt.SetFailPoint(mtest.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode: mtest.FailPointMode{
					Times: 1,
				},
				Data: mtest.FailPointData{
					// Set the fail point for saslContinue rather than saslStart because the driver will use speculative
					// auth on 4.4+ so there won't be an explicit saslStart command.
					FailCommands:    []string{"saslContinue"},
					CloseConnection: true,
				},
			})

			client, err := mongo.Connect(mtest.Background, options.Client().ApplyURI(mt.ConnString()))
			assert.Nil(mt, err, "Connect error: %v", err)
			defer client.Disconnect(mtest.Background)

			// A connection getting closed should manifest as an io.EOF error.
			err = client.Ping(mtest.Background, mtest.PrimaryRp)
			assert.True(mt, errors.Is(err, io.EOF), "expected error %v, got %v", io.EOF, err)
		})
	})

	mt.RunOpts("network timeouts", noClientOpts, func(mt *mtest.T) {
		mt.Run("context timeouts return DeadlineExceeded", func(mt *mtest.T) {
			_, err := mt.Coll.InsertOne(mtest.Background, bson.D{{"x", 1}})
			assert.Nil(mt, err, "InsertOne error: %v", err)

			mt.ClearEvents()
			filter := bson.M{
				"$where": "function() { sleep(1000); return false; }",
			}
			timeoutCtx, cancel := context.WithTimeout(mtest.Background, 100*time.Millisecond)
			defer cancel()
			_, err = mt.Coll.Find(timeoutCtx, filter)

			evt := mt.GetStartedEvent()
			assert.Equal(mt, "find", evt.CommandName, "expected command 'find', got %q", evt.CommandName)
			assert.True(mt, errors.Is(err, context.DeadlineExceeded),
				"errors.Is failure: expected error %v to be %v", err, context.DeadlineExceeded)
		})

		mt.Run("socketTimeoutMS timeouts return network errors", func(mt *mtest.T) {
			_, err := mt.Coll.InsertOne(mtest.Background, bson.D{{"x", 1}})
			assert.Nil(mt, err, "InsertOne error: %v", err)

			// Reset the test client to have a 100ms socket timeout. We do this here rather than passing it in as a
			// test option using mt.RunOpts because that could cause the collection creation or InsertOne to fail.
			resetClientOpts := options.Client().
				SetSocketTimeout(100 * time.Millisecond)
			mt.ResetClient(resetClientOpts)

			mt.ClearEvents()
			filter := bson.M{
				"$where": "function() { sleep(1000); return false; }",
			}
			_, err = mt.Coll.Find(mtest.Background, filter)

			evt := mt.GetStartedEvent()
			assert.Equal(mt, "find", evt.CommandName, "expected command 'find', got %q", evt.CommandName)

			assert.False(mt, errors.Is(err, context.DeadlineExceeded),
				"errors.Is failure: expected error %v to not be %v", err, context.DeadlineExceeded)
			var netErr net.Error
			ok := errors.As(err, &netErr)
			assert.True(mt, ok, "errors.As failure: expected error %v to be a net.Error", err)
			assert.True(mt, netErr.Timeout(), "expected error %v to be a network timeout", err)
		})
	})
}
