package gridfs

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	testhelpers "go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"os"
	"testing"
)

type runOn struct {
	MinServerVersion string   `json:"minServerVersion"`
	MaxServerVersion string   `json:"maxServerVersion"`
	Topology         []string `json:"topology"`
}

type failPoint struct {
	ConfigureFailPoint string          `json:"configureFailPoint"`
	Mode               json.RawMessage `json:"mode"`
	Data               *failPointData  `json:"data"`
}

type failPointData struct {
	FailCommands                  []string `json:"failCommands"`
	CloseConnection               bool     `json:"closeConnection"`
	ErrorCode                     int32    `json:"errorCode"`
	FailBeforeCommitExceptionCode int32    `json:"failBeforeCommitExceptionCode"`
	WriteConcernError             *struct {
		Code   int32  `json:"code"`
		Name   string `json:"codeName"`
		Errmsg string `json:"errmsg"`
	} `json:"writeConcernError"`
}

type op struct {
	Name      string
	Arguments struct {
		Filename string `json:"filename"`
		Oid      primitive.ObjectID `json:"id"`
	} `json:"arguments"`
	Object            string
	CollectionOptions *collOpts `json:"collectionOptions"`
	Error             bool
	Result            json.RawMessage
}

type collOpts struct {
	ReadConcern *struct {
		Level string `json:"level"`
	} `json:"readConcern"`
}

type expectation struct {
	CommandStartedEvent struct {
		CommandName  string          `json:"command_name"`
		DatabaseName string          `json:"database_name"`
		Command      json.RawMessage `json:"command"`
	} `json:"command_started_event"`
}

func skipIfNecessaryRunOnSlice(t *testing.T, runOns []runOn, dbAdmin *mongo.Database) {
	for _, runOn := range runOns {
		if canRunOn(t, runOn, dbAdmin) {
			return
		}
	}
	versionStr, err := getServerVersion(dbAdmin)
	require.NoError(t, err, "unable to run on current server version, topology combination, error getting server version")
	t.Skipf("unable to run on %v %v", os.Getenv("TOPOLOGY"), versionStr)
}

