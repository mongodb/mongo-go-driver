// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/ptrutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/topology"
)

const (
	testDbName = "unitTestDb"
)

func setupColl(name string, opts ...options.Lister[options.CollectionOptions]) *Collection {
	db := setupDb(testDbName)
	return db.Collection(name, opts...)
}

func compareColls(t *testing.T, expected *Collection, got *Collection) {
	assert.Equal(t, expected.readPreference, got.readPreference,
		"mismatch; expected read preference %v, got %v", expected.readPreference, got.readPreference)
	assert.Equal(t, expected.readConcern, got.readConcern,
		"mismatch; expected read concern %v, got %v", expected.readConcern, got.readConcern)
	assert.Equal(t, expected.writeConcern, got.writeConcern,
		"mismatch; expected write concern %v, got %v", expected.writeConcern, got.writeConcern)
}

func TestCollection(t *testing.T) {
	t.Run("initialize", func(t *testing.T) {
		name := "foo"
		coll := setupColl(name)
		assert.Equal(t, name, coll.Name(), "expected coll name %v, got %v", name, coll.Name())
		assert.NotNil(t, coll.Database(), "expected valid database, got nil")
	})
	t.Run("specified options", func(t *testing.T) {
		rpPrimary := readpref.Primary()
		rpSecondary := readpref.Secondary()
		wc1 := &writeconcern.WriteConcern{W: 5}
		wc2 := &writeconcern.WriteConcern{W: 10}
		rcLocal := readconcern.Local()
		rcMajority := readconcern.Majority()

		opts := options.Collection().SetReadPreference(rpPrimary).SetReadConcern(rcLocal).SetWriteConcern(wc1).
			SetReadPreference(rpSecondary).SetReadConcern(rcMajority).SetWriteConcern(wc2)
		expected := &Collection{
			readConcern:    rcMajority,
			readPreference: rpSecondary,
			writeConcern:   wc2,
		}
		got := setupColl("foo", opts)
		compareColls(t, expected, got)
	})
	t.Run("inherit options", func(t *testing.T) {
		rpPrimary := readpref.Primary()
		rcLocal := readconcern.Local()
		wc1 := &writeconcern.WriteConcern{W: 10}

		db := setupDb("foo", options.Database().SetReadPreference(rpPrimary).SetReadConcern(rcLocal))
		coll := db.Collection("bar", options.Collection().SetWriteConcern(wc1))
		expected := &Collection{
			readPreference: rpPrimary,
			readConcern:    rcLocal,
			writeConcern:   wc1,
		}
		compareColls(t, expected, coll)
	})
	t.Run("replaceErrors for disconnected topology", func(t *testing.T) {
		coll := setupColl("foo")
		doc := bson.D{}
		update := bson.D{{"$update", bson.D{{"x", 1}}}}

		topo, ok := coll.client.deployment.(*topology.Topology)
		require.True(t, ok, "client deployment is not a topology")

		err := topo.Disconnect(context.Background())
		require.NoError(t, err)

		_, err = coll.InsertOne(bgCtx, doc)
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)

		_, err = coll.InsertMany(bgCtx, []interface{}{doc})
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)

		_, err = coll.DeleteOne(bgCtx, doc)
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)

		_, err = coll.DeleteMany(bgCtx, doc)
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)

		_, err = coll.UpdateOne(bgCtx, doc, update)
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)

		_, err = coll.UpdateMany(bgCtx, doc, update)
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)

		_, err = coll.ReplaceOne(bgCtx, doc, doc)
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)

		_, err = coll.Aggregate(bgCtx, Pipeline{})
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)

		_, err = coll.EstimatedDocumentCount(bgCtx)
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)

		_, err = coll.CountDocuments(bgCtx, doc)
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)

		err = coll.Distinct(bgCtx, "x", doc).Err()
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)

		_, err = coll.Find(bgCtx, doc)
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)

		err = coll.FindOne(bgCtx, doc).Err()
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)

		err = coll.FindOneAndDelete(bgCtx, doc).Err()
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)

		err = coll.FindOneAndReplace(bgCtx, doc, doc).Err()
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)

		err = coll.FindOneAndUpdate(bgCtx, doc, update).Err()
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)
	})
	t.Run("database accessor", func(t *testing.T) {
		coll := setupColl("bar")
		dbName := coll.Database().Name()
		assert.Equal(t, testDbName, dbName, "expected db name %v, got %v", testDbName, dbName)
	})
	t.Run("nil document error", func(t *testing.T) {
		coll := setupColl("foo")
		doc := bson.D{}

		_, err := coll.InsertOne(bgCtx, nil)
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		_, err = coll.InsertMany(bgCtx, nil)
		assert.True(t, errors.Is(err, ErrNotSlice), "expected error %v, got %v", ErrNotSlice, err)

		_, err = coll.InsertMany(bgCtx, []interface{}{})
		assert.True(t, errors.Is(err, ErrEmptySlice), "expected error %v, got %v", ErrEmptySlice, err)

		_, err = coll.InsertMany(bgCtx, "x")
		assert.True(t, errors.Is(err, ErrNotSlice), "expected error %v, got %v", ErrNotSlice, err)

		_, err = coll.DeleteOne(bgCtx, nil)
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		_, err = coll.DeleteMany(bgCtx, nil)
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		_, err = coll.UpdateOne(bgCtx, nil, doc)
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		_, err = coll.UpdateOne(bgCtx, doc, nil)
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		_, err = coll.UpdateMany(bgCtx, nil, doc)
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		_, err = coll.UpdateMany(bgCtx, doc, nil)
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		_, err = coll.ReplaceOne(bgCtx, nil, doc)
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		_, err = coll.ReplaceOne(bgCtx, doc, nil)
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		_, err = coll.CountDocuments(bgCtx, nil)
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		err = coll.Distinct(bgCtx, "x", nil).Err()
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		_, err = coll.Find(bgCtx, nil)
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		err = coll.FindOne(bgCtx, nil).Err()
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		err = coll.FindOneAndDelete(bgCtx, nil).Err()
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		err = coll.FindOneAndReplace(bgCtx, nil, doc).Err()
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		err = coll.FindOneAndReplace(bgCtx, doc, nil).Err()
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		err = coll.FindOneAndUpdate(bgCtx, nil, doc).Err()
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		err = coll.FindOneAndUpdate(bgCtx, doc, nil).Err()
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		_, err = coll.BulkWrite(bgCtx, nil)
		assert.True(t, errors.Is(err, ErrEmptySlice), "expected error %v, got %v", ErrEmptySlice, err)

		_, err = coll.BulkWrite(bgCtx, []WriteModel{})
		assert.True(t, errors.Is(err, ErrEmptySlice), "expected error %v, got %v", ErrEmptySlice, err)

		_, err = coll.BulkWrite(bgCtx, []WriteModel{nil})
		assert.True(t, errors.Is(err, ErrNilDocument), "expected error %v, got %v", ErrNilDocument, err)

		aggErr := errors.New("can only marshal slices and arrays into aggregation pipelines, but got invalid")
		_, err = coll.Aggregate(bgCtx, nil)
		assert.Equal(t, aggErr, err, "expected error %v, got %v", aggErr, err)

		_, err = coll.Watch(bgCtx, nil)
		assert.Equal(t, aggErr, err, "expected error %v, got %v", aggErr, err)
	})
}

