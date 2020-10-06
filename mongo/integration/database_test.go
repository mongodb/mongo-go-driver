// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

const (
	listCollCapped   = "listcoll_capped"
	listCollUncapped = "listcoll_uncapped"
)

var (
	interfaceAsMapRegistry = bson.NewRegistryBuilder().
		RegisterTypeMapEntry(bsontype.EmbeddedDocument, reflect.TypeOf(bson.M{})).
		Build()
)

func TestDatabase(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))
	defer mt.Close()

	mt.RunOpts("run command", noClientOpts, func(mt *mtest.T) {
		mt.Run("decode raw", func(mt *mtest.T) {
			res, err := mt.DB.RunCommand(mtest.Background, bson.D{{"ismaster", 1}}).DecodeBytes()
			assert.Nil(mt, err, "RunCommand error: %v", err)

			ok, err := res.LookupErr("ok")
			assert.Nil(mt, err, "ok field not found in result")
			assert.Equal(mt, bson.TypeDouble, ok.Type, "expected ok type %v, got %v", bson.TypeDouble, ok.Type)
			assert.Equal(mt, 1.0, ok.Double(), "expected ok value 1.0, got %v", ok.Double())

			isMaster, err := res.LookupErr("ismaster")
			assert.Nil(mt, err, "ismaster field not found in result")
			assert.Equal(mt, bson.TypeBoolean, isMaster.Type, "expected isMaster type %v, got %v", bson.TypeBoolean, isMaster.Type)
			assert.True(mt, isMaster.Boolean(), "expected isMaster value true, got false")
		})
		mt.Run("decode struct", func(mt *mtest.T) {
			result := struct {
				IsMaster bool    `bson:"ismaster"`
				Ok       float64 `bson:"ok"`
			}{}
			err := mt.DB.RunCommand(mtest.Background, bson.D{{"ismaster", 1}}).Decode(&result)
			assert.Nil(mt, err, "RunCommand error: %v", err)
			assert.Equal(mt, true, result.IsMaster, "expected isMaster value true, got false")
			assert.Equal(mt, 1.0, result.Ok, "expected ok value 1.0, got %v", result.Ok)
		})

		// We set min server version 3.6 because pre-3.6 servers use OP_QUERY, so the command document will look like
		// {$query: {...}, $readPreference: {...}}. Per the command monitoring spec, the $query subdocument is unwrapped
		// and the $readPreference is dropped for monitoring purposes, so we can't examine it.
		readPrefOpts := mtest.NewOptions().
			Topologies(mtest.Sharded).
			MinServerVersion("3.6")
		mt.RunOpts("read pref passed to mongos", readPrefOpts, func(mt *mtest.T) {
			// When communicating with a mongos, the supplied read preference should be passed down to the operations
			// layer, which should add a top-level $readPreference field to the command.

			runCmdOpts := options.RunCmd().
				SetReadPreference(readpref.SecondaryPreferred())
			err := mt.DB.RunCommand(mtest.Background, bson.D{{"isMaster", 1}}, runCmdOpts).Err()
			assert.Nil(mt, err, "RunCommand error: %v", err)

			expected := bson.Raw(bsoncore.BuildDocumentFromElements(
				nil,
				bsoncore.AppendStringElement(nil, "mode", "secondaryPreferred"),
			))
			evt := mt.GetStartedEvent()
			assert.Equal(mt, "isMaster", evt.CommandName, "expected 'isMaster' command to be sent, got %q", evt.CommandName)
			actual, ok := evt.Command.Lookup("$readPreference").DocumentOK()
			assert.True(mt, ok, "expected command %v to contain a $readPreference document", evt.Command)
			assert.Equal(mt, expected, actual, "expected $readPreference document %v, got %v", expected, actual)
		})
		failpointOpts := mtest.NewOptions().MinServerVersion("4.0").Topologies(mtest.ReplicaSet)
		mt.RunOpts("gets result and error", failpointOpts, func(mt *mtest.T) {
			mt.SetFailPoint(mtest.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode: mtest.FailPointMode{
					Times: 1,
				},
				Data: mtest.FailPointData{
					FailCommands: []string{"insert"},
					WriteConcernError: &mtest.WriteConcernErrorData{
						Code: 100,
					},
				},
			})
			cmd := bson.D{
				{"insert", "test"},
				{"documents", bson.A{bson.D{{"a", 1}}}},
			}
			res, gotErr := mt.DB.RunCommand(mtest.Background, cmd).DecodeBytes()

			n, ok := res.Lookup("n").Int32OK()
			assert.True(mt, ok, "expected n in response")
			assert.Equal(mt, int32(1), n, "expected n value 1, got %v", n)

			writeExcept, ok := gotErr.(mongo.WriteException)
			assert.True(mt, ok, "expected WriteCommandError, got %T", gotErr)
			assert.NotNil(mt, writeExcept.WriteConcernError, "expected WriteConcernError to be non-nil")
			assert.Equal(mt, writeExcept.WriteConcernError.Code, 100, "expeced error code 100, got %v", writeExcept.WriteConcernError.Code)
		})
	})

	dropOpts := mtest.NewOptions().DatabaseName("dropDb")
	mt.RunOpts("drop", dropOpts, func(mt *mtest.T) {
		err := mt.DB.Drop(mtest.Background)
		assert.Nil(mt, err, "Drop error: %v", err)

		list, err := mt.Client.ListDatabaseNames(mtest.Background, bson.D{})
		assert.Nil(mt, err, "ListDatabaseNames error: %v", err)
		for _, db := range list {
			if db == "dropDb" {
				mt.Fatal("dropped database 'dropDb' found in database names")
			}
		}
	})

	lcNamesOpts := mtest.NewOptions().MinServerVersion("4.0")
	mt.RunOpts("list collection names", lcNamesOpts, func(mt *mtest.T) {
		collName := "lcNamesCollection"
		mt.CreateCollection(mtest.Collection{Name: collName}, true)

		testCases := []struct {
			name   string
			filter bson.D
			found  bool
		}{
			{"no filter", bson.D{}, true},
			{"filter", bson.D{{"name", "lcNamesCollection"}}, true},
			{"filter not found", bson.D{{"name", "123"}}, false},
		}
		for _, tc := range testCases {
			mt.Run(tc.name, func(mt *mtest.T) {
				colls, err := mt.DB.ListCollectionNames(mtest.Background, tc.filter)
				assert.Nil(mt, err, "ListCollectionNames error: %v", err)

				var found bool
				for _, coll := range colls {
					if coll == collName {
						found = true
						break
					}
				}

				assert.Equal(mt, tc.found, found, "expected to find collection: %v, found collection: %v", tc.found, found)
			})
		}
	})

	mt.RunOpts("list collections", noClientOpts, func(mt *mtest.T) {
		testCases := []struct {
			name             string
			expectedTopology mtest.TopologyKind
			cappedOnly       bool
		}{
			{"standalone no filter", mtest.Single, false},
			{"standalone filter", mtest.Single, true},
			{"replica set no filter", mtest.ReplicaSet, false},
			{"replica set filter", mtest.ReplicaSet, true},
			{"sharded no filter", mtest.Sharded, false},
			{"sharded filter", mtest.Sharded, true},
		}
		for _, tc := range testCases {
			tcOpts := mtest.NewOptions().Topologies(tc.expectedTopology)
			mt.RunOpts(tc.name, tcOpts, func(mt *mtest.T) {
				mt.CreateCollection(mtest.Collection{Name: listCollUncapped}, true)
				mt.CreateCollection(mtest.Collection{
					Name:       listCollCapped,
					CreateOpts: bson.D{{"capped", true}, {"size", 64 * 1024}},
				}, true)

				filter := bson.D{}
				if tc.cappedOnly {
					filter = bson.D{{"options.capped", true}}
				}

				var err error
				for i := 0; i < 1; i++ {
					cursor, err := mt.DB.ListCollections(mtest.Background, filter)
					assert.Nil(mt, err, "ListCollections error (iteration %v): %v", i, err)

					err = verifyListCollections(cursor, tc.cappedOnly)
					if err == nil {
						return
					}
				}
				mt.Fatalf("error verifying list collections result: %v", err)
			})
		}
	})

	mt.RunOpts("run command cursor", noClientOpts, func(mt *mtest.T) {
		var data []interface{}
		for i := 0; i < 5; i++ {
			data = append(data, bson.D{{"x", i}})
		}
		findCollName := "runcommandcursor_find"
		findCmd := bson.D{{"find", findCollName}}
		aggCollName := "runcommandcursor_agg"
		aggCmd := bson.D{
			{"aggregate", aggCollName},
			{"pipeline", bson.A{}},
			{"cursor", bson.D{}},
		}
		pingCmd := bson.D{{"ping", 1}}
		pingErr := errors.New("cursor should be an embedded document but is of BSON type invalid")

		testCases := []struct {
			name        string
			collName    string
			cmd         interface{}
			toInsert    []interface{}
			expectedErr error
			numExpected int
			minVersion  string
		}{
			{"success find", findCollName, findCmd, data, nil, 5, "3.2"},
			{"success aggregate", aggCollName, aggCmd, data, nil, 5, ""},
			{"failures", "runcommandcursor_ping", pingCmd, nil, pingErr, 0, ""},
		}
		for _, tc := range testCases {
			tcOpts := mtest.NewOptions().CollectionName(tc.collName)
			if tc.minVersion != "" {
				tcOpts.MinServerVersion(tc.minVersion)
			}

			mt.RunOpts(tc.name, tcOpts, func(mt *mtest.T) {
				if len(tc.toInsert) > 0 {
					_, err := mt.Coll.InsertMany(mtest.Background, tc.toInsert)
					assert.Nil(mt, err, "InsertMany error: %v", err)
				}

				cursor, err := mt.DB.RunCommandCursor(mtest.Background, tc.cmd)
				assert.Equal(mt, tc.expectedErr, err, "expected error %v, got %v", tc.expectedErr, err)
				if tc.expectedErr != nil {
					return
				}

				var count int
				for cursor.Next(mtest.Background) {
					count++
				}
				assert.Equal(mt, tc.numExpected, count, "expected document count %v, got %v", tc.numExpected, count)
			})
		}
	})

	mt.RunOpts("create collection", noClientOpts, func(mt *mtest.T) {
		collectionName := "create-collection-test"

		mt.RunOpts("options", noClientOpts, func(mt *mtest.T) {
			// Tests for various options combinations. The test creates a collection with some options and then verifies
			// the result using the options document reported by listCollections.

			// All possible options except collation. The collation is omitted here and tested below because the
			// collation document reported by listCollections fills in extra fields and includes a "version" field
			// that's not described in https://docs.mongodb.com/manual/reference/collation/.
			storageEngine := bson.M{
				"wiredTiger": bson.M{
					"configString": "block_compressor=zlib",
				},
			}
			defaultIndexOpts := options.DefaultIndex().SetStorageEngine(storageEngine)
			validator := bson.M{
				"$or": bson.A{
					bson.M{
						"phone": bson.M{"$type": "string"},
					},
					bson.M{
						"email": bson.M{"$type": "string"},
					},
				},
			}
			nonCollationOpts := options.CreateCollection().
				SetCapped(true).
				SetDefaultIndexOptions(defaultIndexOpts).
				SetMaxDocuments(100).
				SetSizeInBytes(1024).
				SetStorageEngine(storageEngine).
				SetValidator(validator).
				SetValidationAction("warn").
				SetValidationLevel("moderate")
			nonCollationExpected := bson.M{
				"capped": true,
				"indexOptionDefaults": bson.M{
					"storageEngine": storageEngine,
				},
				"max":              int32(100),
				"size":             int32(1024),
				"storageEngine":    storageEngine,
				"validator":        validator,
				"validationAction": "warn",
				"validationLevel":  "moderate",
			}

			testCases := []struct {
				name             string
				minServerVersion string
				maxServerVersion string
				createOpts       *options.CreateCollectionOptions
				expectedOpts     bson.M
			}{
				{"all options except collation", "3.2", "", nonCollationOpts, nonCollationExpected},
			}

			for _, tc := range testCases {
				mtOpts := mtest.NewOptions().
					MinServerVersion(tc.minServerVersion).
					MaxServerVersion(tc.maxServerVersion)

				mt.RunOpts(tc.name, mtOpts, func(mt *mtest.T) {
					mt.CreateCollection(mtest.Collection{
						Name: collectionName,
					}, false)

					err := mt.DB.CreateCollection(mtest.Background, collectionName, tc.createOpts)
					assert.Nil(mt, err, "CreateCollection error: %v", err)

					actualOpts := getCollectionOptions(mt, collectionName)
					assert.Equal(mt, tc.expectedOpts, actualOpts, "options mismatch; expected %v, got %v",
						tc.expectedOpts, actualOpts)
				})
			}
		})
		mt.RunOpts("collation", mtest.NewOptions().MinServerVersion("3.4"), func(mt *mtest.T) {
			mt.CreateCollection(mtest.Collection{
				Name: collectionName,
			}, false)

			locale := "en_US"
			createOpts := options.CreateCollection().SetCollation(&options.Collation{
				Locale: locale,
			})
			err := mt.DB.CreateCollection(mtest.Background, collectionName, createOpts)
			assert.Nil(mt, err, "CreateCollection error: %v", err)

			actualOpts := getCollectionOptions(mt, collectionName)
			collationVal, ok := actualOpts["collation"]
			assert.True(mt, ok, "expected key 'collation' in collection options %v", actualOpts)
			collation := collationVal.(bson.M)
			assert.Equal(mt, locale, collation["locale"], "expected locale %v, got %v", locale, collation["locale"])
		})
		mt.Run("write concern", func(mt *mtest.T) {
			mt.CreateCollection(mtest.Collection{
				Name: collectionName,
			}, false)

			err := mt.DB.CreateCollection(mtest.Background, collectionName)
			assert.Nil(mt, err, "CreateCollection error: %v", err)

			evt := mt.GetStartedEvent()
			assert.Equal(mt, evt.CommandName, "create", "expected event for 'create', got '%v'", evt.CommandName)
			_, err = evt.Command.LookupErr("writeConcern")
			assert.Nil(mt, err, "expected write concern to be included in command %v", evt.Command)
		})
	})

	mt.RunOpts("create view", mtest.NewOptions().CreateClient(false).MinServerVersion("3.4"), func(mt *mtest.T) {
		sourceCollectionName := "create-view-test-collection"
		viewName := "create-view-test-view"
		projectStage := bson.M{
			"$project": bson.M{
				"projectedField": "foo",
			},
		}
		pipeline := []bson.M{projectStage}

		mt.Run("function parameters are translated into options", func(mt *mtest.T) {
			mt.CreateCollection(mtest.Collection{
				Name: viewName,
			}, false)

			err := mt.DB.CreateView(mtest.Background, viewName, sourceCollectionName, pipeline)
			assert.Nil(mt, err, "CreateView error: %v", err)

			expectedOpts := bson.M{
				"viewOn":   sourceCollectionName,
				"pipeline": bson.A{projectStage},
			}
			actualOpts := getCollectionOptions(mt, viewName)
			assert.Equal(mt, expectedOpts, actualOpts, "options mismatch; expected %v, got %v", expectedOpts,
				actualOpts)
		})
		mt.Run("collation", func(mt *mtest.T) {
			mt.CreateCollection(mtest.Collection{
				Name: viewName,
			}, false)

			locale := "en_US"
			viewOpts := options.CreateView().SetCollation(&options.Collation{
				Locale: locale,
			})
			err := mt.DB.CreateView(mtest.Background, viewName, sourceCollectionName, mongo.Pipeline{}, viewOpts)
			assert.Nil(mt, err, "CreateView error: %v", err)

			actualOpts := getCollectionOptions(mt, viewName)
			collationVal, ok := actualOpts["collation"]
			assert.True(mt, ok, "expected key 'collation' in view options %v", actualOpts)
			collation := collationVal.(bson.M)
			assert.Equal(mt, locale, collation["locale"], "expected locale %v, got %v", locale, collation["locale"])
		})
	})
}

