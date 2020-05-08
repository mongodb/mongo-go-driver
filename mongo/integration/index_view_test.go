// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type index struct {
	Key  bson.D
	Name string
}

func TestIndexView(t *testing.T) {
	mt := mtest.New(t, noClientOpts)
	defer mt.Close()

	mt.Run("list", func(mt *mtest.T) {
		verifyIndexExists(mt, mt.Coll.Indexes(), index{
			Key:  bson.D{{"_id", int32(1)}},
			Name: "_id_",
		})
	})
	mt.RunOpts("create one", noClientOpts, func(mt *mtest.T) {
		mt.Run("name not specified", func(mt *mtest.T) {
			iv := mt.Coll.Indexes()
			keysDoc := bson.D{
				{"foo", int32(1)},
				{"bar", int32(-1)},
			}
			expectedName := "foo_1_bar_-1"

			indexName, err := iv.CreateOne(mtest.Background, mongo.IndexModel{
				Keys: keysDoc,
			})
			assert.Nil(mt, err, "CreateOne error: %v", err)
			assert.Equal(mt, expectedName, indexName, "expected name %q, got %q", expectedName, indexName)

			verifyIndexExists(mt, iv, index{
				Key:  keysDoc,
				Name: indexName,
			})
		})
		mt.Run("specify name", func(mt *mtest.T) {
			iv := mt.Coll.Indexes()
			keysDoc := bson.D{{"foo", int32(-1)}}
			name := "testname"

			indexName, err := iv.CreateOne(mtest.Background, mongo.IndexModel{
				Keys:    keysDoc,
				Options: options.Index().SetName(name),
			})
			assert.Nil(mt, err, "CreateOne error: %v", err)
			assert.Equal(mt, name, indexName, "expected returned name %q, got %q", name, indexName)

			verifyIndexExists(mt, iv, index{
				Key:  keysDoc,
				Name: indexName,
			})
		})
		mt.Run("all options", func(mt *mtest.T) {
			// Omits collation option because it's incompatible with version option
			_, err := mt.Coll.Indexes().CreateOne(mtest.Background, mongo.IndexModel{
				Keys: bson.D{{"foo", "text"}},
				Options: options.Index().
					SetBackground(false).
					SetExpireAfterSeconds(10).
					SetName("a").
					SetSparse(false).
					SetUnique(false).
					SetVersion(1).
					SetDefaultLanguage("english").
					SetLanguageOverride("english").
					SetTextVersion(1).
					SetWeights(bson.D{}).
					SetSphereVersion(1).
					SetBits(2).
					SetMax(10).
					SetMin(1).
					SetBucketSize(1).
					SetPartialFilterExpression(bson.D{}).
					SetStorageEngine(bson.D{
						{"wiredTiger", bson.D{
							{"configString", "block_compressor=zlib"},
						}},
					}),
			})
			assert.Nil(mt, err, "CreateOne error: %v", err)
		})
		mt.RunOpts("collation", mtest.NewOptions().MinServerVersion("3.4"), func(mt *mtest.T) {
			// collation invalid for server versions < 3.4
			_, err := mt.Coll.Indexes().CreateOne(mtest.Background, mongo.IndexModel{
				Keys: bson.D{{"bar", "text"}},
				Options: options.Index().SetCollation(&options.Collation{
					Locale: "simple",
				}),
			})
			assert.Nil(mt, err, "CreateOne error: %v", err)
		})
		mt.RunOpts("wildcard", mtest.NewOptions().MinServerVersion("4.1").CreateClient(false), func(mt *mtest.T) {
			keysDoc := bson.D{{"$**", int32(1)}}

			mt.Run("no options", func(mt *mtest.T) {
				iv := mt.Coll.Indexes()
				indexName, err := iv.CreateOne(mtest.Background, mongo.IndexModel{
					Keys: keysDoc,
				})
				assert.Nil(mt, err, "CreateOne error: %v", err)
				verifyIndexExists(mt, iv, index{
					Key:  keysDoc,
					Name: indexName,
				})
			})
			mt.Run("wildcard projection", func(mt *mtest.T) {
				iv := mt.Coll.Indexes()

				// Create an index with a wildcard projection document. The "_id: false" isn't needed to create the
				// index. We use the listIndexes command below to assert that the created index has this projection
				// document and the format of the document returned by listIndexes was changed in 4.5.x to explicitly
				// include "_id: false", so we include it here too.
				proj := bson.D{{"a", true}, {"_id", false}}
				_, err := iv.CreateOne(mtest.Background, mongo.IndexModel{
					Keys:    keysDoc,
					Options: options.Index().SetWildcardProjection(proj),
				})
				assert.Nil(mt, err, "CreateOne error: %v", err)

				indexDoc := getIndexDoc(mt, iv, keysDoc)
				assert.NotNil(mt, indexDoc, "expected to find keys document %v but was not found", keysDoc)
				checkIndexDocContains(mt, indexDoc, bson.E{
					Key:   "wildcardProjection",
					Value: proj,
				})
			})
		})
		mt.RunOpts("hidden", mtest.NewOptions().MinServerVersion("4.4"), func(mt *mtest.T) {
			iv := mt.Coll.Indexes()
			keysDoc := bson.D{{"x", int32(1)}}
			model := mongo.IndexModel{
				Keys:    keysDoc,
				Options: options.Index().SetHidden(true),
			}

			_, err := iv.CreateOne(mtest.Background, model)
			assert.Nil(mt, err, "CreateOne error: %v", err)

			indexDoc := getIndexDoc(mt, iv, keysDoc)
			assert.NotNil(mt, indexDoc, "index with keys document %v was not found", keysDoc)
			checkIndexDocContains(mt, indexDoc, bson.E{
				Key:   "hidden",
				Value: true,
			})
		})
		mt.Run("nil keys", func(mt *mtest.T) {
			_, err := mt.Coll.Indexes().CreateOne(mtest.Background, mongo.IndexModel{
				Keys: nil,
			})
			assert.NotNil(mt, err, "expected CreateOne error, got nil")
		})
		// Only run on replica sets as commitQuorum is not supported on standalones.
		mt.RunOpts("commit quorum", mtest.NewOptions().Topologies(mtest.ReplicaSet).CreateClient(false), func(mt *mtest.T) {
			intVal := options.CreateIndexes().SetCommitQuorumInt(1)
			stringVal := options.CreateIndexes().SetCommitQuorumString("majority")
			majority := options.CreateIndexes().SetCommitQuorumMajority()
			votingMembers := options.CreateIndexes().SetCommitQuorumVotingMembers()

			indexModel := mongo.IndexModel{
				Keys: bson.D{{"x", 1}},
			}

			testCases := []struct {
				name             string
				opts             *options.CreateIndexesOptions
				expectError      bool
				expectedValue    interface{} // ignored if expectError is true
				minServerVersion string
				maxServerVersion string
			}{
				{"error on server versions before 4.4", majority, true, nil, "", "4.2"},
				{"integer value", intVal, false, int32(1), "4.4", ""},
				{"string value", stringVal, false, "majority", "4.4", ""},
				{"majority", majority, false, "majority", "4.4", ""},
				{"votingMembers", votingMembers, false, "votingMembers", "4.4", ""},
			}
			for _, tc := range testCases {
				mtOpts := mtest.NewOptions().MinServerVersion(tc.minServerVersion).MaxServerVersion(tc.maxServerVersion)
				mt.RunOpts(tc.name, mtOpts, func(mt *mtest.T) {
					mt.ClearEvents()
					_, err := mt.Coll.Indexes().CreateOne(mtest.Background, indexModel, tc.opts)
					if tc.expectError {
						assert.NotNil(mt, err, "expected CreateOne error, got nil")
						return
					}

					assert.Nil(mt, err, "CreateOne error: %v", err)
					cmd := mt.GetStartedEvent().Command
					sentBSONValue, err := cmd.LookupErr("commitQuorum")
					assert.Nil(mt, err, "expected commitQuorum in command %s", cmd)

					var sentValue interface{}
					err = sentBSONValue.Unmarshal(&sentValue)
					assert.Nil(mt, err, "Unmarshal error: %v", err)

					assert.Equal(mt, tc.expectedValue, sentValue, "expected commitQuorum value %v, got %v",
						tc.expectedValue, sentValue)
				})
			}
		})
	})
	mt.Run("create many", func(mt *mtest.T) {
		mt.Run("success", func(mt *mtest.T) {
			iv := mt.Coll.Indexes()
			firstKeysDoc := bson.D{{"foo", int32(-1)}}
			secondKeysDoc := bson.D{{"bar", int32(1)}, {"baz", int32(-1)}}
			expectedNames := []string{"foo_-1", "bar_1_baz_-1"}
			indexNames, err := iv.CreateMany(mtest.Background, []mongo.IndexModel{
				{
					Keys: firstKeysDoc,
				},
				{
					Keys: secondKeysDoc,
				},
			})
			assert.Nil(mt, err, "CreateMany error: %v", err)
			assert.Equal(mt, expectedNames, indexNames, "expected returned names %v, got %v", expectedNames, indexNames)

			verifyIndexExists(mt, iv, index{
				Key:  firstKeysDoc,
				Name: indexNames[0],
			})
			verifyIndexExists(mt, iv, index{
				Key:  secondKeysDoc,
				Name: indexNames[1],
			})
		})
		wc := writeconcern.New(writeconcern.W(1))
		wcMtOpts := mtest.NewOptions().CollectionOptions(options.Collection().SetWriteConcern(wc))
		mt.RunOpts("uses writeconcern", wcMtOpts, func(mt *mtest.T) {
			iv := mt.Coll.Indexes()
			_, err := iv.CreateMany(mtest.Background, []mongo.IndexModel{
				{
					Keys: bson.D{{"foo", -1}},
				},
				{
					Keys: bson.D{{"bar", 1}, {"baz", -1}},
				},
			})
			assert.Nil(mt, err, "CreateMany error: %v", err)

			evt := mt.GetStartedEvent()
			assert.NotNil(mt, evt, "expected CommandStartedEvent, got nil")

			assert.Equal(mt, "createIndexes", evt.CommandName, "command name mismatch; expected createIndexes, got %s", evt.CommandName)

			actual, err := evt.Command.LookupErr("writeConcern", "w")
			assert.Nil(mt, err, "error getting writeConcern.w: %s", err)

			wcVal := numberFromValue(mt, actual)
			assert.Equal(mt, int64(1), wcVal, "expected writeConcern to be 1, got: %v", wcVal)
		})
		// Only run on replica sets as commitQuorum is not supported on standalones.
		mt.RunOpts("commit quorum", mtest.NewOptions().Topologies(mtest.ReplicaSet).CreateClient(false), func(mt *mtest.T) {
			intVal := options.CreateIndexes().SetCommitQuorumInt(1)
			stringVal := options.CreateIndexes().SetCommitQuorumString("majority")
			majority := options.CreateIndexes().SetCommitQuorumMajority()
			votingMembers := options.CreateIndexes().SetCommitQuorumVotingMembers()

			indexModel := mongo.IndexModel{
				Keys: bson.D{{"x", 1}},
			}

			testCases := []struct {
				name             string
				opts             *options.CreateIndexesOptions
				expectError      bool
				expectedValue    interface{} // ignored if expectError is true
				minServerVersion string
				maxServerVersion string
			}{
				{"error on server versions before 4.4", majority, true, nil, "", "4.2"},
				{"integer value", intVal, false, int32(1), "4.4", ""},
				{"string value", stringVal, false, "majority", "4.4", ""},
				{"majority", majority, false, "majority", "4.4", ""},
				{"votingMembers", votingMembers, false, "votingMembers", "4.4", ""},
			}
			for _, tc := range testCases {
				mtOpts := mtest.NewOptions().MinServerVersion(tc.minServerVersion).MaxServerVersion(tc.maxServerVersion)
				mt.RunOpts(tc.name, mtOpts, func(mt *mtest.T) {
					mt.ClearEvents()
					_, err := mt.Coll.Indexes().CreateOne(mtest.Background, indexModel, tc.opts)
					if tc.expectError {
						assert.NotNil(mt, err, "expected CreateOne error, got nil")
						return
					}

					assert.Nil(mt, err, "CreateOne error: %v", err)
					cmd := mt.GetStartedEvent().Command
					sentBSONValue, err := cmd.LookupErr("commitQuorum")
					assert.Nil(mt, err, "expected commitQuorum in command %s", cmd)

					var sentValue interface{}
					err = sentBSONValue.Unmarshal(&sentValue)
					assert.Nil(mt, err, "Unmarshal error: %v", err)

					assert.Equal(mt, tc.expectedValue, sentValue, "expected commitQuorum value %v, got %v",
						tc.expectedValue, sentValue)
				})
			}
		})
	})
	mt.Run("drop one", func(mt *mtest.T) {
		iv := mt.Coll.Indexes()
		indexNames, err := iv.CreateMany(mtest.Background, []mongo.IndexModel{
			{
				Keys: bson.D{{"foo", -1}},
			},
			{
				Keys: bson.D{{"bar", 1}, {"baz", -1}},
			},
		})
		assert.Nil(mt, err, "CreateMany error: %v", err)
		assert.Equal(mt, 2, len(indexNames), "expected 2 index names, got %v", len(indexNames))

		_, err = iv.DropOne(mtest.Background, indexNames[1])
		assert.Nil(mt, err, "DropOne error: %v", err)

		cursor, err := iv.List(mtest.Background)
		assert.Nil(mt, err, "List error: %v", err)
		for cursor.Next(mtest.Background) {
			var idx index
			err = cursor.Decode(&idx)
			assert.Nil(mt, err, "Decode error: %v (document %v)", err, cursor.Current)
			assert.NotEqual(mt, indexNames[1], idx.Name, "found index %v after dropping", indexNames[1])
		}
		assert.Nil(mt, cursor.Err(), "cursor error: %v", cursor.Err())
	})
	mt.Run("drop all", func(mt *mtest.T) {
		iv := mt.Coll.Indexes()
		names, err := iv.CreateMany(mtest.Background, []mongo.IndexModel{
			{
				Keys: bson.D{{"foo", -1}},
			},
			{
				Keys: bson.D{{"bar", 1}, {"baz", -1}},
			},
		})
		assert.Nil(mt, err, "CreateMany error: %v", err)
		assert.Equal(mt, 2, len(names), "expected 2 index names, got %v", len(names))
		_, err = iv.DropAll(mtest.Background)
		assert.Nil(mt, err, "DropAll error: %v", err)

		cursor, err := iv.List(mtest.Background)
		assert.Nil(mt, err, "List error: %v", err)
		for cursor.Next(mtest.Background) {
			var idx index
			err = cursor.Decode(&idx)
			assert.Nil(mt, err, "Decode error: %v (document %v)", err, cursor.Current)
			assert.NotEqual(mt, names[0], idx.Name, "found index %v, after dropping", names[0])
			assert.NotEqual(mt, names[1], idx.Name, "found index %v, after dropping", names[1])
		}
		assert.Nil(mt, cursor.Err(), "cursor error: %v", cursor.Err())
	})
}

