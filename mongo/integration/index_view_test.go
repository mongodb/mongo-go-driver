// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

type index struct {
	Key  bson.D
	Name string
}

func TestIndexView(t *testing.T) {
	mt := mtest.New(t, noClientOpts)
	defer mt.Close()

	var pbool = func(b bool) *bool { return &b }
	var pint32 = func(i int32) *int32 { return &i }

	mt.Run("list", func(mt *mtest.T) {
		createIndexes := func(mt *mtest.T, numIndexes int) {
			mt.Helper()

			models := make([]mongo.IndexModel, 0, numIndexes)
			for i, key := 0, 'a'; i < numIndexes; i, key = i+1, key+1 {
				models = append(models, mongo.IndexModel{
					Keys: bson.M{string(key): 1},
				})
			}

			_, err := mt.Coll.Indexes().CreateMany(context.Background(), models)
			assert.Nil(mt, err, "CreateMany error: %v", err)
		}

		// For server versions below 3.0, we internally execute List() as a legacy OP_QUERY against the system.indexes
		// collection. Command monitoring upconversions translate this to a "find" command rather than "listIndexes".
		cmdName := "listIndexes"
		if mtest.CompareServerVersions(mtest.ServerVersion(), "3.0") < 0 {
			cmdName = "find"
		}

		mt.Run("_id index is always listed", func(mt *mtest.T) {
			verifyIndexExists(mt, mt.Coll.Indexes(), index{
				Key:  bson.D{{"_id", int32(1)}},
				Name: "_id_",
			})
		})
		mt.Run("getMore commands are monitored", func(mt *mtest.T) {
			createIndexes(mt, 2)
			assertGetMoreCommandsAreMonitored(mt, cmdName, func() (*mongo.Cursor, error) {
				return mt.Coll.Indexes().List(context.Background(), options.ListIndexes().SetBatchSize(2))
			})
		})
		mt.Run("killCursors commands are monitored", func(mt *mtest.T) {
			createIndexes(mt, 2)
			assertKillCursorsCommandsAreMonitored(mt, cmdName, func() (*mongo.Cursor, error) {
				return mt.Coll.Indexes().List(context.Background(), options.ListIndexes().SetBatchSize(2))
			})
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

			indexName, err := iv.CreateOne(context.Background(), mongo.IndexModel{
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

			indexName, err := iv.CreateOne(context.Background(), mongo.IndexModel{
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
			opts := options.Index().
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
				SetPartialFilterExpression(bson.D{}).
				SetStorageEngine(bson.D{
					{"wiredTiger", bson.D{
						{"configString", "block_compressor=zlib"},
					}},
				})

			// Only check SetBucketSize if version is less than 4.9
			if mtest.CompareServerVersions(mtest.ServerVersion(), "4.9") < 0 {
				opts.SetBucketSize(1)
			}
			// Omits collation option because it's incompatible with version option
			_, err := mt.Coll.Indexes().CreateOne(context.Background(), mongo.IndexModel{
				Keys:    bson.D{{"foo", "text"}},
				Options: opts,
			})
			assert.Nil(mt, err, "CreateOne error: %v", err)
		})
		mt.RunOpts("collation", mtest.NewOptions().MinServerVersion("3.4"), func(mt *mtest.T) {
			// collation invalid for server versions < 3.4
			_, err := mt.Coll.Indexes().CreateOne(context.Background(), mongo.IndexModel{
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
				indexName, err := iv.CreateOne(context.Background(), mongo.IndexModel{
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
				_, err := iv.CreateOne(context.Background(), mongo.IndexModel{
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

			_, err := iv.CreateOne(context.Background(), model)
			assert.Nil(mt, err, "CreateOne error: %v", err)

			indexDoc := getIndexDoc(mt, iv, keysDoc)
			assert.NotNil(mt, indexDoc, "index with keys document %v was not found", keysDoc)
			checkIndexDocContains(mt, indexDoc, bson.E{
				Key:   "hidden",
				Value: true,
			})
		})
		mt.Run("nil keys", func(mt *mtest.T) {
			_, err := mt.Coll.Indexes().CreateOne(context.Background(), mongo.IndexModel{
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
					_, err := mt.Coll.Indexes().CreateOne(context.Background(), indexModel, tc.opts)
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
		unackClientOpts := options.Client().
			SetWriteConcern(writeconcern.New(writeconcern.W(0)))
		unackMtOpts := mtest.NewOptions().
			ClientOptions(unackClientOpts).
			MinServerVersion("3.6")
		mt.RunOpts("unacknowledged write", unackMtOpts, func(mt *mtest.T) {
			_, err := mt.Coll.Indexes().CreateOne(context.Background(), mongo.IndexModel{Keys: bson.D{{"x", 1}}})
			if err != mongo.ErrUnacknowledgedWrite {
				// Use a direct comparison rather than assert.Equal because assert.Equal will compare the error strings,
				// so the assertion would succeed even if the error had not been wrapped.
				mt.Fatalf("expected CreateOne error %v, got %v", mongo.ErrUnacknowledgedWrite, err)
			}
		})
		// Needs to run on these versions for failpoints
		mt.RunOpts("replace error", mtest.NewOptions().Topologies(mtest.ReplicaSet).MinServerVersion("4.0"), func(mt *mtest.T) {
			mt.SetFailPoint(mtest.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode:               "alwaysOn",
				Data: mtest.FailPointData{
					FailCommands: []string{"createIndexes"},
					ErrorCode:    100,
				},
			})

			_, err := mt.Coll.Indexes().CreateOne(context.Background(), mongo.IndexModel{Keys: bson.D{{"x", 1}}})
			assert.NotNil(mt, err, "expected CreateOne error, got nil")
			cmdErr, ok := err.(mongo.CommandError)
			assert.True(mt, ok, "expected mongo.CommandError, got %T", err)
			assert.Equal(mt, int32(100), cmdErr.Code, "expected error code 100, got %v", cmdErr.Code)

		})
		mt.Run("multi-key map", func(mt *mtest.T) {
			iv := mt.Coll.Indexes()

			_, err := iv.CreateOne(context.Background(), mongo.IndexModel{
				Keys: bson.M{"foo": 1, "bar": -1},
			})
			assert.NotNil(mt, err, "expected CreateOne error, got nil")
			assert.Equal(mt, mongo.ErrMapForOrderedArgument{"keys"}, err, "expected error %v, got %v", mongo.ErrMapForOrderedArgument{"key"}, err)
		})
		mt.Run("single key map", func(mt *mtest.T) {
			iv := mt.Coll.Indexes()
			expectedName := "foo_1"

			indexName, err := iv.CreateOne(context.Background(), mongo.IndexModel{
				Keys: bson.M{"foo": 1},
			})
			assert.Nil(mt, err, "CreateOne error: %v", err)
			assert.Equal(mt, expectedName, indexName, "expected name %q, got %q", expectedName, indexName)

			verifyIndexExists(mt, iv, index{
				Key:  bson.D{{"foo", int32(1)}},
				Name: indexName,
			})
		})
	})
	mt.Run("create many", func(mt *mtest.T) {
		mt.Run("success", func(mt *mtest.T) {
			iv := mt.Coll.Indexes()
			firstKeysDoc := bson.D{{"foo", int32(-1)}}
			secondKeysDoc := bson.D{{"bar", int32(1)}, {"baz", int32(-1)}}
			expectedNames := []string{"foo_-1", "bar_1_baz_-1"}
			indexNames, err := iv.CreateMany(context.Background(), []mongo.IndexModel{
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
			_, err := iv.CreateMany(context.Background(), []mongo.IndexModel{
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

			indexModel1 := mongo.IndexModel{
				Keys: bson.D{{"x", 1}},
			}
			indexModel2 := mongo.IndexModel{
				Keys: bson.D{{"y", 1}},
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
					_, err := mt.Coll.Indexes().CreateMany(context.Background(), []mongo.IndexModel{indexModel1, indexModel2}, tc.opts)
					if tc.expectError {
						assert.NotNil(mt, err, "expected CreateMany error, got nil")
						return
					}

					assert.Nil(mt, err, "CreateMany error: %v", err)
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
		// Needs to run on these versions for failpoints
		mt.RunOpts("replace error", mtest.NewOptions().Topologies(mtest.ReplicaSet).MinServerVersion("4.0"), func(mt *mtest.T) {
			mt.SetFailPoint(mtest.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode:               "alwaysOn",
				Data: mtest.FailPointData{
					FailCommands: []string{"createIndexes"},
					ErrorCode:    100,
				},
			})

			_, err := mt.Coll.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
				{
					Keys: bson.D{{"foo", int32(-1)}},
				},
				{
					Keys: bson.D{{"bar", int32(1)}, {"baz", int32(-1)}},
				},
			})
			assert.NotNil(mt, err, "expected CreateMany error, got nil")
			cmdErr, ok := err.(mongo.CommandError)
			assert.True(mt, ok, "expected mongo.CommandError, got %T", err)
			assert.Equal(mt, int32(100), cmdErr.Code, "expected error code 100, got %v", cmdErr.Code)

		})
		mt.Run("multi-key map", func(mt *mtest.T) {
			iv := mt.Coll.Indexes()
			_, err := iv.CreateMany(context.Background(), []mongo.IndexModel{
				{
					Keys: bson.M{"foo": 1, "bar": -1},
				},
				{
					Keys: bson.D{{"bar", int32(1)}, {"baz", int32(-1)}},
				},
			})
			assert.NotNil(mt, err, "expected CreateOne error, got nil")
			assert.Equal(mt, mongo.ErrMapForOrderedArgument{"keys"}, err, "expected error %v, got %v", mongo.ErrMapForOrderedArgument{"keys"}, err)
		})
		mt.Run("single key map", func(mt *mtest.T) {
			iv := mt.Coll.Indexes()
			firstKeysDoc := bson.M{"foo": -1}
			secondKeysDoc := bson.D{{"bar", int32(1)}, {"baz", int32(-1)}}
			expectedNames := []string{"foo_-1", "bar_1_baz_-1"}
			indexNames, err := iv.CreateMany(context.Background(), []mongo.IndexModel{
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
				Key:  bson.D{{"foo", int32(-1)}},
				Name: indexNames[0],
			})
			verifyIndexExists(mt, iv, index{
				Key:  secondKeysDoc,
				Name: indexNames[1],
			})
		})
	})
	mt.RunOpts("list specifications", noClientOpts, func(mt *mtest.T) {
		mt.Run("verify results", func(mt *mtest.T) {
			// Create a handful of indexes
			_, err := mt.Coll.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
				{
					Keys:    bson.D{{"foo", int32(-1)}},
					Options: options.Index().SetUnique(true),
				},
				{
					Keys:    bson.D{{"bar", int32(1)}},
					Options: options.Index().SetExpireAfterSeconds(120),
				},
				{
					Keys:    bson.D{{"baz", int32(1)}},
					Options: options.Index().SetSparse(true),
				},
				{
					Keys: bson.D{{"bar", int32(1)}, {"baz", int32(-1)}},
				},
			})
			assert.Nil(mt, err, "CreateMany error: %v", err)

			expectedSpecs := []*mongo.IndexSpecification{
				{
					Name:               "_id_",
					Namespace:          mt.DB.Name() + "." + mt.Coll.Name(),
					KeysDocument:       bson.Raw(bsoncore.NewDocumentBuilder().AppendInt32("_id", 1).Build()),
					Version:            2,
					ExpireAfterSeconds: nil,
					Sparse:             nil,
					// ID index is special and does not return 'true', despite being unique.
					Unique: nil,
				},
				{
					Name:               "foo_-1",
					Namespace:          mt.DB.Name() + "." + mt.Coll.Name(),
					KeysDocument:       bson.Raw(bsoncore.NewDocumentBuilder().AppendInt32("foo", -1).Build()),
					Version:            2,
					ExpireAfterSeconds: nil,
					Sparse:             nil,
					Unique:             pbool(true),
				},
				{
					Name:               "bar_1",
					Namespace:          mt.DB.Name() + "." + mt.Coll.Name(),
					KeysDocument:       bson.Raw(bsoncore.NewDocumentBuilder().AppendInt32("bar", 1).Build()),
					Version:            2,
					ExpireAfterSeconds: pint32(120),
					Sparse:             nil,
					Unique:             nil,
				},
				{
					Name:               "baz_1",
					Namespace:          mt.DB.Name() + "." + mt.Coll.Name(),
					KeysDocument:       bson.Raw(bsoncore.NewDocumentBuilder().AppendInt32("baz", 1).Build()),
					Version:            2,
					ExpireAfterSeconds: nil,
					Sparse:             pbool(true),
					Unique:             nil,
				},
				{
					Name:               "bar_1_baz_-1",
					Namespace:          mt.DB.Name() + "." + mt.Coll.Name(),
					KeysDocument:       bson.Raw(bsoncore.NewDocumentBuilder().AppendInt32("bar", 1).AppendInt32("baz", -1).Build()),
					Version:            2,
					ExpireAfterSeconds: nil,
					Sparse:             nil,
					Unique:             nil,
				},
			}
			if mtest.CompareServerVersions(mtest.ServerVersion(), "3.4") < 0 {
				for _, expectedSpec := range expectedSpecs {
					expectedSpec.Version = 1
				}
			}

			specs, err := mt.Coll.Indexes().ListSpecifications(context.Background())
			assert.Nil(mt, err, "ListSpecifications error: %v", err)
			assert.Equal(mt, len(expectedSpecs), len(specs), "expected %d specification, got %d", len(expectedSpecs), len(specs))
			assert.True(mt, cmp.Equal(specs, expectedSpecs), "expected specifications to match: %v", cmp.Diff(specs, expectedSpecs))
		})
		mt.RunOpts("options passed to listIndexes", mtest.NewOptions().MinServerVersion("3.0"), func(mt *mtest.T) {
			opts := options.ListIndexes().SetMaxTime(100 * time.Millisecond)
			_, err := mt.Coll.Indexes().ListSpecifications(context.Background(), opts)
			assert.Nil(mt, err, "ListSpecifications error: %v", err)

			evt := mt.GetStartedEvent()
			assert.Equal(mt, evt.CommandName, "listIndexes", "expected %q command to be sent, got %q", "listIndexes",
				evt.CommandName)
			maxTimeMS, ok := evt.Command.Lookup("maxTimeMS").Int64OK()
			assert.True(mt, ok, "expected command %v to contain %q field", evt.Command, "maxTimeMS")
			assert.Equal(mt, int64(100), maxTimeMS, "expected maxTimeMS value to be 100, got %d", maxTimeMS)
		})
	})
	mt.Run("drop one", func(mt *mtest.T) {
		iv := mt.Coll.Indexes()
		indexNames, err := iv.CreateMany(context.Background(), []mongo.IndexModel{
			{
				Keys: bson.D{{"foo", -1}},
			},
			{
				Keys: bson.D{{"bar", 1}, {"baz", -1}},
			},
		})
		assert.Nil(mt, err, "CreateMany error: %v", err)
		assert.Equal(mt, 2, len(indexNames), "expected 2 index names, got %v", len(indexNames))

		_, err = iv.DropOne(context.Background(), indexNames[1])
		assert.Nil(mt, err, "DropOne error: %v", err)

		cursor, err := iv.List(context.Background())
		assert.Nil(mt, err, "List error: %v", err)
		for cursor.Next(context.Background()) {
			var idx index
			err = cursor.Decode(&idx)
			assert.Nil(mt, err, "Decode error: %v (document %v)", err, cursor.Current)
			assert.NotEqual(mt, indexNames[1], idx.Name, "found index %v after dropping", indexNames[1])
		}
		assert.Nil(mt, cursor.Err(), "cursor error: %v", cursor.Err())
	})
	mt.Run("drop all", func(mt *mtest.T) {
		iv := mt.Coll.Indexes()
		names, err := iv.CreateMany(context.Background(), []mongo.IndexModel{
			{
				Keys: bson.D{{"foo", -1}},
			},
			{
				Keys: bson.D{{"bar", 1}, {"baz", -1}},
			},
		})
		assert.Nil(mt, err, "CreateMany error: %v", err)
		assert.Equal(mt, 2, len(names), "expected 2 index names, got %v", len(names))
		_, err = iv.DropAll(context.Background())
		assert.Nil(mt, err, "DropAll error: %v", err)

		cursor, err := iv.List(context.Background())
		assert.Nil(mt, err, "List error: %v", err)
		for cursor.Next(context.Background()) {
			var idx index
			err = cursor.Decode(&idx)
			assert.Nil(mt, err, "Decode error: %v (document %v)", err, cursor.Current)
			assert.NotEqual(mt, names[0], idx.Name, "found index %v, after dropping", names[0])
			assert.NotEqual(mt, names[1], idx.Name, "found index %v, after dropping", names[1])
		}
		assert.Nil(mt, cursor.Err(), "cursor error: %v", cursor.Err())
	})
	mt.RunOpts("clustered indexes", mtest.NewOptions().MinServerVersion("5.3"), func(mt *mtest.T) {
		const name = "clustered"
		clustered := mt.CreateCollection(mtest.Collection{
			Name:       name,
			CreateOpts: options.CreateCollection().SetClusteredIndex(bson.D{{"key", bson.D{{"_id", 1}}}, {"unique", true}}),
		}, true)
		mt.Run("create one", func(mt *mtest.T) {
			_, err := clustered.Indexes().CreateOne(context.Background(), mongo.IndexModel{
				Keys: bson.D{{"foo", int32(-1)}},
			})
			assert.Nil(mt, err, "CreateOne error: %v", err)
			specs, err := clustered.Indexes().ListSpecifications(context.Background())
			assert.Nil(mt, err, "ListSpecifications error: %v", err)
			expectedSpecs := []*mongo.IndexSpecification{
				{
					Name:         "_id_",
					Namespace:    mt.DB.Name() + "." + name,
					KeysDocument: bson.Raw(bsoncore.NewDocumentBuilder().AppendInt32("_id", 1).Build()),
					Version:      2,
					Unique:       func(b bool) *bool { return &b }(true),
					Clustered:    func(b bool) *bool { return &b }(true),
				},
				{
					Name:         "foo_-1",
					Namespace:    mt.DB.Name() + "." + name,
					KeysDocument: bson.Raw(bsoncore.NewDocumentBuilder().AppendInt32("foo", -1).Build()),
					Version:      2,
				},
			}
			assert.True(mt, cmp.Equal(specs, expectedSpecs), "expected specifications to match: %v", cmp.Diff(specs, expectedSpecs))
		})
	})
}

func getIndexDoc(mt *mtest.T, iv mongo.IndexView, expectedKeyDoc bson.D) bson.D {
	c, err := iv.List(context.Background())
	assert.Nil(mt, err, "List error: %v", err)

	for c.Next(context.Background()) {
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

	cursor, err := iv.List(context.Background())
	assert.Nil(mt, err, "List error: %v", err)

	var found bool
	for cursor.Next(context.Background()) {
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
