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
	"testing"

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

		authOpts := mtest.NewOptions().Auth(false).Topologies(mtest.ReplicaSet, mtest.Single).MinServerVersion("4.0")
		mt.RunOpts("network error during auth", authOpts, func(mt *mtest.T) {
			mt.SetFailPoint(mtest.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode: mtest.FailPointMode{
					Times: 1,
				},
				Data: mtest.FailPointData{
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
}
