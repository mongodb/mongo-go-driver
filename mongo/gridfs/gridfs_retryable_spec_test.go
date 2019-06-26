package gridfs

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/testutil"
	testhelpers "go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"io/ioutil"
	"path"
	"testing"
)

const gridFSRetryDir = "../../data/gridfs/retryable-tests/"

var gridFSStartedChan = make(chan *event.CommandStartedEvent, 100)

var gridFSMonitor = &event.CommandMonitor{
	Started: func(ctx context.Context, cse *event.CommandStartedEvent) {
		gridFSStartedChan <- cse
	},
}

type gridFSRetryTestFile struct {
	RunOns       []runOn `json:"runOn"`
	DatabaseName string  `json:"database_name"`
	BucketName   string  `json:"bucket_name"`
	Data         struct {
		Files  []json.RawMessage `json:"fs.files"`
		Chunks []json.RawMessage `json:"fs.chunks"`
	} `json:"data"`
	Tests []gridFSRetryTest `json:"tests"`
}

type gridFSRetryTest struct {
	Description         string          `json:"description"`
	ClientOptions       json.RawMessage `json:"clientOptions"`
	UseMultipleMongoses bool            `json:"useMultipleMongoses"`
	SkipReason          string          `json:"skipReason"`
	FailPoint           *failPoint      `json:"failPoint"`
	Operations          []op            `json:"operations"`
	Expectations        []expectation   `json:"expectations"`
}

func TestGridFSRetryableReadSpec(t *testing.T) {
	for _, testFile := range testhelpers.FindJSONFilesInDir(t, gridFSRetryDir) {
		runGridFSRetryReadTestFile(t, testFile)
	}
}

func runGridFSRetryReadTestFile(t *testing.T, testFileName string) {

	content, err := ioutil.ReadFile(path.Join(gridFSRetryDir, testFileName))
	require.NoError(t, err)

	var testFile gridFSRetryTestFile
	err = json.Unmarshal(content, &testFile)
	require.NoError(t, err)

	t.Run(testFileName, func(t *testing.T) {
		cs := testutil.ConnString(t)
		client, err := mongo.NewClient(options.Client().ApplyURI(cs.String()))
		require.NoError(t, err, "unable to create client")

		err = client.Connect(ctx)
		require.NoError(t, err, "unable to connect client")

		dbName := testFile.DatabaseName
		if dbName == "" {
			dbName = "retryable-reads-tests"
		}
		db = client.Database(dbName)

		skipIfNecessaryRunOnSlice(t, testFile.RunOns, db)

		chunks = db.Collection("fs.chunks")
		files = db.Collection("fs.files")
		expectedChunks = db.Collection("expected.chunks")
		expectedFiles = db.Collection("expected.files")

		for _, test := range testFile.Tests {
			t.Run(test.Description, func(t *testing.T) {
				runRetryGridFSTest(t, &testFile, &test, db)
			})
		}
	})
}

func runRetryGridFSTest(t *testing.T, testFile *gridFSRetryTestFile, test *gridFSRetryTest, db *mongo.Database) {
	bucket, cleanup := setupTest(t, testFile, test, db)
	defer cleanup()

Loop:
	for _, op := range test.Operations {
		if op.Object != "gridfsbucket" && op.Object != "" {
			t.Fatalf("unrecognized op.Object: %v", op.Object)
		}

		switch op.Name {
		case "download":
			downloadStream := bytes.NewBuffer(downloadBuffer)
			//oid, err := strconv.Atoi(op.Arguments.Oid)
			//if err != nil {
			//	t.Fatalf("unable to parse oid out of testcase")
			//}
			_, err := bucket.DownloadToStream(bson.D{
				{"id", op.Arguments.Oid},
			}, downloadStream)
			if op.Error {
				require.Error(t, err, "download failed to error when expected")
				break Loop
			}
			require.NoError(t, err, "download errored unexpectedly")
		case "download_by_name":
			if op.Arguments.Filename == "" {
				t.Fatalf("unable to find filename in arguments")
			}
			_, err := bucket.OpenDownloadStreamByName(op.Arguments.Filename)
			if op.Error {
				require.Error(t, err, "download_by_name failed to error when expected")
				break Loop
			}
			require.NoError(t, err, "download_by_name errored unexpectedly")
		default:
			t.Fatalf("unrecognized operation name: %v", op.Name)
		}

		if op.Result != nil {
			t.Fatalf("unexpected result in GridFS test: %v", op.Result)
		}
	}
	checkExpectations(t, test.Expectations)
}

func checkExpectations(t *testing.T, expectations []expectation) {
	for _, expectation := range expectations {
		var evt *event.CommandStartedEvent
		select {
		case evt = <-gridFSStartedChan:
		default:
			require.Fail(t, "Expected command started event", expectation.CommandStartedEvent.CommandName)
		}

		if evt == nil {
			t.Fatalf("nil command started event occured")
		}

		if expectation.CommandStartedEvent.CommandName != "" {
			require.Equal(t, expectation.CommandStartedEvent.CommandName, evt.CommandName)
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

func setupTest(t *testing.T, testFile *gridFSRetryTestFile, test *gridFSRetryTest, db *mongo.Database) (*Bucket, func()) {
	clearCollections(t)

	chunkSize := loadInitialFiles(t, testFile.Data.Files, testFile.Data.Chunks)
	if chunkSize == 0 {
		chunkSize = DefaultChunkSize
	}

	opts := make([]*options.BucketOptions, 0)
	if testFile.BucketName != "" {
		opt := options.GridFSBucket()
		opt.Name = &testFile.BucketName
		opts = append(opts, opt)
	}
	opt := options.GridFSBucket()
	opt.SetChunkSizeBytes(chunkSize)
	opts = append(opts, opt)

	bucket, err := NewBucket(db, opts...)
	require.NoError(t, err, "unable to create bucket")
	err = bucket.SetReadDeadline(deadline)
	require.NoError(t, err, "unable to set ReadDeadline")
	err = bucket.SetWriteDeadline(deadline)
	require.NoError(t, err, "unable to set WriteDeadline")

	return bucket, func() {
		clearCollections(t)
	}
}
