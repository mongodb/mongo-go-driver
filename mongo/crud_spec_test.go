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

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/aggregateopt"
	"github.com/mongodb/mongo-go-driver/mongo/countopt"
	"github.com/mongodb/mongo-go-driver/mongo/deleteopt"
	"github.com/mongodb/mongo-go-driver/mongo/distinctopt"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
	"github.com/mongodb/mongo-go-driver/mongo/replaceopt"
	"github.com/mongodb/mongo-go-driver/mongo/updateopt"
	"github.com/stretchr/testify/require"
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
	serverStatus, err := db.RunCommand(
		context.Background(),
		bson.NewDocument(bson.EC.Int32("serverStatus", 1)),
	)
	if err != nil {
		return "", err
	}

	version, err := serverStatus.Lookup("version")
	if err != nil {
		return "", err
	}

	return version.Value().StringValue(), nil
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

func sanitizeCollectionName(name string) string {
	// Collections can't have "$" in their names, so we substitute it with "%".
	name = strings.Replace(name, "$", "%", -1)

	// Namespaces can only have 120 bytes max.
	if len("crud-spec-tests."+name) >= 119 {
		name = name[:119-len("crud-spec-tests.")]
	}

	return name
}

func runCRUDTestFile(t *testing.T, filepath string, db *Database) {
	content, err := ioutil.ReadFile(filepath)
	require.NoError(t, err)

	var testfile testFile
	require.NoError(t, json.Unmarshal(content, &testfile))

	if shouldSkip(t, &testfile, db) {
		return
	}

	for _, test := range testfile.Tests {
		collName := sanitizeCollectionName(test.Description)

		_, _ = db.RunCommand(
			context.Background(),
			bson.NewDocument(bson.EC.String("drop", collName)),
		)

		if test.Outcome.Collection != nil && len(test.Outcome.Collection.Name) > 0 {
			_, _ = db.RunCommand(
				context.Background(),
				bson.NewDocument(bson.EC.String("drop", test.Outcome.Collection.Name)),
			)
		}

		coll := db.Collection(collName)
		docsToInsert := docSliceToInterfaceSlice(docSliceFromRaw(t, testfile.Data))
		_, err = coll.InsertMany(context.Background(), docsToInsert)
		require.NoError(t, err)

		switch test.Operation.Name {
		case "aggregate":
			aggregateTest(t, db, coll, &test)
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

		opts := aggregateopt.BundleAggregate()

		if batchSize, found := test.Operation.Arguments["batchSize"]; found {
			opts = opts.BatchSize(int32(batchSize.(float64)))
		}

		if collation, found := test.Operation.Arguments["collation"]; found {
			opts = opts.Collation(collationFromMap(collation.(map[string]interface{})))
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
			verifyCursorResults(t, cursor, test.Outcome.Result)
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

func countTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		filter := test.Operation.Arguments["filter"].(map[string]interface{})

		var opts []countopt.Count

		if skip, found := test.Operation.Arguments["skip"]; found {
			opts = append(opts, countopt.Skip(int64(skip.(float64))))
		}

		if limit, found := test.Operation.Arguments["limit"]; found {
			opts = append(opts, countopt.Limit(int64(limit.(float64))))
		}

		if collation, found := test.Operation.Arguments["collation"]; found {
			opts = append(opts, countopt.Collation(collationFromMap(collation.(map[string]interface{}))))
		}

		actualCount, err := coll.Count(context.Background(), filter, opts...)
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
		filter := test.Operation.Arguments["filter"].(map[string]interface{})

		var opts []deleteopt.Delete

		if collation, found := test.Operation.Arguments["collation"]; found {
			mapCollation := collation.(map[string]interface{})
			opts = append(opts, deleteopt.Collation(collationFromMap(mapCollation)))
		}

		actual, err := coll.DeleteMany(context.Background(), filter, opts...)
		require.NoError(t, err)

		expectedBytes, err := test.Outcome.Result.MarshalJSON()
		require.NoError(t, err)

		var expected DeleteResult
		err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)
		require.NoError(t, err)

		require.Equal(t, expected.DeletedCount, actual.DeletedCount)

		verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
	})
}

func deleteOneTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		filter := test.Operation.Arguments["filter"].(map[string]interface{})

		var opts []deleteopt.Delete

		if collation, found := test.Operation.Arguments["collation"]; found {
			mapCollation := collationFromMap(collation.(map[string]interface{}))
			opts = append(opts, deleteopt.Collation(mapCollation))
		}

		actual, err := coll.DeleteOne(context.Background(), filter, opts...)
		require.NoError(t, err)

		expectedBytes, err := test.Outcome.Result.MarshalJSON()
		require.NoError(t, err)

		var expected DeleteResult
		err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)
		require.NoError(t, err)

		require.Equal(t, expected.DeletedCount, actual.DeletedCount)

		if test.Outcome.Collection != nil {
			verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
		}
	})
}

func distinctTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		fieldName := test.Operation.Arguments["fieldName"].(string)

		var filter map[string]interface{}

		if filterArg, found := test.Operation.Arguments["filter"]; found {
			filter = filterArg.(map[string]interface{})
		}

		var opts []distinctopt.Distinct

		if collation, found := test.Operation.Arguments["collation"]; found {
			mapCollation := collationFromMap(collation.(map[string]interface{}))
			opts = append(opts, distinctopt.Collation(mapCollation))
		}

		actual, err := coll.Distinct(context.Background(), fieldName, filter, opts...)
		require.NoError(t, err)

		resultBytes, err := test.Outcome.Result.MarshalJSON()
		require.NoError(t, err)

		var expected []interface{}
		require.NoError(t, json.NewDecoder(bytes.NewBuffer(resultBytes)).Decode(&expected))

		require.Equal(t, len(expected), len(actual))

		for i := range expected {
			expectedElem := expected[i]
			actualElem := actual[i]

			iExpected := testhelpers.GetIntFromInterface(expectedElem)
			iActual := testhelpers.GetIntFromInterface(actualElem)

			require.Equal(t, iExpected == nil, iActual == nil)
			if iExpected != nil {
				require.Equal(t, *iExpected, *iActual)
				continue
			}

			require.Equal(t, expected[i], actual[i])
		}
	})
}

func findTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		filter := test.Operation.Arguments["filter"].(map[string]interface{})

		var opts []findopt.Find

		if sort, found := test.Operation.Arguments["sort"]; found {
			opts = append(opts, findopt.Sort(sort.(map[string]interface{})))
		}

		if skip, found := test.Operation.Arguments["skip"]; found {
			opts = append(opts, findopt.Skip(int64(skip.(float64))))
		}

		if limit, found := test.Operation.Arguments["limit"]; found {
			opts = append(opts, findopt.Limit(int64(limit.(float64))))
		}

		if batchSize, found := test.Operation.Arguments["batchSize"]; found {
			opts = append(opts, findopt.BatchSize(int32(batchSize.(float64))))
		}

		if collation, found := test.Operation.Arguments["collation"]; found {
			opts = append(opts, findopt.Collation(collationFromMap(collation.(map[string]interface{}))))
		}

		cursor, err := coll.Find(context.Background(), filter, opts...)
		require.NoError(t, err)

		verifyCursorResults(t, cursor, test.Outcome.Result)
	})
}

func findOneAndDeleteTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		filter := test.Operation.Arguments["filter"].(map[string]interface{})

		var opts []findopt.DeleteOne

		if sort, found := test.Operation.Arguments["sort"]; found {
			opts = append(opts, findopt.Sort(sort.(map[string]interface{})))
		}

		if projection, found := test.Operation.Arguments["projection"]; found {
			opts = append(opts, findopt.Projection(projection.(map[string]interface{})))
		}

		if collation, found := test.Operation.Arguments["collation"]; found {
			opts = append(opts, findopt.Collation(collationFromMap(collation.(map[string]interface{}))))
		}

		actualResult := coll.FindOneAndDelete(context.Background(), filter, opts...)

		jsonBytes, err := test.Outcome.Result.MarshalJSON()
		require.NoError(t, err)

		actual := bson.NewDocument()
		err = actualResult.Decode(actual)
		if err == ErrNoDocuments {
			var expected map[string]interface{}
			err := json.NewDecoder(bytes.NewBuffer(jsonBytes)).Decode(&expected)
			require.NoError(t, err)

			require.Nil(t, expected)
			return
		}

		require.NoError(t, err)

		doc, err := bson.ParseExtJSONObject(string(jsonBytes))
		require.NoError(t, err)

		require.True(t, doc.Equal(actual))

		verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
	})
}

func findOneAndReplaceTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		filter := test.Operation.Arguments["filter"].(map[string]interface{})
		replacement := test.Operation.Arguments["replacement"].(map[string]interface{})

		// For some reason, the filter and replacement documents are unmarshaled with floats
		// rather than integers, but the documents that are used to initially populate the
		// collection are unmarshaled correctly with integers. To ensure that the tests can
		// correctly compare them, we iterate through the filter and replacement documents and
		// change any valid integers stored as floats to actual integers.
		replaceFloatsWithInts(filter)
		replaceFloatsWithInts(replacement)

		var opts []findopt.ReplaceOne

		if projection, found := test.Operation.Arguments["projection"]; found {
			opts = append(opts, findopt.Projection(projection.(map[string]interface{})))
		}

		if returnDocument, found := test.Operation.Arguments["returnDocument"]; found {
			switch returnDocument.(string) {
			case "After":
				opts = append(opts, findopt.ReturnDocument(mongoopt.After))
			case "Before":
				opts = append(opts, findopt.ReturnDocument(mongoopt.Before))
			}
		}

		if sort, found := test.Operation.Arguments["sort"]; found {
			opts = append(opts, findopt.Sort(sort.(map[string]interface{})))
		}

		if upsert, found := test.Operation.Arguments["upsert"]; found {
			opts = append(opts, findopt.Upsert(upsert.(bool)))
		}

		if collation, found := test.Operation.Arguments["collation"]; found {
			opts = append(opts, findopt.Collation(collationFromMap(collation.(map[string]interface{}))))
		}

		actualResult := coll.FindOneAndReplace(context.Background(), filter, replacement, opts...)

		jsonBytes, err := test.Outcome.Result.MarshalJSON()
		require.NoError(t, err)

		actual := bson.NewDocument()
		err = actualResult.Decode(actual)
		if err == ErrNoDocuments {
			var expected map[string]interface{}
			err := json.NewDecoder(bytes.NewBuffer(jsonBytes)).Decode(&expected)
			require.NoError(t, err)

			require.Nil(t, expected)
			return
		}

		require.NoError(t, err)
		doc, err := bson.ParseExtJSONObject(string(jsonBytes))
		require.NoError(t, err)

		require.True(t, doc.Equal(actual))

		verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
	})
}

func findOneAndUpdateTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		filter := test.Operation.Arguments["filter"].(map[string]interface{})
		update := test.Operation.Arguments["update"].(map[string]interface{})

		// For some reason, the filter and update documents are unmarshaled with floats
		// rather than integers, but the documents that are used to initially populate the
		// collection are unmarshaled correctly with integers. To ensure that the tests can
		// correctly compare them, we iterate through the filter and update documents and
		// change any valid integers stored as floats to actual integers.
		replaceFloatsWithInts(filter)
		replaceFloatsWithInts(update)

		var opts []findopt.UpdateOne

		if arrayFilters, found := test.Operation.Arguments["arrayFilters"]; found {
			arrayFiltersSlice := arrayFilters.([]interface{})
			opts = append(opts, findopt.ArrayFilters(arrayFiltersSlice...))
		}

		if projection, found := test.Operation.Arguments["projection"]; found {
			opts = append(opts, findopt.Projection(projection.(map[string]interface{})))
		}

		if returnDocument, found := test.Operation.Arguments["returnDocument"]; found {
			switch returnDocument.(string) {
			case "After":
				opts = append(opts, findopt.ReturnDocument(mongoopt.After))
			case "Before":
				opts = append(opts, findopt.ReturnDocument(mongoopt.Before))
			}
		}

		if sort, found := test.Operation.Arguments["sort"]; found {
			opts = append(opts, findopt.Sort(sort.(map[string]interface{})))
		}

		if upsert, found := test.Operation.Arguments["upsert"]; found {
			opts = append(opts, findopt.Upsert(upsert.(bool)))
		}

		if collation, found := test.Operation.Arguments["collation"]; found {
			opts = append(opts, findopt.Collation(collationFromMap(collation.(map[string]interface{}))))
		}

		actualResult := coll.FindOneAndUpdate(context.Background(), filter, update, opts...)

		jsonBytes, err := test.Outcome.Result.MarshalJSON()
		require.NoError(t, err)

		actual := bson.NewDocument()
		err = actualResult.Decode(actual)
		if err == ErrNoDocuments {
			var expected map[string]interface{}
			err := json.NewDecoder(bytes.NewBuffer(jsonBytes)).Decode(&expected)
			require.NoError(t, err)

			require.Nil(t, expected)
			return
		}

		require.NoError(t, err)

		doc, err := bson.ParseExtJSONObject(string(jsonBytes))
		require.NoError(t, err)

		require.True(t, doc.Equal(actual))

		verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
	})
}

func insertManyTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		documents := test.Operation.Arguments["documents"].([]interface{})

		// For some reason, the insertion documents are unmarshaled with a float rather than
		// integer, but the documents that are used to initially populate the collection are
		// unmarshaled correctly with integers. To ensure that the tests can correctly compare
		// them, we iterate through the insertion documents and change any valid integers stored
		// as floats to actual integers.
		for i, doc := range documents {
			docM := doc.(map[string]interface{})
			replaceFloatsWithInts(docM)

			documents[i] = docM
		}

		actual, err := coll.InsertMany(context.Background(), documents)
		require.NoError(t, err)

		expectedBytes, err := test.Outcome.Result.MarshalJSON()
		require.NoError(t, err)

		var expected struct{ InsertedIds map[string]interface{} }
		err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)
		require.NoError(t, err)

		replaceFloatsWithInts(expected.InsertedIds)

		for i, elem := range actual.InsertedIDs {
			actual.InsertedIDs[i] = elem.(*bson.Element).Value().Interface()
		}

		for _, val := range expected.InsertedIds {
			require.Contains(t, actual.InsertedIDs, val)
		}

		verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
	})
}

func insertOneTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		document := test.Operation.Arguments["document"].(map[string]interface{})

		// For some reason, the insertion document is unmarshaled with a float rather than integer,
		// but the documents that are used to initially populate the collection are unmarshaled
		// correctly with integers. To ensure that the tests can correctly compare them, we iterate
		// through the insertion document and change any valid integers stored as floats to actual
		// integers.
		replaceFloatsWithInts(document)

		actual, err := coll.InsertOne(context.Background(), document)
		require.NoError(t, err)

		expectedBytes, err := test.Outcome.Result.MarshalJSON()
		require.NoError(t, err)

		var expected InsertOneResult
		err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)

		expectedID := expected.InsertedID
		if f, ok := expectedID.(float64); ok && f == math.Floor(f) {
			expectedID = int64(f)
		}

		require.Equal(t, expectedID, actual.InsertedID.(*bson.Element).Value().Interface())

		verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
	})
}

func replaceOneTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		filter := test.Operation.Arguments["filter"].(map[string]interface{})
		replacement := test.Operation.Arguments["replacement"].(map[string]interface{})

		// For some reason, the filter and replacement documents are unmarshaled with floats
		// rather than integers, but the documents that are used to initially populate the
		// collection are unmarshaled correctly with integers. To ensure that the tests can
		// correctly compare them, we iterate through the filter and replacement documents and
		// change any valid integers stored as floats to actual integers.
		replaceFloatsWithInts(filter)
		replaceFloatsWithInts(replacement)

		var opts []replaceopt.Replace

		if upsert, found := test.Operation.Arguments["upsert"]; found {
			opts = append(opts, replaceopt.Upsert(upsert.(bool)))
		}

		if collation, found := test.Operation.Arguments["collation"]; found {
			opts = append(opts, replaceopt.Collation(collationFromMap(collation.(map[string]interface{}))))
		}

		actual, err := coll.ReplaceOne(context.Background(), filter, replacement, opts...)
		require.NoError(t, err)

		expectedBytes, err := test.Outcome.Result.MarshalJSON()
		require.NoError(t, err)

		var expected struct {
			MatchedCount  int64
			ModifiedCount int64
			UpsertedCount int64
		}
		err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)

		require.Equal(t, expected.MatchedCount, actual.MatchedCount)
		require.Equal(t, expected.ModifiedCount, actual.ModifiedCount)

		actualUpsertedCount := int64(0)
		if actual.UpsertedID != nil {
			actualUpsertedCount = 1
		}

		require.Equal(t, expected.UpsertedCount, actualUpsertedCount)

		if test.Outcome.Collection != nil {
			verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
		}
	})
}

func updateManyTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		filter := test.Operation.Arguments["filter"].(map[string]interface{})
		update := test.Operation.Arguments["update"].(map[string]interface{})

		// For some reason, the filter and update documents are unmarshaled with floats
		// rather than integers, but the documents that are used to initially populate the
		// collection are unmarshaled correctly with integers. To ensure that the tests can
		// correctly compare them, we iterate through the filter and update documents and
		// change any valid integers stored as floats to actual integers.
		replaceFloatsWithInts(filter)
		replaceFloatsWithInts(update)

		var opts []updateopt.Update

		if arrayFilters, found := test.Operation.Arguments["arrayFilters"]; found {
			arrayFiltersSlice := arrayFilters.([]interface{})
			opts = append(opts, updateopt.ArrayFilters(arrayFiltersSlice...))
		}

		if upsert, found := test.Operation.Arguments["upsert"]; found {
			opts = append(opts, updateopt.Upsert(upsert.(bool)))
		}

		if collation, found := test.Operation.Arguments["collation"]; found {
			opts = append(opts, updateopt.Collation(collationFromMap(collation.(map[string]interface{}))))
		}

		actual, err := coll.UpdateMany(context.Background(), filter, update, opts...)
		require.NoError(t, err)

		expectedBytes, err := test.Outcome.Result.MarshalJSON()
		require.NoError(t, err)

		var expected struct {
			MatchedCount  int64
			ModifiedCount int64
			UpsertedCount int64
		}
		err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)

		require.Equal(t, expected.MatchedCount, actual.MatchedCount)
		require.Equal(t, expected.ModifiedCount, actual.ModifiedCount)

		actualUpsertedCount := int64(0)
		if actual.UpsertedID != nil {
			actualUpsertedCount = 1
		}

		require.Equal(t, expected.UpsertedCount, actualUpsertedCount)

		if test.Outcome.Collection != nil {
			verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
		}
	})
}

func updateOneTest(t *testing.T, coll *Collection, test *testCase) {
	t.Run(test.Description, func(t *testing.T) {
		filter := test.Operation.Arguments["filter"].(map[string]interface{})
		update := test.Operation.Arguments["update"].(map[string]interface{})

		// For some reason, the filter and update documents are unmarshaled with floats
		// rather than integers, but the documents that are used to initially populate the
		// collection are unmarshaled correctly with integers. To ensure that the tests can
		// correctly compare them, we iterate through the filter and update documents and
		// change any valid integers stored as floats to actual integers.
		replaceFloatsWithInts(filter)
		replaceFloatsWithInts(update)

		var opts []updateopt.Update

		if arrayFilters, found := test.Operation.Arguments["arrayFilters"]; found {
			arrayFiltersSlice := arrayFilters.([]interface{})
			opts = append(opts, updateopt.ArrayFilters(arrayFiltersSlice...))
		}

		if upsert, found := test.Operation.Arguments["upsert"]; found {
			opts = append(opts, updateopt.Upsert(upsert.(bool)))
		}

		if collation, found := test.Operation.Arguments["collation"]; found {
			opts = append(opts, updateopt.Collation(collationFromMap(collation.(map[string]interface{}))))
		}

		actual, err := coll.UpdateOne(context.Background(), filter, update, opts...)
		require.NoError(t, err)

		expectedBytes, err := test.Outcome.Result.MarshalJSON()
		require.NoError(t, err)

		var expected struct {
			MatchedCount  int64
			ModifiedCount int64
			UpsertedCount int64
		}
		err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)

		require.Equal(t, expected.MatchedCount, actual.MatchedCount)
		require.Equal(t, expected.ModifiedCount, actual.ModifiedCount)

		actualUpsertedCount := int64(0)
		if actual.UpsertedID != nil {
			actualUpsertedCount = 1
		}

		require.Equal(t, expected.UpsertedCount, actualUpsertedCount)

		if test.Outcome.Collection != nil {
			verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
		}
	})
}

