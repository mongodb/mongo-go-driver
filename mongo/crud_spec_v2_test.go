package mongo

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"math"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/launchpadcentral/mongo-driver/bson"
	"github.com/launchpadcentral/mongo-driver/bson/bsontype"
	"github.com/launchpadcentral/mongo-driver/event"
	testhelpers "github.com/launchpadcentral/mongo-driver/internal/testutil/helpers"
	"github.com/launchpadcentral/mongo-driver/mongo/options"
	"github.com/launchpadcentral/mongo-driver/mongo/readconcern"
	"github.com/launchpadcentral/mongo-driver/mongo/writeconcern"
	"github.com/launchpadcentral/mongo-driver/x/bsonx"
)

type testFileV2 struct {
	Data             json.RawMessage
	MinServerVersion string
	MaxServerVersion string
	DatabaseName     string `json:"database_name"`
	CollectionName   string `json:"collection_name"`
	Tests            []testCaseV2
}

type testCaseV2 struct {
	Description   string
	SkipReason    string
	ClientOptions collOpts `json:"collectionOptions"`
	Operations    []op
	Outcome       *outcome
	Expectations  []*expectation
}

type collOpts struct {
	ReadConcern *struct {
		Level string `json:"level"`
	} `json:"readConcern"`
}

const v2Dir = "v2"

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
		if err != nil {
			return 1
		}

		i2, err := strconv.Atoi(n2[i])
		if err != nil {
			return -1
		}

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

func TestCRUDSpecV2(t *testing.T) {
	for _, file := range testhelpers.FindJSONFilesInDir(t, path.Join(crudTestsDir, v2Dir)) {
		runCRUDTestFileV2(t, path.Join(crudTestsDir, v2Dir, file))
	}
}

func runCRUDTestFileV2(t *testing.T, filepath string) {
	content, err := ioutil.ReadFile(filepath)
	require.NoError(t, err)

	var testfile testFileV2
	require.NoError(t, json.Unmarshal(content, &testfile))

	dbName := "crud-spec-tests"
	if testfile.DatabaseName != "" {
		dbName = testfile.DatabaseName
	}
	client := createMonitoredClient(t, monitor)
	db := client.Database(dbName)

	if shouldSkip(t, testfile.MinServerVersion, testfile.MaxServerVersion, db) {
		return
	}

	for _, test := range testfile.Tests {
		t.Run(test.Description, func(t *testing.T) {
			collName := sanitizeCollectionName("crud-spec-tests", testfile.CollectionName)

			_ = db.RunCommand(
				context.Background(),
				bson.D{{"drop", collName}},
			)

			if test.Outcome != nil && test.Outcome.Collection != nil && len(test.Outcome.Collection.Name) > 0 {
				_ = db.RunCommand(
					context.Background(),
					bson.D{{"drop", test.Outcome.Collection.Name}},
				)
			}

			coll := db.Collection(collName, options.Collection().SetWriteConcern(writeconcern.New(writeconcern.WMajority())))

			if len(testfile.Data) != 0 {
				docsToInsert := docSliceToInterfaceSlice(docSliceFromRaw(t, testfile.Data))
				_, err = coll.InsertMany(context.Background(), docsToInsert)
				require.NoError(t, err)
			}

			testOperations(t, coll, db, &test)
		})
	}
}

