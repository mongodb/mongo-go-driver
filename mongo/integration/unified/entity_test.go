// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
)

func TestEntityMap(t *testing.T) {
	record := bson.D{{"foo", 1}}
	doc, err := bson.Marshal(record)
	assert.Nil(t, err, "error marshaling example doc %s", err)
	t.Run("bson array entity", func(t *testing.T) {
		name := "errors"
		em := newEntityMap()
		err = em.addBSONArrayEntity(name)
		assert.Nil(t, err, "expected nil error, got %s", err)
		// adding an existing bson array entity twice shouldn't error
		err = em.addBSONArrayEntity(name)
		assert.Nil(t, err, "expected nil error, got %s", err)

		em.appendBSONArrayEntity(name, doc)

		// bson array can't be retrieved until the map is closed
		_, found := em.GetBSONArray(name)
		assert.False(t, found, "expected found to be false")

		em.close(context.Background())

		retDocs, found := em.GetBSONArray(name)
		assert.True(t, found, "expected entity %s to be found", name)
		assert.Equal(t, bson.Raw(doc), retDocs[0], "expected %s, got %s", bson.Raw(doc), retDocs[0])

		_, found = em.GetBSONArray("failures")
		assert.False(t, found, "expected found to be false")
	})
	t.Run("events entity", func(t *testing.T) {
		name := "events"
		em := newEntityMap()
		err = em.addEventsEntity(name)
		assert.Nil(t, err, "expected nil error, got %s", err)
		err = em.addEventsEntity(name)
		assert.NotNil(t, err, "expected error for duplicate entity name")

		em.appendEventsEntity(name, doc)

		// Events can't be retrieved until the map is closed
		_, found := em.GetEventList(name)
		assert.False(t, found, "expected found to be false")

		em.close(context.Background())

		retDocs, found := em.GetEventList(name)
		assert.True(t, found, "expected entity %s to be found", name)
		assert.Equal(t, bson.Raw(doc), retDocs[0], "expected %s, got %s", bson.Raw(doc), retDocs[0])

		_, found = em.GetEventList("bar")
		assert.False(t, found, "expected found to be false")
	})
	t.Run("interations entity", func(t *testing.T) {
		name := "iters"
		em := newEntityMap()
		err = em.addIterationsEntity(name)
		assert.Nil(t, err, "expected nil error, got %s", err)
		err = em.addIterationsEntity(name)
		assert.NotNil(t, err, "expected error for duplicate entity name")

		em.incrementIterations(name)

		retVal, found := em.GetIterations(name)
		assert.True(t, found, "expected entity %s to be found", name)
		assert.Equal(t, int32(1), retVal, "expected %v, got %v", int32(1), retVal)

		_, found = em.GetIterations("bar")
		assert.False(t, found, "expected found to be false")
	})
	t.Run("successes entity", func(t *testing.T) {
		name := "successes"
		em := newEntityMap()
		err = em.addSuccessesEntity(name)
		assert.Nil(t, err, "expected nil error, got %s", err)
		err = em.addSuccessesEntity(name)
		assert.NotNil(t, err, "expected error for duplicate entity name")

		em.incrementSuccesses(name)

		retVal, found := em.GetSuccesses(name)
		assert.True(t, found, "expected entity %s to be found", name)
		assert.Equal(t, int32(1), retVal, "expected %v, got %v", int32(1), retVal)

		_, found = em.GetSuccesses("bar")
		assert.False(t, found, "expected found to be false")
	})
}