func shouldSkip(t *testing.T, test *testFile, db *Database) bool {
	versionStr, err := getServerVersion(db)
	require.NoError(t, err)

	if len(test.MinServerVersion) > 0 && compareVersions(t, test.MinServerVersion, versionStr) > 0 {
		return true
	}

	if len(test.MaxServerVersion) > 0 && compareVersions(t, test.MaxServerVersion, versionStr) < 0 {
		return true
	}

	return false
}

func verifyCollectionContents(t *testing.T, coll *Collection, result json.RawMessage) {
	cursor, err := coll.Find(context.Background(), nil)
	require.NoError(t, err)

	verifyCursorResults(t, cursor, result)
}

func verifyCursorResults(t *testing.T, cursor Cursor, result json.RawMessage) {
	for _, expected := range docSliceFromRaw(t, result) {
		require.True(t, cursor.Next(context.Background()))

		actual := bson.NewDocument()
		require.NoError(t, cursor.Decode(actual))

		require.True(t, expected.Equal(actual))
	}

	require.False(t, cursor.Next(context.Background()))
	require.NoError(t, cursor.Err())
}

func collationFromMap(m map[string]interface{}) *mongoopt.Collation {
	var collation mongoopt.Collation

	if locale, found := m["locale"]; found {
		collation.Locale = locale.(string)
	}

	if caseLevel, found := m["caseLevel"]; found {
		collation.CaseLevel = caseLevel.(bool)
	}

	if caseFirst, found := m["caseFirst"]; found {
		collation.CaseFirst = caseFirst.(string)
	}

	if strength, found := m["strength"]; found {
		collation.Strength = int(strength.(float64))
	}

	if numericOrdering, found := m["numericOrdering"]; found {
		collation.NumericOrdering = numericOrdering.(bool)
	}

	if alternate, found := m["alternate"]; found {
		collation.Alternate = alternate.(string)
	}

	if maxVariable, found := m["maxVariable"]; found {
		collation.MaxVariable = maxVariable.(string)
	}

	if backwards, found := m["backwards"]; found {
		collation.Backwards = backwards.(bool)
	}

	return &collation
}

func docSliceFromRaw(t *testing.T, raw json.RawMessage) []*bson.Document {
	jsonBytes, err := raw.MarshalJSON()
	require.NoError(t, err)

	array, err := bson.ParseExtJSONArray(string(jsonBytes))
	require.NoError(t, err)

	docs := make([]*bson.Document, 0)

	for i := 0; i < array.Len(); i++ {
		item, err := array.Lookup(uint(i))
		require.NoError(t, err)
		docs = append(docs, item.MutableDocument())
	}

	return docs
}

func docSliceToInterfaceSlice(docs []*bson.Document) []interface{} {
	out := make([]interface{}, 0, len(docs))

	for _, doc := range docs {
		out = append(out, doc)
	}

	return out
}

func replaceFloatsWithInts(m map[string]interface{}) {
	for key, val := range m {
		if f, ok := val.(float64); ok && f == math.Floor(f) {
			m[key] = int64(f)
			continue
		}

		if innerM, ok := val.(map[string]interface{}); ok {
			replaceFloatsWithInts(innerM)
			m[key] = innerM
		}
	}
}