func TestCollation(t *testing.T) {
	t.Run("TestCollationToDocument", func(t *testing.T) {
		c := &options.Collation{
			Locale:          "locale",
			CaseLevel:       true,
			CaseFirst:       "first",
			Strength:        1,
			NumericOrdering: true,
			Alternate:       "alternate",
			MaxVariable:     "maxVariable",
			Normalization:   true,
			Backwards:       true,
		}

		doc := toDocument(c)
		expected := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendStringElement(nil, "locale", "locale"),
			bsoncore.AppendBooleanElement(nil, "caseLevel", (true)),
			bsoncore.AppendStringElement(nil, "caseFirst", ("first")),
			bsoncore.AppendInt32Element(nil, "strength", (1)),
			bsoncore.AppendBooleanElement(nil, "numericOrdering", (true)),
			bsoncore.AppendStringElement(nil, "alternate", ("alternate")),
			bsoncore.AppendStringElement(nil, "maxVariable", ("maxVariable")),
			bsoncore.AppendBooleanElement(nil, "normalization", (true)),
			bsoncore.AppendBooleanElement(nil, "backwards", (true)),
		)

		if !bytes.Equal(doc, expected) {
			t.Fatalf("collation did not match expected. got %v; wanted %v", doc, expected)
		}
	})
}

func TestNewFindArgsFromFindOneArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args *options.FindOneOptions
		want *options.FindOptions
	}{
		{
			name: "nil",
			args: nil,
			want: &options.FindOptions{
				Limit: ptrutil.Ptr(int64(-1)),
			},
		},
		{
			name: "empty",
			args: &options.FindOneOptions{},
			want: &options.FindOptions{
				Limit: ptrutil.Ptr(int64(-1)),
			},
		},
		{
			name: "non empty",
			args: &options.FindOneOptions{
				Skip: ptrutil.Ptr(int64(1)),
			},
			want: &options.FindOptions{
				Skip:  ptrutil.Ptr(int64(1)),
				Limit: ptrutil.Ptr(int64(-1)),
			},
		},
	}

	for _, test := range tests {
		test := test // Capture the range variable

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.want, newFindArgsFromFindOneArgs(test.args))
		})
	}
}