func getServerVersion(db *mongo.Database) (string, error) {
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

func canRunOn(t *testing.T, runOn runOn, dbAdmin *mongo.Database) bool {
	if shouldSkip(t, runOn.MinServerVersion, runOn.MaxServerVersion, dbAdmin) {
		return false
	}

	for _, top := range runOn.Topology {
		if os.Getenv("TOPOLOGY") == runOnTopologyToEnvTopology(t, top) {
			return true
		}
	}
	return false
}

func shouldSkip(t *testing.T, minVersion string, maxVersion string, db *mongo.Database) bool {
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

func runOnTopologyToEnvTopology(t *testing.T, runOnTopology string) string {
	switch runOnTopology {
	case "single":
		return "server"
	case "replicaset":
		return "replica_set"
	case "sharded":
		return "sharded_cluster"
	default:
		t.Fatalf("unknown topology%v", runOnTopology)
	}
	return ""
}

// load initial data files into a files and chunks collection
// returns the chunkSize embedded in the documents if there is one
func loadInitialFiles(t *testing.T, filesJSONs, chunksJSONs []json.RawMessage) int32 {
	if filesJSONs == nil || chunksJSONs == nil {
		return 0
	}
	filesDocs := make([]interface{}, 0, len(filesJSONs))
	chunksDocs := make([]interface{}, 0, len(chunksJSONs))
	var chunkSize int32

	for _, v := range filesJSONs {
		docBytes, err := v.MarshalJSON()
		testhelpers.RequireNil(t, err, "error converting raw message to bytes: %s", err)
		doc := bsonx.Doc{}
		err = bson.UnmarshalExtJSON(docBytes, false, &doc)
		testhelpers.RequireNil(t, err, "error creating file document: %s", err)

		// convert length from int32 to int64
		if length, err := doc.LookupErr("length"); err == nil {
			doc = doc.Delete("length")
			doc = doc.Append("length", bsonx.Int64(int64(length.Int32())))
		}
		if cs, err := doc.LookupErr("chunkSize"); err == nil {
			chunkSize = cs.Int32()
		}

		filesDocs = append(filesDocs, doc)
	}

	for _, v := range chunksJSONs {
		docBytes, err := v.MarshalJSON()
		testhelpers.RequireNil(t, err, "error converting raw message to bytes: %s", err)
		doc := bsonx.Doc{}
		err = bson.UnmarshalExtJSON(docBytes, false, &doc)
		testhelpers.RequireNil(t, err, "error creating file document: %s", err)

		// convert data $hex to binary value
		if hexStr, err := doc.LookupErr("data", "$hex"); err == nil {
			hexBytes := convertHexToBytes(t, hexStr.StringValue())
			doc = doc.Delete("data")
			doc = append(doc, bsonx.Elem{"data", bsonx.Binary(0x00, hexBytes)})
		}

		// convert n from int64 to int32
		if n, err := doc.LookupErr("n"); err == nil {
			doc = doc.Delete("n")
			doc = append(doc, bsonx.Elem{"n", bsonx.Int32(n.Int32())})
		}

		chunksDocs = append(chunksDocs, doc)
	}

	var res *mongo.InsertManyResult
	var err error
	if len(filesDocs) > 0 {
		res, err = files.InsertMany(ctx, filesDocs)
		testhelpers.RequireNil(t, err, "error inserting into files: %s", err)
		_, err = expectedFiles.InsertMany(ctx, filesDocs)
		testhelpers.RequireNil(t, err, "error inserting into expected files: %s", err)
	}

	for _, d := range res.InsertedIDs {
		fmt.Printf("files inserted doc: %s\n", d)
	}

	if len(chunksDocs) > 0 {
		res, err = chunks.InsertMany(ctx, chunksDocs)
		testhelpers.RequireNil(t, err, "error inserting into chunks: %s", err)
		_, err = expectedChunks.InsertMany(ctx, chunksDocs)
		testhelpers.RequireNil(t, err, "error inserting into expected chunks: %s", err)
	}

	for _, d := range res.InsertedIDs {
		fmt.Printf("chunks inserted doc: %s\n", d)
	}

	return chunkSize
}

func dropColl(t *testing.T, c *mongo.Collection) {
	err := c.Drop(ctx)
	testhelpers.RequireNil(t, err, "error dropping %s: %s", c.Name(), err)
}

func clearCollections(t *testing.T) {
	dropColl(t, files)
	dropColl(t, expectedFiles)
	dropColl(t, chunks)
	dropColl(t, expectedChunks)
}

func compareElements(t *testing.T, expected bsonx.Elem, actual bsonx.Elem) {
	if expected.Value.IsNumber() {
		if expectedNum, ok := expected.Value.Int64OK(); ok {
			switch actual.Value.Type() {
			case bson.TypeInt32:
				require.Equal(t, expectedNum, int64(actual.Value.Int32()), "For key %v", expected.Key)
			case bson.TypeInt64:
				require.Equal(t, expectedNum, actual.Value.Int64(), "For key %v\n", expected.Key)
			case bson.TypeDouble:
				require.Equal(t, expectedNum, int64(actual.Value.Double()), "For key %v\n", expected.Key)
			}
		} else {
			expectedNum := expected.Value.Int32()
			switch actual.Value.Type() {
			case bson.TypeInt32:
				require.Equal(t, expectedNum, actual.Value.Int32(), "For key %v", expected.Key)
			case bson.TypeInt64:
				require.Equal(t, expectedNum, int32(actual.Value.Int64()), "For key %v\n", expected.Key)
			case bson.TypeDouble:
				require.Equal(t, expectedNum, int32(actual.Value.Double()), "For key %v\n", expected.Key)
			}
		}
	} else if conv, ok := expected.Value.DocumentOK(); ok {
		actualConv, actualOk := actual.Value.DocumentOK()
		require.True(t, actualOk)
		compareDocs(t, conv, actualConv)
	} else if conv, ok := expected.Value.ArrayOK(); ok {
		actualConv, actualOk := actual.Value.ArrayOK()
		require.True(t, actualOk)
		compareArrays(t, conv, actualConv)
	} else {
		require.True(t, actual.Equal(expected), "For key %s, expected %v\nactual: %v", expected.Key, expected, actual)
	}
}

func compareDocs(t *testing.T, expected bsonx.Doc, actual bsonx.Doc) {
	// this is necessary even though Equal() exists for documents because types not match between commands and the BSON
	// documents given in test cases. for example, all numbers in the test case JSON are parsed as int64, but many nubmers
	// sent over the wire are type int32
	if len(expected) != len(actual) {
		t.Errorf("doc length mismatch. expected %d got %d", len(expected), len(actual))
		t.FailNow()
	}

	for _, expectedElem := range expected {

		aVal, err := actual.LookupErr(expectedElem.Key)
		testhelpers.RequireNil(t, err, "docs not equal. key %s not found in actual", expectedElem.Key)

		eVal := expectedElem.Value

		if doc, ok := eVal.DocumentOK(); ok {
			// nested doc
			compareDocs(t, doc, aVal.Document())

			// nested docs were equal
			continue
		}

		if !compareValues(eVal, aVal) {
			t.Errorf("docs not equal because value mismatch for key %s", expectedElem.Key)
		}
	}
}

func compareArrays(t *testing.T, expected bsonx.Arr, actual bsonx.Arr) {
	if len(expected) != len(actual) {
		t.Errorf("array length mismatch. expected %d got %d", len(expected), len(actual))
		t.FailNow()
	}

	for idx := range expected {
		expectedDoc := expected[idx].Document()
		actualDoc := actual[idx].Document()
		compareDocs(t, expectedDoc, actualDoc)
	}
}
