// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver_test

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
)

type countingAuthenticator struct {
	reauths int
}

func (*countingAuthenticator) Auth(context.Context, *driver.AuthConfig) error { return nil }

func (a *countingAuthenticator) Reauth(context.Context, *driver.AuthConfig) error {
	a.reauths++
	return nil
}

func TestBatchCursorReauthenticatesGetMore(t *testing.T) {
	t.Parallel()

	newCursor := func(t *testing.T, auth driver.Authenticator) *driver.BatchCursor {
		t.Helper()

		deployment := drivertest.NewMockDeployment(
			bson.D{
				{Key: "ok", Value: 0},
				{Key: "code", Value: 391},
				{Key: "codeName", Value: "ReauthenticationRequired"},
				{Key: "errmsg", Value: "reauthentication required"},
			},
			bson.D{
				{Key: "ok", Value: 1},
				{Key: "cursor", Value: bson.D{
					{Key: "id", Value: int64(0)},
					{Key: "ns", Value: "db.coll"},
					{Key: "nextBatch", Value: bson.A{bson.D{{Key: "x", Value: 2}}}},
				}},
			},
		)
		firstDoc := bsoncore.NewDocumentBuilder().AppendInt32("x", 1).Build()
		bc, err := driver.NewBatchCursor(
			driver.CursorResponse{
				Server:     deployment,
				Desc:       drivertest.MockDescription,
				FirstBatch: &bsoncore.Iterator{List: bsoncore.BuildArray(nil, bsoncore.Value{Type: bsoncore.TypeEmbeddedDocument, Data: firstDoc})},
				Database:   "db",
				Collection: "coll",
				ID:         1,
			},
			nil,
			nil,
			driver.CursorOptions{Authenticator: auth},
		)
		require.NoError(t, err, "NewBatchCursor error")
		require.True(t, bc.Next(context.Background()), "expected the first batch")
		return bc
	}

	t.Run("reauthenticates and retries getMore", func(t *testing.T) {
		t.Parallel()

		auth := &countingAuthenticator{}
		bc := newCursor(t, auth)

		assert.True(t, bc.Next(context.Background()), "expected the getMore batch, got error %v", bc.Err())
		assert.NoError(t, bc.Err(), "unexpected getMore error")
		assert.Equal(t, 1, auth.reauths, "expected one reauthentication")
	})

	t.Run("returns the error without an authenticator", func(t *testing.T) {
		t.Parallel()

		bc := newCursor(t, nil)

		assert.False(t, bc.Next(context.Background()), "expected getMore to fail")
		var driverErr driver.Error
		require.ErrorAs(t, bc.Err(), &driverErr, "expected a driver.Error")
		assert.Equal(t, int32(391), driverErr.Code, "expected ReauthenticationRequired")
	})
}
