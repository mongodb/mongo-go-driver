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

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/auth/mongoawsv2"
)

func TestMongoAWS(t *testing.T) {
	auth.RegisterAuthenticatorFactory(auth.MongoDBAWS, mongoawsv2.NewAuthenticator)

	uri := os.Getenv("MONGODB_URI")
	ctx := context.Background()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err, "Connect error")

	defer func() {
		err = client.Disconnect(ctx)
		require.NoError(t, err, "Disconnect error")
	}()

	db := client.Database("aws")
	coll := db.Collection("test")
	if err = coll.FindOne(ctx, bson.D{{"x", 1}}).Err(); err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("FindOne error: %v", err)
	}
}