func getCollectionOptions(mt *mtest.T, collectionName string) bson.M {
	mt.Helper()

	filter := bson.M{
		"name": collectionName,
	}
	cursor, err := mt.DB.ListCollections(mtest.Background, filter)
	assert.Nil(mt, err, "ListCollections error: %v", err)
	defer cursor.Close(mtest.Background)
	assert.True(mt, cursor.Next(mtest.Background), "expected Next to return true, got false")

	var actualOpts bson.M
	err = bson.UnmarshalWithRegistry(interfaceAsMapRegistry, cursor.Current.Lookup("options").Document(), &actualOpts)
	assert.Nil(mt, err, "UnmarshalWithRegistry error: %v", err)

	return actualOpts
}

func verifyListCollections(cursor *mongo.Cursor, cappedOnly bool) error {
	var cappedFound, uncappedFound bool

	for cursor.Next(mtest.Background) {
		nameElem, err := cursor.Current.LookupErr("name")
		if err != nil {
			return fmt.Errorf("name element not found in document %v", cursor.Current)
		}
		if nameElem.Type != bson.TypeString {
			return fmt.Errorf("expected name type %v, got %v", bson.TypeString, nameElem.Type)
		}

		name := nameElem.StringValue()
		// legacy servers can return an indexes collection that shouldn't be considered here
		if name != listCollUncapped && name != listCollCapped {
			continue
		}

		if name == listCollUncapped && !uncappedFound {
			if cappedOnly {
				return fmt.Errorf("found uncapped collection %v but expected only capped collections", listCollUncapped)
			}

			uncappedFound = true
			continue
		}
		if name == listCollCapped && !cappedFound {
			cappedFound = true
			continue
		}

		// duplicate found
		return fmt.Errorf("found duplicate collection %v", name)
	}

	if !cappedFound {
		return fmt.Errorf("capped collection %v not found", listCollCapped)
	}
	if !cappedOnly && !uncappedFound {
		return fmt.Errorf("uncapped collection %v not found", listCollUncapped)
	}
	return nil
}
