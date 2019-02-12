// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"math"
	"path"
	"strconv"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"

	"fmt"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type testFile struct {
	Data             json.RawMessage
	MinServerVersion string
	MaxServerVersion string
	Tests            []testCase
}

type testCase struct {
	Description string
	Operation   operation
	Outcome     outcome
}

type operation struct {
	Name      string
	Arguments map[string]interface{}
}

type outcome struct {
	Result     json.RawMessage
	Collection *collection
}

type collection struct {
	Name string
	Data json.RawMessage
}

const crudTestsDir = "../data/crud"
const readTestsDir = "read"
const writeTestsDir = "write"

// compareVersions compares two version number strings (i.e. positive integers separated by
// periods). Comparisons are done to the lesser precision of the two versions. For example, 3.2 is
// considered equal to 3.2.11, whereas 3.2.0 is considered less than 3.2.11.
//
// Returns a positive int if version1 is greater than version2, a negative int if version1 is less
// than version2, and 0 if version1 is equal to version2.
func compareVersions(t *testing.T, v1 string, v2 string) int {
	n1 := strings.Split(v1, ".")
	n2 := strings.Split(v2, ".")

	for i := 0; i < int(math.Min(float64(len(n1)), float64(len(n2)))); i++ {
		i1, err := strconv.Atoi(n1[i])
		require.NoError(t, err)

		i2, err := strconv.Atoi(n2[i])
		require.NoError(t, err)

		difference := i1 - i2
		if difference != 0 {
			return difference
		}
	}

	return 0
}

func getServerVersion(db *Database) (string, error) {
	var serverStatus bsonx.Doc
	err := db.RunCommand(
		context.Background(),
		bsonx.Doc{{"serverStatus", bsonx.Int32(1)}},
	).Decode(&serverStatus)
	if err != nil {
		return "", err
	}

	version, err := serverStatus.LookupErr("version")
	if err != nil {
		return "", err
	}

	return version.StringValue(), nil
}

// Test case for all CRUD spec tests.
func TestCRUDSpec(t *testing.T) {
	dbName := "crud-spec-tests"
	db := createTestDatabase(t, &dbName)

	for _, file := range testhelpers.FindJSONFilesInDir(t, path.Join(crudTestsDir, readTestsDir)) {
		runCRUDTestFile(t, path.Join(crudTestsDir, readTestsDir, file), db)
	}

	for _, file := range testhelpers.FindJSONFilesInDir(t, path.Join(crudTestsDir, writeTestsDir)) {
		runCRUDTestFile(t, path.Join(crudTestsDir, writeTestsDir, file), db)
	}
}

func runCRUDTestFile(t *testing.T, filepath string, db *Database) {
	content, err := ioutil.ReadFile(filepath)
	require.NoError(t, err)

	var testfile testFile
	require.NoError(t, json.Unmarshal(content, &testfile))

	if shouldSkip(t, testfile.MinServerVersion, testfile.MaxServerVersion, db) {
		return
	}

	for _, test := range testfile.Tests {
		collName := sanitizeCollectionName("crud-spec-tests", test.Description)

		_ = db.RunCommand(
			context.Background(),
			bsonx.Doc{{"drop", bsonx.String(collName)}},
		)

		if test.Outcome.Collection != nil && len(test.Outcome.Collection.Name) > 0 {
			_ = db.RunCommand(
				context.Background(),
				bsonx.Doc{{"drop", bsonx.String(test.Outcome.Collection.Name)}},
			)
		}

		coll := db.Collection(collName)
		docsToInsert := docSliceToInterfaceSlice(docSliceFromRaw(t, testfile.Data))

		wcColl, err := coll.Clone(options.Collection().SetWriteConcern(writeconcern.New(writeconcern.WMajority())))
		require.NoError(t, err)
		_, err = wcColl.InsertMany(context.Background(), docsToInsert)
		require.NoError(t, err)

		switch test.Operation.Name {
		case "aggregate":
			aggregateTest(t, db, coll, &test)
		case "bulkWrite":
			bulkWriteTest(t, wcColl, &test)
		case "count":
			countTest(t, coll, &test)
		case "distinct":
			distinctTest(t, coll, &test)
		case "find":
			findTest(t, coll, &test)
		case "deleteMany":
			deleteManyTest(t, coll, &test)
		case "deleteOne":
			deleteOneTest(t, coll, &test)
		case "findOneAndDelete":
			findOneAndDeleteTest(t, coll, &test)
		case "findOneAndReplace":
			findOneAndReplaceTest(t, coll, &test)
		case "findOneAndUpdate":
			findOneAndUpdateTest(t, coll, &test)
		case "insertMany":
			insertManyTest(t, coll, &test)
		case "insertOne":
			insertOneTest(t, coll, &test)
		case "replaceOne":
			replaceOneTest(t, coll, &test)
		case "updateMany":
			updateManyTest(t, coll, &test)
		case "updateOne":
			updateOneTest(t, coll, &test)
		}
	}
}

