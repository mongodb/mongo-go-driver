// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package drivertest

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestOPMSGMockDeployment(t *testing.T) {
	md := NewMockDeployment()

	opts := options.Client()
	opts.Deployment = md

	client, err := mongo.Connect(opts)

	t.Run("NewMockDeployment connect to client", func(t *testing.T) {
		require.NoError(t, err, "unexpected error from connect to mockdeployment")
	})
	t.Run("AddResponses with one", func(t *testing.T) {
		res := bson.D{{"ok", 1}}
		md.AddResponses(res)
		assert.NotNil(t, md.conn.responses, "expected non-nil responses")
		assert.Len(t, md.conn.responses, 1, "expected 1 response, got %v", len(md.conn.responses))
		err = client.Ping(context.Background(), nil)
		require.NoError(t, err)
	})
	t.Run("AddResponses with multiple", func(t *testing.T) {
		res1 := bson.D{{"ok", 1}}
		res2 := bson.D{{"ok", 2}}
		res3 := bson.D{{"ok", 3}}
		md.AddResponses(res1, res2, res3)
		assert.NotNil(t, md.conn.responses, "expected non-nil responses")
		assert.Len(t, md.conn.responses, 3, "expected 3 responses, got %v", len(md.conn.responses))
		err = client.Ping(context.Background(), nil)
		require.NoError(t, err)
	})
	t.Run("ClearResponses", func(t *testing.T) {
		md.ClearResponses()
		assert.NotNil(t, md.conn.responses, "expected non-nil responses")
		assert.Len(t, md.conn.responses, 0, "expected 0 responses, got %v", len(md.conn.responses))
		err = client.Ping(context.Background(), nil)
		require.Error(t, err, "expected Ping error, got nil")
	})
}