func getIndexDoc(mt *mtest.T, iv mongo.IndexView, expectedKeyDoc bson.D) bson.D {
	c, err := iv.List(mtest.Background)
	assert.Nil(mt, err, "List error: %v", err)

	for c.Next(mtest.Background) {
		var index bson.D
		err = c.Decode(&index)
		assert.Nil(mt, err, "Decode error: %v", err)

		for _, elem := range index {
			if elem.Key != "key" {
				continue
			}

			if cmp.Equal(expectedKeyDoc, elem.Value.(bson.D)) {
				return index
			}
		}
	}
	return nil
}

func checkIndexDocContains(mt *mtest.T, indexDoc bson.D, expectedElem bson.E) {
	for _, elem := range indexDoc {
		if elem.Key != expectedElem.Key {
			continue
		}

		assert.Equal(mt, expectedElem, elem, "expected element %v, got %v", expectedElem, elem)
		return
	}

	mt.Fatalf("no element matching %v found", expectedElem)
}

func verifyIndexExists(mt *mtest.T, iv mongo.IndexView, expected index) {
	mt.Helper()

	cursor, err := iv.List(mtest.Background)
	assert.Nil(mt, err, "List error: %v", err)

	var found bool
	for cursor.Next(mtest.Background) {
		var idx index
		err = cursor.Decode(&idx)
		assert.Nil(mt, err, "Decode error: %v", err)

		if idx.Name == expected.Name {
			if expected.Key != nil {
				assert.Equal(mt, expected.Key, idx.Key, "key document mismatch; expected %v, got %v", expected.Key, idx.Key)
			}
			found = true
		}
	}
	assert.Nil(mt, cursor.Err(), "cursor error: %v", err)
	assert.True(mt, found, "expected to find index %v but was not found", expected.Name)
}
