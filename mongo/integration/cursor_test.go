// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	errorCursorNotFound = 43
)

func TestCursor(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))

	// server versions 2.6 and 3.0 use OP_GET_MORE so this works on >= 3.2
	mt.RunOpts("cursor is killed on server", mtest.NewOptions().MinServerVersion("3.2"), func(mt *mtest.T) {
		initCollection(mt, mt.Coll)
		c, err := mt.Coll.Find(mtest.Background, bson.D{}, options.Find().SetBatchSize(2))
		assert.Nil(mt, err, "Find error: %v", err)

		id := c.ID()
		assert.True(mt, c.Next(mtest.Background), "expected Next true, got false")
		err = c.Close(mtest.Background)
		assert.Nil(mt, err, "Close error: %v", err)

		err = mt.DB.RunCommand(mtest.Background, bson.D{
			{"getMore", id},
			{"collection", mt.Coll.Name()},
		}).Err()
		ce := err.(mongo.CommandError)
		assert.Equal(mt, int32(errorCursorNotFound), ce.Code, "expected error code %v, got %v", errorCursorNotFound, ce.Code)
	})
}
