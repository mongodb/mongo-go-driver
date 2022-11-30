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
	"go.mongodb.org/mongo-driver/internal/assert"
)

func TestEntityMap(t *testing.T) {
	record := bson.D{{"foo", 1}}
	doc, err := bson.Marshal(record)
	assert.Nil(t, err, "error marshaling example doc %v", err)
	t.Run("bson array entity", func(t *testing.T) {
		name := "errors"
		notFoundName := "failures"
		notFoundErr := newEntityNotFoundError("BSON array", "failures")
		em := newEntityMap()

		err = em.addBSONArrayEntity(name)
		assert.Nil(t, err, "addBSONArrayEntity error: %v", err)
		// adding an existing bson array entity twice shouldn't error
		err = em.addBSONArrayEntity(name)
		assert.Nil(t, err, "addBSONArrayEntity error: %v", err)

		err = em.appendBSONArrayEntity(name, doc)
		assert.Nil(t, err, "appendBSONArrayEntity error: %v", err)

		err = em.appendBSONArrayEntity(notFoundName, doc)
		assert.Equal(t, err, notFoundErr, "expected error %v, got %v", notFoundErr, err)

		// bson array can't be retrieved until the map is closed
		_, err = em.BSONArray(name)
		assert.Equal(t, err, ErrEntityMapOpen, "expected error %v, got %v", ErrEntityMapOpen, err)

		em.close(context.Background())

		retDocs, err := em.BSONArray(name)
		assert.Nil(t, err, "BSONArray error: %v", err)
		assert.Equal(t, bson.Raw(doc), retDocs[0], "expected %s, got %s", bson.Raw(doc), retDocs[0])

		_, err = em.BSONArray(notFoundName)
		assert.Equal(t, err, notFoundErr, "expected error %v, got %v", notFoundErr, err)
	})
	t.Run("events entity", func(t *testing.T) {
		name := "events"
		em := newEntityMap()
		err = em.addEventsEntity(name)
		assert.Nil(t, err, "addEventsEntity error: %v", err)
		err = em.addEventsEntity(name)
		assert.NotNil(t, err, "expected error for duplicate entity name")

		em.appendEventsEntity(name, doc)

		// Events can't be retrieved until the map is closed
		_, err := em.EventList(name)
		assert.Equal(t, err, ErrEntityMapOpen, "expected error %v, got %v", ErrEntityMapOpen, err)

		em.close(context.Background())

		retDocs, err := em.EventList(name)
		assert.Nil(t, err, "EventList error: %v", err)
		assert.Equal(t, bson.Raw(doc), retDocs[0], "expected %s, got %s", bson.Raw(doc), retDocs[0])

		_, err = em.EventList("bar")
		notFoundErr := newEntityNotFoundError("event list", "bar")
		assert.Equal(t, err, notFoundErr, "expected error %v, got %v", notFoundErr, err)
	})
	t.Run("interations entity", func(t *testing.T) {
		name := "iters"
		notFoundName := "bar"
		notFoundErr := newEntityNotFoundError("iterations", "bar")
		em := newEntityMap()
		err = em.addIterationsEntity(name)
		assert.Nil(t, err, "addIterationsEntity error: %v", err)
		err = em.addIterationsEntity(name)
		assert.NotNil(t, err, "expected error for duplicate entity name")

		err = em.incrementIterations(name)
		assert.Nil(t, err, "incrementIterations error: %v", err)

		err = em.incrementIterations(notFoundName)
		assert.Equal(t, err, notFoundErr, "expected error %v, got %v", notFoundErr, err)

		retVal, err := em.Iterations(name)
		assert.Nil(t, err, "expected nil error, got %v", err)
		assert.Equal(t, int32(1), retVal, "expected %v, got %v", int32(1), retVal)

		_, err = em.Iterations(notFoundName)
		assert.Equal(t, err, notFoundErr, "expected error %v, got %v", notFoundErr, err)
	})
	t.Run("successes entity", func(t *testing.T) {
		name := "successes"
		notFoundName := "bar"
		notFoundErr := newEntityNotFoundError("successes", "bar")
		em := newEntityMap()
		err = em.addSuccessesEntity(name)
		assert.Nil(t, err, "addSuccessesEntity error: %v", err)
		err = em.addSuccessesEntity(name)
		assert.NotNil(t, err, "expected error for duplicate entity name")

		err = em.incrementSuccesses(name)
		assert.Nil(t, err, "incrementSuccesses error: %v", err)

		err = em.incrementSuccesses(notFoundName)
		assert.Equal(t, err, notFoundErr, "expected error %v, got %v", notFoundErr, err)

		retVal, err := em.Successes(name)
		assert.Nil(t, err, "Successes error: %v", err)
		assert.Equal(t, int32(1), retVal, "expected %v, got %v", int32(1), retVal)

		_, err = em.Successes(notFoundName)
		assert.Equal(t, err, notFoundErr, "expected error %v, got %v", notFoundErr, err)
	})
}
