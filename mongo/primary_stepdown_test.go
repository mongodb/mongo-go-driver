// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type serverStatus struct {
	Host        string
	Connections struct {
		TotalCreated int32 `bson:"totalCreated"`
	}
}

func getTotalCreatedConnections(db *Database) (int32, error) {
	var status serverStatus
	res := db.RunCommand(
		context.Background(),
		bson.D{{"serverStatus", 1}},
	)
	err := res.Decode(&status)
	// fmt.Println(status.Host)
	// fmt.Println(status.Connections)
	return status.Connections.TotalCreated, err
}

func TestConnectionsSurvivePrimaryStepDown(t *testing.T) {
	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip("Needs to run on a replica set")
	}
	ctx := context.Background()
	mongodbURI := testutil.ConnString(t)
	client, err := Connect(ctx, options.Client().ApplyURI(mongodbURI.String()).SetRetryWrites(false))
	require.NoError(t, err)
	db := client.Database("step-down", options.Database().SetWriteConcern(writeconcern.New(writeconcern.WMajority())))
	collName := "step-down"
	err = db.Collection(collName).Drop(ctx)
	require.NoError(t, err)

	err = db.RunCommand(
		context.Background(),
		bson.D{{"create", collName}},
	).Err()
	require.NoError(t, err)
	coll := db.Collection(collName)

	serverVersion, err := getServerVersion(db)
	require.NoError(t, err)
	adminDB := client.Database("admin")

	t.Run("getMore_iteration", func(t *testing.T) {
		if compareVersions(t, serverVersion, "4.2") < 0 {
			t.Skip("Needs server version >= 4.2")
		}
		initCollection(t, coll)
		cur, err := coll.Find(ctx, bson.D{}, options.Find().SetBatchSize(2))
		ok := cur.Next(ctx)
		require.True(t, ok)

		origConns, err := getTotalCreatedConnections(db)
		require.NoError(t, err)

		err = adminDB.RunCommand(
			context.Background(),
			bson.D{{"replSetStepDown", 5}, {"force", true}},
			options.RunCmd().SetReadPreference(readpref.Primary()),
		).Err()
		require.NoError(t, err)

		ok = cur.Next(ctx)
		require.True(t, ok)

		newConns, err := getTotalCreatedConnections(db)
		require.NoError(t, err)

		require.Equal(t, origConns, newConns)
	})
	t.Run("notMaster_keep_pool", func(t *testing.T) {
		if compareVersions(t, serverVersion, "4.2") < 0 {
			t.Skip("Needs server version >= 4.2")
		}

		err = adminDB.RunCommand(
			ctx,
			bson.D{{"configureFailPoint", "failCommand"},
				{"mode", bson.D{{"times", 1}}},
				{"data", bson.D{{"failCommands", bson.A{"insert"}}, {"errorCode", 10107}}}},
		).Err()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, adminDB.RunCommand(ctx, bson.D{
				{"configureFailPoint", "failCommand"},
				{"mode", "off"},
			}).Err())
		}()

		origConns, err := getTotalCreatedConnections(db)
		require.NoError(t, err)

		_, err = coll.InsertOne(ctx, bson.D{{"test", 1}})
		require.Error(t, err)

		cerr, ok := err.(CommandError)
		require.True(t, ok)
		require.Equal(t, int32(10107), cerr.Code)

		_, err = coll.InsertOne(ctx, bson.D{{"test", 1}})
		require.NoError(t, err)

		newConns, err := getTotalCreatedConnections(db)
		require.NoError(t, err)

		require.Equal(t, origConns, newConns)
	})
	t.Run("notMaster_reset_pool", func(t *testing.T) {
		if compareVersions(t, serverVersion, "4.0") != 0 {
			t.Skip("Needs server version 4.0")
		}
		origConns, err := getTotalCreatedConnections(db)
		require.NoError(t, err)

		err = adminDB.RunCommand(
			ctx,
			bson.D{{"configureFailPoint", "failCommand"},
				{"mode", bson.D{{"times", 1}}},
				{"data", bson.D{{"failCommands", bson.A{"insert"}}, {"errorCode", 10107}}}},
		).Err()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, adminDB.RunCommand(ctx, bson.D{
				{"configureFailPoint", "failCommand"},
				{"mode", "off"},
			}).Err())
		}()
		_, err = coll.InsertOne(ctx, bson.D{{"test", 1}})
		require.Error(t, err)

		cerr, ok := err.(CommandError)
		require.True(t, ok)
		require.Equal(t, int32(10107), cerr.Code)

		newConns, err := getTotalCreatedConnections(db)
		require.NoError(t, err)

		require.Equal(t, origConns+1, newConns)
	})
	t.Run("shutdownInProgress_reset_pool", func(t *testing.T) {
		if compareVersions(t, serverVersion, "4.0") < 0 {
			t.Skip("Needs server version >= 4.0")
		}
		origConns, err := getTotalCreatedConnections(db)
		require.NoError(t, err)
		err = adminDB.RunCommand(
			ctx,
			bson.D{{"configureFailPoint", "failCommand"},
				{"mode", bson.D{{"times", 1}}},
				{"data", bson.D{{"failCommands", bson.A{"insert"}}, {"errorCode", 91}}}},
		).Err()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, adminDB.RunCommand(ctx, bson.D{
				{"configureFailPoint", "failCommand"},
				{"mode", "off"},
			}).Err())
		}()

		_, err = coll.InsertOne(ctx, bson.D{{"test", 1}})
		require.Error(t, err)

		cerr, ok := err.(CommandError)
		require.True(t, ok)
		require.Equal(t, int32(91), cerr.Code)

		newConns, err := getTotalCreatedConnections(db)
		require.NoError(t, err)

		require.Equal(t, origConns+1, newConns)
	})
	t.Run("interruptedAtShutdown_reset_pool", func(t *testing.T) {
		if compareVersions(t, serverVersion, "4.0") < 0 {
			t.Skip("Needs server version >= 4.0")
		}
		origConns, err := getTotalCreatedConnections(db)
		require.NoError(t, err)
		err = adminDB.RunCommand(
			ctx,
			bson.D{{"configureFailPoint", "failCommand"},
				{"mode", bson.D{{"times", 1}}},
				{"data", bson.D{{"failCommands", bson.A{"insert"}}, {"errorCode", 11600}}}},
		).Err()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, adminDB.RunCommand(ctx, bson.D{
				{"configureFailPoint", "failCommand"},
				{"mode", "off"},
			}).Err())
		}()

		_, err = coll.InsertOne(ctx, bson.D{{"test", 1}})
		require.Error(t, err)

		cerr, ok := err.(CommandError)
		require.True(t, ok)
		require.Equal(t, int32(11600), cerr.Code)

		newConns, err := getTotalCreatedConnections(db)
		require.NoError(t, err)

		require.Equal(t, origConns+1, newConns)
	})
}