func testOperations(t *testing.T, coll *Collection, db *Database, test *testCaseV2) {
	if test.Expectations != nil && len(test.Expectations) != 0 {
		drainChannels()
	}

	for _, operation := range test.Operations {
		switch operation.Name {
		case "aggregate":
			runOperationAggregate(t, coll, db, &operation, test)
		default:
			t.Fatalf("Unknown operation name: %v", operation.Name)
		}
	}

	checkCrudExpectations(t, test.Expectations)

	if test.Outcome != nil {
		collName := sanitizeCollectionName("crud-spec-tests", test.Description)
		outColl := db.Collection(collName)
		if test.Outcome.Collection != nil {
			if len(test.Outcome.Collection.Name) > 0 {
				outColl = db.Collection(test.Outcome.Collection.Name)
			}
			verifyCollectionContents(t, outColl, test.Outcome.Collection.Data)
		}
	}
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

func runOperationAggregate(t *testing.T, coll *Collection, db *Database, oper *op, test *testCaseV2) {
	var a aggregator = db
	var err error
	if oper.Object == "collection" {
		if coll == nil {
			t.Fatalf("collection was not properly made: %v", coll)
		}
		if oper.CollectionOptions != nil && oper.CollectionOptions.ReadConcern != nil {
			if oper.CollectionOptions.ReadConcern.Level == "" {
				t.Fatalf("unspecified level in specified readConcern")
			}

			coll, err = coll.Clone(&options.CollectionOptions{
				ReadConcern: readconcern.New(readconcern.Level(oper.CollectionOptions.ReadConcern.Level)),
			})
			if err != nil {
				t.Fatalf("unable to clone collection: %v", err)
			}
		}
		a = coll
	}

	pipeline := oper.Arguments["pipeline"].([]interface{})

	opts := options.Aggregate()
	if batchSize, found := oper.Arguments["batchSize"]; found {
		opts = opts.SetBatchSize(int32(batchSize.(float64)))
	}

	if collation, found := oper.Arguments["collation"]; found {
		opts = opts.SetCollation(collationFromMap(collation.(map[string]interface{})))
	}

	if diskUse, found := oper.Arguments["allowDiskUse"]; found {
		opts = opts.SetAllowDiskUse(diskUse.(bool))
	}

	var out bool
	if len(pipeline) > 0 {
		if _, found := pipeline[len(pipeline)-1].(map[string]interface{})["$out"]; found {
			out = true
		}
	}

	cursor, err := a.Aggregate(context.Background(), pipeline, opts)
	errored := err != nil
	if errored != oper.Error {
		t.Fatalf("got error: %v; expected an error: %v", err, oper.Error)
	}

	if !out && oper.Result != nil {
		require.NotNil(t, cursor)
		verifyCursorResult(t, cursor, oper.Result)
	}
}

func checkCrudExpectations(t *testing.T, expectations []*expectation) {
	for _, expectation := range expectations {
		var evt *event.CommandStartedEvent
		select {
		case evt = <-startedChan:
		default:
			require.Fail(t, "Expected command started event", expectation.CommandStartedEvent.CommandName)
		}

		if expectation.CommandStartedEvent.CommandName != "" {
			require.Equal(t, expectation.CommandStartedEvent.CommandName, evt.CommandName)
		}
		if expectation.CommandStartedEvent.DatabaseName != "" {
			require.Equal(t, expectation.CommandStartedEvent.DatabaseName, evt.DatabaseName)
		}

		jsonBytes, err := expectation.CommandStartedEvent.Command.MarshalJSON()
		require.NoError(t, err)

		expected := bsonx.Doc{}
		err = bson.UnmarshalExtJSON(jsonBytes, true, &expected)
		require.NoError(t, err)

		actual := evt.Command
		for _, elem := range expected {
			key := elem.Key
			val := elem.Value

			actualVal := actual.Lookup(key)

			// Keys that may be nil
			if val.Type() == bson.TypeNull {
				require.Equal(t, actual.Lookup(key), bson.RawValue{}, "Expected %s to be nil", key)
				continue
			} else if key == "ordered" {
				// TODO: some tests specify that "ordered" must be a key in the event but ordered isn't a valid option for some of these cases (e.g. insertOne)
				continue
			}

			// Keys that should not be nil
			require.NotEqual(t, actualVal.Type, bsontype.Null, "Expected %v, got nil for key: %s", elem, key)
			require.NoError(t, actualVal.Validate(), "Expected %v, couldn't validate", elem)
			if key == "getMore" {
				require.NotNil(t, actualVal, "Expected %v, got nil for key: %s", elem, key)
				expectedCursorID := val.Int64()
				// ignore if equal to 42
				if expectedCursorID != 42 {
					require.Equal(t, expectedCursorID, actualVal.Int64())
				}
				continue
			}
			if key == "readConcern" {
				rcExpectDoc := val.Document()
				rcActualDoc := actualVal.Document()
				clusterTime := rcExpectDoc.Lookup("afterClusterTime")
				level := rcExpectDoc.Lookup("level")
				if clusterTime.Type() != bsontype.Null {
					require.NotNil(t, rcActualDoc.Lookup("afterClusterTime"))
				}
				if level.Type() != bsontype.Null {
					doc, err := bsonx.ReadDoc(rcActualDoc)
					require.NoError(t, err)
					compareElements(t, rcExpectDoc.LookupElement("level"), doc.LookupElement("level"))
				}
				continue
			}
			doc, err := bsonx.ReadDoc(actual)
			require.NoError(t, err)
			compareElements(t, elem, doc.LookupElement(key))
		}
	}
}