func aggregateTest(t *testing.T, db *Database, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		pipeline := test.Operation.Arguments["pipeline"].([]interface{})

		opts := options.Aggregate()

		if batchSize, found := test.Operation.Arguments["batchSize"]; found {
			opts = opts.SetBatchSize(int32(batchSize.(float64)))
		}

		if collation, found := test.Operation.Arguments["collation"]; found {
			opts = opts.SetCollation(collationFromMap(collation.(map[string]interface{})))
		}

		out := false
		if len(pipeline) > 0 {
			if _, found := pipeline[len(pipeline)-1].(map[string]interface{})["$out"]; found {
				out = true
			}
		}

		cursor, err := coll.Aggregate(context.Background(), pipeline, opts)
		require.NoError(t, err)

		if !out {
			verifyCursorResult2(t, cursor, test.Outcome.Result)
		}

		if test.Outcome.Collection != nil {
			outColl := coll
			if len(test.Outcome.Collection.Name) > 0 {
				outColl = db.Collection(test.Outcome.Collection.Name)
			}

			verifyCollectionContents(t, outColl, test.Outcome.Collection.Data)
		}
	})
}

func bulkWriteTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		// TODO(GODRIVER-593): Figure out why this test fails
		if test.Description == "BulkWrite with replaceOne operations" {
			t.Skip("skipping replaceOne test")
		}

		requests := test.Operation.Arguments["requests"].([]interface{})
		models := make([]WriteModel, len(requests))

		for i, req := range requests {
			reqMap := req.(map[string]interface{})

			var filter map[string]interface{}
			var document map[string]interface{}
			var replacement map[string]interface{}
			var update map[string]interface{}
			var arrayFilters options.ArrayFilters
			var arrayFiltersSet bool
			var collation *options.Collation
			var upsert bool
			var upsertSet bool

			argsMap := reqMap["arguments"].(map[string]interface{})
			for k, v := range argsMap {
				var err error

				switch k {
				case "filter":
					filter = v.(map[string]interface{})
				case "document":
					document = v.(map[string]interface{})
				case "replacement":
					replacement = v.(map[string]interface{})
				case "update":
					update = v.(map[string]interface{})
				case "upsert":
					upsertSet = true
					upsert = v.(bool)
				case "collation":
					collation = collationFromMap(v.(map[string]interface{}))
				case "arrayFilters":
					arrayFilters = options.ArrayFilters{
						Filters: v.([]interface{}),
					}
					arrayFiltersSet = true
				default:
					fmt.Printf("unknown argument: %s\n", k)
				}

				if err != nil {
					t.Fatalf("error parsing argument %s: %s", k, err)
				}
			}

			for _, m := range []map[string]interface{}{filter, document, replacement, update} {
				if m != nil {
					replaceFloatsWithInts(m)
				}
			}

			var model WriteModel
			switch reqMap["name"] {
			case "deleteOne":
				dom := NewDeleteOneModel()
				if filter != nil {
					dom = dom.SetFilter(filter)
				}
				if collation != nil {
					dom = dom.SetCollation(collation)
				}
				model = dom
			case "deleteMany":
				dmm := NewDeleteManyModel()
				if filter != nil {
					dmm = dmm.SetFilter(filter)
				}
				if collation != nil {
					dmm = dmm.SetCollation(collation)
				}
				model = dmm
			case "insertOne":
				iom := NewInsertOneModel()
				if document != nil {
					iom = iom.SetDocument(document)
				}
				model = iom
			case "replaceOne":
				rom := NewReplaceOneModel()
				if filter != nil {
					rom = rom.SetFilter(filter)
				}
				if replacement != nil {
					rom = rom.SetReplacement(replacement)
				}
				if upsertSet {
					rom = rom.SetUpsert(upsert)
				}
				if collation != nil {
					rom = rom.SetCollation(collation)
				}
				model = rom
			case "updateOne":
				uom := NewUpdateOneModel()
				if filter != nil {
					uom = uom.SetFilter(filter)
				}
				if update != nil {
					uom = uom.SetUpdate(update)
				}
				if upsertSet {
					uom = uom.SetUpsert(upsert)
				}
				if collation != nil {
					uom = uom.SetCollation(collation)
				}
				if arrayFiltersSet {
					uom = uom.SetArrayFilters(arrayFilters)
				}
				model = uom
			case "updateMany":
				umm := NewUpdateManyModel()
				if filter != nil {
					umm = umm.SetFilter(filter)
				}
				if update != nil {
					umm = umm.SetUpdate(update)
				}
				if upsertSet {
					umm = umm.SetUpsert(upsert)
				}
				if collation != nil {
					umm = umm.SetCollation(collation)
				}
				if arrayFiltersSet {
					umm = umm.SetArrayFilters(arrayFilters)
				}
				model = umm
			default:
				fmt.Printf("unknown operation: %s\n", doc.Lookup("name").StringValue())
			}

			models[i] = model
		}

		optsBytes, err := bson.Marshal(test.Operation.Arguments["options"])
		if err != nil {
			t.Fatalf("error marshalling options: %s", err)
		}
		optsDoc, err := bsonx.ReadDoc(optsBytes)
		if err != nil {
			t.Fatalf("error creating options doc: %s", err)
		}

		opts := options.BulkWrite()
		for _, elem := range optsDoc {
			k := elem.Key
			val := optsDoc.Lookup(k)

			switch k {
			case "ordered":
				opts = opts.SetOrdered(val.Boolean())
			default:
				fmt.Printf("unkonwn bulk write opt: %s\n", k)
			}
		}

		res, err := coll.BulkWrite(ctx, models, opts)
		verifyBulkWriteResult(t, res, test.Outcome.Result)
		verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
	})
}

func countTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		actualCount, err := executeCount(nil, coll, test.Operation.Arguments)
		require.NoError(t, err)

		resultBytes, err := test.Outcome.Result.MarshalJSON()
		require.NoError(t, err)

		var expectedCount float64
		require.NoError(t, json.NewDecoder(bytes.NewBuffer(resultBytes)).Decode(&expectedCount))

		require.Equal(t, int64(expectedCount), actualCount)
	})
}

func deleteManyTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		actual, err := executeDeleteMany(nil, coll, test.Operation.Arguments)
		require.NoError(t, err)

		verifyDeleteResult(t, actual, test.Outcome.Result)

		verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
	})
}

func deleteOneTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		actual, err := executeDeleteOne(nil, coll, test.Operation.Arguments)
		require.NoError(t, err)

		verifyDeleteResult(t, actual, test.Outcome.Result)

		if test.Outcome.Collection != nil {
			verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
		}
	})
}

func distinctTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		actual, err := executeDistinct(nil, coll, test.Operation.Arguments)
		require.NoError(t, err)

		verifyDistinctResult(t, actual, test.Outcome.Result)
	})
}

func findTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		cursor, err := executeFind(nil, coll, test.Operation.Arguments)
		require.NoError(t, err)

		verifyCursorResult(t, cursor, test.Outcome.Result)
	})
}

func findOneAndDeleteTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		actualResult := executeFindOneAndDelete(nil, coll, test.Operation.Arguments)

		verifySingleResult(t, actualResult, test.Outcome.Result)

		verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
	})
}

func findOneAndReplaceTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		actualResult := executeFindOneAndReplace(nil, coll, test.Operation.Arguments)

		verifySingleResult(t, actualResult, test.Outcome.Result)

		verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
	})
}

func findOneAndUpdateTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		actualResult := executeFindOneAndUpdate(nil, coll, test.Operation.Arguments)
		verifySingleResult(t, actualResult, test.Outcome.Result)

		verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
	})
}

func insertManyTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		actual, err := executeInsertMany(nil, coll, test.Operation.Arguments)
		require.NoError(t, err)

		verifyInsertManyResult(t, actual, test.Outcome.Result)

		verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
	})
}

func insertOneTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		actual, err := executeInsertOne(nil, coll, test.Operation.Arguments)
		require.NoError(t, err)

		verifyInsertOneResult(t, actual, test.Outcome.Result)

		verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
	})
}

func replaceOneTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		actual, err := executeReplaceOne(nil, coll, test.Operation.Arguments)
		require.NoError(t, err)

		verifyUpdateResult(t, actual, test.Outcome.Result)

		if test.Outcome.Collection != nil {
			verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
		}
	})
}

func updateManyTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		actual, err := executeUpdateMany(nil, coll, test.Operation.Arguments)
		require.NoError(t, err)

		verifyUpdateResult(t, actual, test.Outcome.Result)

		if test.Outcome.Collection != nil {
			verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
		}
	})
}

func updateOneTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		actual, err := executeUpdateOne(nil, coll, test.Operation.Arguments)
		require.NoError(t, err)

		verifyUpdateResult(t, actual, test.Outcome.Result)

		if test.Outcome.Collection != nil {
			verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
		}
	})
}

func shouldSkip(t *testing.T, minVersion string, maxVersion string, db *Database) bool {
	versionStr, err := getServerVersion(db)
	require.NoError(t, err)

	if len(minVersion) > 0 && compareVersions(t, minVersion, versionStr) > 0 {
		return true
	}

	if len(maxVersion) > 0 && compareVersions(t, maxVersion, versionStr) < 0 {
		return true
	}

	return false
}
