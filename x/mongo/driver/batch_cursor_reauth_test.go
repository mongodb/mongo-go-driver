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
	type errorReply struct {
		OK   int32 `bson:"ok"`
		Code int32 `bson:"code"`
	}

	type cursor struct {
		ID        int64    `bson:"id"`
		NS        string   `bson:"ns"`
		NextBatch []bson.M `bson:"nextBatch"`
	}

	type cursorReply struct {
		OK     int32  `bson:"ok"`
		Cursor cursor `bson:"cursor"`
	}

	// The getMore fails with ReauthenticationRequired (391), then the retry
	// returns one document and closes the cursor (ID 0).
	replies := []any{
		errorReply{OK: 0, Code: 391},
		cursorReply{OK: 1, Cursor: cursor{ID: 0, NS: "db.coll", NextBatch: []bson.M{{"x": 2}}}},
	}

	// newCursor returns an open cursor whose first batch has been read.
	newCursor := func(t *testing.T, auth driver.Authenticator) *driver.BatchCursor {
		t.Helper()

		// The mock deployment takes bson.D, so round-trip each reply through
		// BSON.
		docs := make([]bson.D, len(replies))
		for i, reply := range replies {
			b, err := bson.Marshal(reply)
			require.NoError(t, err, "failed marshaling reply")
			require.NoError(t, bson.Unmarshal(b, &docs[i]), "failed unmarshaling reply")
		}

		firstDoc := bsoncore.NewDocumentBuilder().AppendInt32("x", 1).Build()
		firstBatch := bsoncore.BuildArray(nil, bsoncore.Value{Type: bsoncore.TypeEmbeddedDocument, Data: firstDoc})

		bc, err := driver.NewBatchCursor(
			driver.CursorResponse{
				Server:     drivertest.NewMockDeployment(docs...),
				Desc:       drivertest.MockDescription,
				FirstBatch: &bsoncore.Iterator{List: firstBatch},
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
		auth := &countingAuthenticator{}
		bc := newCursor(t, auth)

		assert.True(t, bc.Next(context.Background()), "expected the getMore batch, got error %v", bc.Err())
		assert.NoError(t, bc.Err(), "unexpected getMore error")
		assert.Equal(t, 1, auth.reauths, "expected one reauthentication")
	})

	t.Run("returns the error without an authenticator", func(t *testing.T) {
		bc := newCursor(t, nil)

		assert.False(t, bc.Next(context.Background()), "expected getMore to fail")
		var driverErr driver.Error
		require.ErrorAs(t, bc.Err(), &driverErr, "expected a driver.Error")
		assert.Equal(t, int32(391), driverErr.Code, "expected ReauthenticationRequired")
	})
}
