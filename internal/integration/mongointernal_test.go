// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build mongointernal

package integration

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

func TestNewSessionWithID(t *testing.T) {
	mt := mtest.New(t)

	mt.Run("can be used to pass a specific session ID to CRUD commands", func(mt *mtest.T) {
		mt.Parallel()

		// Create a session ID document, which is a BSON document with field
		// "id" containing a 16-byte UUID (binary subtype 4).
		sessionID := bson.Raw(bsoncore.NewDocumentBuilder().
			AppendBinary("id", 4, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}).
			Build())

		sess := mongo.NewSessionWithID(mt.Client, sessionID)

		ctx := mongo.NewSessionContext(context.Background(), sess)
		_, err := mt.Coll.InsertOne(ctx, bson.D{{"foo", "bar"}})
		require.NoError(mt, err)

		evt := mt.GetStartedEvent()
		val, err := evt.Command.LookupErr("lsid")
		require.NoError(mt, err, "lsid should be present in the command document")

		doc, ok := val.DocumentOK()
		require.True(mt, ok, "lsid should be a document")

		assert.Equal(mt, sessionID, doc)
	})

	mt.Run("EndSession panics", func(mt *mtest.T) {
		mt.Parallel()

		sessionID := bson.Raw(bsoncore.NewDocumentBuilder().
			AppendBinary("id", 4, []byte{}).
			Build())
		sess := mongo.NewSessionWithID(mt.Client, sessionID)

		// Use a defer-recover block to catch the expected panic and assert that
		// the recovered error is not nil.
		defer func() {
			err := recover()
			assert.NotNil(mt, err, "expected panic error to not be nil")
		}()

		sess.EndSession(context.Background())

		// We expect that calling EndSession on a Session returned by
		// NewSessionWithID panics. This code will only be reached if EndSession
		// doesn't panic.
		t.Errorf("expected EndSession to panic")
	})
}
