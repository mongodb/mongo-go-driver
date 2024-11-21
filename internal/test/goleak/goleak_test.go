// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/goleak"
)

var dbName = fmt.Sprintf("goleak-%d", time.Now().Unix())

// TestGoroutineLeak creates clients with various client configurations, runs
// some operations with each one, then disconnects the client. It asserts that
// no goroutines were leaked after the client is disconnected.
func TestGoroutineLeak(t *testing.T) {
	testCases := []struct {
		desc string
		opts *options.ClientOptions
	}{
		{
			desc: "base",
			opts: options.Client(),
		},
		{
			desc: "compressors=snappy",
			opts: options.Client().SetCompressors([]string{"snappy"}),
		},
		{
			desc: "compressors=zlib",
			opts: options.Client().SetCompressors([]string{"zlib"}),
		},
		{
			desc: "compressors=zstd",
			opts: options.Client().SetCompressors([]string{"zstd"}),
		},
		{
			desc: "minPoolSize=10",
			opts: options.Client().SetMinPoolSize(10),
		},
		{
			desc: "serverMonitoringMode=poll",
			opts: options.Client().SetServerMonitoringMode(options.ServerMonitoringModePoll),
		},
	}

	for _, tc := range testCases {
		// These can't be run in parallel because goleak currently can't filter
		// out goroutines from other parallel subtests.
		t.Run(tc.desc, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			base := options.Client()
			if u := os.Getenv("MONGODB_URI"); u != "" {
				base.ApplyURI(u)
			}
			client, err := mongo.Connect(base, tc.opts)
			require.NoError(t, err)

			defer func() {
				err = client.Disconnect(context.Background())
				require.NoError(t, err)
			}()

			db := client.Database(dbName)
			defer func() {
				err := db.Drop(context.Background())
				require.NoError(t, err)
			}()

			coll := db.Collection(collectionName(t))

			// Start a change stream to simulate a change listener workload.
			cs, err := coll.Watch(context.Background(), mongo.Pipeline{})
			require.NoError(t, err)
			defer cs.Close(context.Background())

			// Run some Insert and FindOne operations to simulate a writing and
			// reading workload. Run 50 iterations to increase the probability
			// that a goroutine leak will happen if a problem exists.
			for i := 0; i < 50; i++ {
				_, err = coll.InsertOne(context.Background(), bson.M{"x": 123})
				require.NoError(t, err)

				var res bson.D
				err = coll.FindOne(context.Background(), bson.D{}).Decode(&res)
				require.NoError(t, err)
			}

			// Intentionally cause some timeouts. Ignore any errors.
			for i := 0; i < 50; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Microsecond)
				coll.FindOne(ctx, bson.D{}).Err()
				cancel()
			}

			// Finish simulating the change listener workload. Use "Next" to
			// fetch at least one change stream document batch and decode the
			// first document.
			cs.Next(context.Background())
			var res bson.D
			err = cs.Decode(&res)
			require.NoError(t, err)
		})
	}
}

func collectionName(t *testing.T) string {
	return fmt.Sprintf("%s-%d", t.Name(), time.Now().Unix())
}
