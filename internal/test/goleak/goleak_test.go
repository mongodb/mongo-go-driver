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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/goleak"
)

var dbName = fmt.Sprintf("goleak-%d", time.Now().Unix())

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// TestGoroutineLeak creates clients with various client configurations, runs
// some operations with each one, then disconnects the client. It asserts that
// no goroutines were leaked after the client is disconnected.
func TestGoroutineLeak(t *testing.T) {
	uri := "mongodb://localhost:27017"
	if u := os.Getenv("MONGODB_URI"); u != "" {
		uri = u
	}

	testCases := []struct {
		desc string
		opts *options.ClientOptions
	}{
		{
			desc: "base",
			opts: options.Client().ApplyURI(uri),
		},
		{
			desc: "compressors=snappy",
			opts: options.Client().ApplyURI(uri).SetCompressors([]string{"snappy"}),
		},
		{
			desc: "compressors=zlib",
			opts: options.Client().ApplyURI(uri).SetCompressors([]string{"zlib"}),
		},
		{
			desc: "compressors=zstd",
			opts: options.Client().ApplyURI(uri).SetCompressors([]string{"zstd"}),
		},
		{
			desc: "minPoolSize=10",
			opts: options.Client().ApplyURI(uri).SetMinPoolSize(10),
		},
		{
			desc: "serverMonitoringMode=poll",
			opts: options.Client().ApplyURI(uri).SetServerMonitoringMode(options.ServerMonitoringModePoll),
		},
	}
	for _, tc := range testCases {
		// These can't be run in parallel because goleak currently can't filter
		// out goroutines from other parallel subtests.
		t.Run(tc.desc, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			client, err := mongo.Connect(context.Background(), tc.opts)
			require.NoError(t, err)

			defer func() {
				err = client.Disconnect(context.Background())
				require.NoError(t, err)
			}()

			coll := client.Database(dbName).Collection(collectionName(t))
			defer func() {
				err := coll.Drop(context.Background())
				require.NoError(t, err)
			}()

			_, err = coll.InsertOne(context.Background(), bson.M{"x": 123})
			require.NoError(t, err)

			for i := 0; i < 20; i++ {
				var res bson.D
				err = coll.FindOne(context.Background(), bson.D{}).Decode(&res)
				require.NoError(t, err)
				time.Sleep(50 * time.Millisecond)
			}
		})
	}
}

func collectionName(t *testing.T) string {
	return fmt.Sprintf("%s-%d", t.Name(), time.Now().Unix())
}
