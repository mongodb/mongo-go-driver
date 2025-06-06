// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"errors"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestAWS(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err, "Connect error")

	defer func() {
		err = client.Disconnect(context.Background())
		require.NoError(t, err)
	}()

	coll := client.Database("aws").Collection("test")

	err = coll.FindOne(context.Background(), bson.D{{Key: "x", Value: 1}}).Err()
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		t.Logf("FindOne error: %v", err)
	}
}
