// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestCSOTProse(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))
	defer mt.Close()

	mt.Run("server selection", func(mt *mtest.T) {
		cliOpts := options.Client().ApplyURI("mongodb://invalid/?serverSelectionTimeoutMS=10")
		mtOpts := mtest.NewOptions().ClientOptions(cliOpts).CreateCollection(false)
		mt.RunOpts("serverSelectionTimeoutMS honored if timeoutMS is not set", mtOpts, func(mt *mtest.T) {
			callback := func(ctx context.Context) {
				err := mt.Client.Ping(ctx, nil)
				assert.NotNil(mt, err, "expected Ping error, got nil")
			}

			// Assert that Ping fails within 15ms due to server selection timeout.
			assert.Soon(mt, callback, 15*time.Millisecond)
		})

		cliOpts = options.Client().ApplyURI("mongodb://invalid/?timeoutMS=10&serverSelectionTimeoutMS=20")
		mtOpts = mtest.NewOptions().ClientOptions(cliOpts).CreateCollection(false)
		mt.RunOpts("timeoutMS honored for server selection if it's lower than serverSelectionTimeoutMS", mtOpts, func(mt *mtest.T) {
			callback := func(ctx context.Context) {
				err := mt.Client.Ping(ctx, nil)
				assert.NotNil(mt, err, "expected Ping error, got nil")
			}

			// Assert that Ping fails within 15ms due to timeout.
			assert.Soon(mt, callback, 15*time.Millisecond)
		})

		cliOpts = options.Client().ApplyURI("mongodb://invalid/?timeoutMS=20&serverSelectionTimeoutMS=10")
		mtOpts = mtest.NewOptions().ClientOptions(cliOpts).CreateCollection(false)
		mt.RunOpts("serverSelectionTimeoutMS honored for server selection if it's lower than timeoutMS", mtOpts, func(mt *mtest.T) {
			callback := func(ctx context.Context) {
				err := mt.Client.Ping(ctx, nil)
				assert.NotNil(mt, err, "expected Ping error, got nil")
			}

			// Assert that Ping fails within 15ms due to server selection timeout.
			assert.Soon(mt, callback, 15*time.Millisecond)
		})

		cliOpts = options.Client().ApplyURI("mongodb://invalid/?timeoutMS=0&serverSelectionTimeoutMS=10")
		mtOpts = mtest.NewOptions().ClientOptions(cliOpts).CreateCollection(false)
		mt.RunOpts("serverSelectionTimeoutMS honored for server selection if timeoutMS=0", mtOpts, func(mt *mtest.T) {
			callback := func(ctx context.Context) {
				err := mt.Client.Ping(ctx, nil)
				assert.NotNil(mt, err, "expected Ping error, got nil")
			}

			// Assert that Ping fails within 15ms due to server selection timeout.
			assert.Soon(mt, callback, 15*time.Millisecond)
		})
	})
}
