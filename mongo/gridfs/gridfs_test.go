package gridfs

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"path"
	"testing"

	"bytes"

	"fmt"

	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo"
)

type testFile struct {
	Data  dataSection `json:"data"`
	Tests []test      `json:"tests"`
}

type dataSection struct {
	Files  []json.RawMessage `json:"files"`
	Chunks []json.RawMessage `json:"chunks"`
}

type test struct {
	Description string         `json:"description"`
	Arrange     arrangeSection `json:"arrange"`
	Act         actSection     `json:"act"`
	Assert      assertSection  `json:"assert"`
}

type arrangeSection struct {
	Data []json.RawMessage `json:"data"`
}

type actSection struct {
	Operation string          `json:"operation"`
	Arguments json.RawMessage `json:"arguments"`
}

type assertSection struct {
	Result json.RawMessage     `json:"result"`
	Error  string              `json:"error"`
	Data   []assertDataSection `json:"data"`
}

type assertDataSection struct {
	Insert    string            `json:"insert"`
	Documents []interface{}     `json:"documents"`
	Delete    string            `json:"delete"`
	Deletes   []json.RawMessage `json:"deletes"`
}

const gridFsTestDir = "../../data/gridfs"
const downloadBufferSize = 100

var ctx = context.Background()
var emptyDoc = bson.NewDocument()
var client *mongo.Client
var db *mongo.Database
var chunks, files, expectedChunks, expectedFiles *mongo.Collection

var downloadBuffer = make([]byte, downloadBufferSize)
var deadline = time.Now().Add(time.Hour)

// load initial data files into a files and chunks collection
// returns the chunkSize embedded in the documents if there is one
func loadInitialFiles(t *testing.T, data dataSection) int32 {
	filesDocs := make([]interface{}, 0, len(data.Files))
	chunksDocs := make([]interface{}, 0, len(data.Chunks))
	var chunkSize int32

	for _, v := range data.Files {
		docBytes, err := v.MarshalJSON()
		testhelpers.RequireNil(t, err, "error converting raw message to bytes: %s", err)
		doc := bson.NewDocument()
		err = bson.UnmarshalExtJSON(docBytes, false, &doc)
		//fmt.Println(doc.LookupElement("_id"))
		testhelpers.RequireNil(t, err, "error creating file document: %s", err)

		// convert n from int64 to int32
		if cs, err := doc.LookupErr("chunkSize"); err == nil {
			doc.Delete("chunkSize")
			chunkSize = cs.Int32()
			doc.Append(bson.EC.Int32("chunkSize", chunkSize))
		}

		filesDocs = append(filesDocs, doc)
	}

	for _, v := range data.Chunks {
		docBytes, err := v.MarshalJSON()
		testhelpers.RequireNil(t, err, "error converting raw message to bytes: %s", err)
		doc := bson.NewDocument()
		err = bson.UnmarshalExtJSON(docBytes, false, &doc)
		testhelpers.RequireNil(t, err, "error creating file document: %s", err)

		// convert data $hex to binary value
		if hexStr, err := doc.LookupErr("data", "$hex"); err == nil {
			hexBytes := convertHexToBytes(t, hexStr.StringValue())
			doc.Delete("data")
			doc.Append(bson.EC.Binary("data", hexBytes))
		}

		// convert n from int64 to int32
		if n, err := doc.LookupErr("n"); err == nil {
			doc.Delete("n")
			doc.Append(bson.EC.Int32("n", n.Int32()))
		}

		chunksDocs = append(chunksDocs, doc)
	}

	if len(filesDocs) > 0 {
		//fmt.Println(filesDocs)
		_, err := files.InsertMany(ctx, filesDocs)
		testhelpers.RequireNil(t, err, "error inserting into files: %s", err)
		_, err = expectedFiles.InsertMany(ctx, filesDocs)
		testhelpers.RequireNil(t, err, "error inserting into expected files: %s", err)
	}

	if len(chunksDocs) > 0 {
		_, err := chunks.InsertMany(ctx, chunksDocs)
		testhelpers.RequireNil(t, err, "error inserting into chunks: %s", err)
		_, err = expectedChunks.InsertMany(ctx, chunksDocs)
		testhelpers.RequireNil(t, err, "error inserting into expected chunks: %s", err)
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

func TestGridFSSpec(t *testing.T) {
	var err error
	client, err = mongo.NewClientFromConnString(testutil.ConnString(t))
	testhelpers.RequireNil(t, err, "error creating client: %s", err)

	err = client.Connect(ctx)
	testhelpers.RequireNil(t, err, "error connecting client: %s", err)

	db = client.Database("gridFSTestDB")
	chunks = db.Collection("fs.chunks")
	files = db.Collection("fs.files")
	expectedChunks = db.Collection("expected.chunks")
	expectedFiles = db.Collection("expected.files")

	for _, file := range testhelpers.FindJSONFilesInDir(t, gridFsTestDir) {
		runGridFSTestFile(t, path.Join(gridFsTestDir, file), db)
	}
}

func runGridFSTestFile(t *testing.T, filepath string, db *mongo.Database) {
	content, err := ioutil.ReadFile(filepath)
	testhelpers.RequireNil(t, err, "error reading file %s: %s", filepath, err)

	var testfile testFile
	err = json.Unmarshal(content, &testfile)
	testhelpers.RequireNil(t, err, "error unmarshalling test file for %s: %s", filepath, err)

	clearCollections(t)
	chunkSize := loadInitialFiles(t, testfile.Data)
	if chunkSize == 0 {
		chunkSize = DefaultChunkSize
	}

	bucket, err := NewBucket(db, ChunkSizeBytes(chunkSize))
	testhelpers.RequireNil(t, err, "error creating bucket: %s", err)
	err = bucket.SetWriteDeadline(deadline)
	testhelpers.RequireNil(t, err, "error setting write deadline: %s", err)
	err = bucket.SetReadDeadline(deadline)
	testhelpers.RequireNil(t, err, "error setting read deadline: %s", err)

	for _, test := range testfile.Tests {
		t.Run(test.Description, func(t *testing.T) {
			switch test.Act.Operation {
			case "upload":
				runUploadTest(t, test, bucket)
				clearCollections(t)
				runUploadFromStreamTest(t, test, bucket)
			case "download":
				runDownloadTest(t, test, bucket)
				//runDownloadToStreamTest(t, test, bucket)
			case "download_by_name":
				runDownloadByNameTest(t, test, bucket)
				runDownloadByNameToStreamTest(t, test, bucket)
			case "delete":
				runDeleteTest(t, test, bucket)
			}
		})

		if test.Arrange.Data != nil {
			clearCollections(t)
			loadInitialFiles(t, testfile.Data)
		}
	}
}

func getInt64(val *bson.Value) int64 {
	switch val.Type() {
	case bson.TypeInt32:
		return int64(val.Int32())
	case bson.TypeInt64:
		return val.Int64()
	case bson.TypeDouble:
		return int64(val.Double())
	}

	return 0
}

func compareValues(expected *bson.Value, actual *bson.Value) bool {
	if expected.IsNumber() {
		if !actual.IsNumber() {
			return false
		}

		return getInt64(expected) == getInt64(actual)
	}

	switch expected.Type() {
	case bson.TypeString:
		return actual.StringValue() == expected.StringValue()
	case bson.TypeBinary:
		aSub, aBytes := actual.Binary()
		eSub, eBytes := expected.Binary()

		return aSub == eSub && bytes.Equal(aBytes, eBytes)
	case bson.TypeObjectID:
		eID := [12]byte(expected.ObjectID())
		aID := [12]byte(actual.ObjectID())

		return bytes.Equal(eID[:], aID[:])
	case bson.TypeEmbeddedDocument:
		return expected.MutableDocument().Equal(actual.MutableDocument())
	default:
		fmt.Printf("unknown type: %d\n", expected.Type())
	}

	return true // shouldn't get here
}

func compareGfsDoc(t *testing.T, expected *bson.Document, actual *bson.Document, filesID objectid.ObjectID) {
	iter := expected.Iterator()
	for iter.Next() {
		elem := iter.Element()
		key := elem.Key()

		// continue for deprecated fields
		if key == "md5" || key == "contentType" || key == "aliases" {
			continue
		}

		actualVal, err := actual.LookupErr(key)
		testhelpers.RequireNil(t, err, "key %s not found in actual for test %s", key, t.Name())

		// continue for fields with unknown values
		if key == "_id" || key == "uploadDate" {
			continue
		}

		if key == "files_id" {
			expectedBytes := make([]byte, 12)
			actualBytes := make([]byte, 12)

			err = (&filesID).UnmarshalJSON(expectedBytes)
			testhelpers.RequireNil(t, err, "error unmarshalling expected bytes: %s", err)
			actualID := actualVal.ObjectID()
			err = (&actualID).UnmarshalJSON(actualBytes)
			testhelpers.RequireNil(t, err, "error unmarshalling actual bytes: %s", err)

			if !bytes.Equal(expectedBytes, actualBytes) {
				t.Fatalf("files_id mismatch for test %s", t.Name())
			}

			continue
		}

		if eDoc, ok := elem.Value().MutableDocumentOK(); ok {
			compareGfsDoc(t, eDoc, actualVal.MutableDocument(), filesID)
			continue
		}

		if !compareValues(elem.Value(), actualVal) {
			t.Fatalf("values for key %s not equal for test %s", key, t.Name())
		}
	}
}

// compare chunks and expectedChunks collections
func compareChunks(t *testing.T, filesID objectid.ObjectID) {
	actualCursor, err := chunks.Find(ctx, emptyDoc)
	testhelpers.RequireNil(t, err, "error running Find for chunks: %s", err)
	expectedCursor, err := expectedChunks.Find(ctx, emptyDoc)
	testhelpers.RequireNil(t, err, "error running Find for expected chunks: %s", err)

	for expectedCursor.Next(ctx) {
		if !actualCursor.Next(ctx) {
			t.Fatalf("chunks has fewer documents than expectedChunks")
		}

		var actualChunk *bson.Document
		var expectedChunk *bson.Document

		err = actualCursor.Decode(&actualChunk)
		testhelpers.RequireNil(t, err, "error decoding actual chunk: %s", err)
		err = expectedCursor.Decode(&expectedChunk)
		testhelpers.RequireNil(t, err, "error decoding expected chunk: %s", err)

		compareGfsDoc(t, expectedChunk, actualChunk, filesID)
	}
}

// compare files and expectedFiles collections
func compareFiles(t *testing.T) {
	actualCursor, err := files.Find(ctx, emptyDoc)
	testhelpers.RequireNil(t, err, "error running Find for files: %s", err)
	expectedCursor, err := expectedFiles.Find(ctx, emptyDoc)
	testhelpers.RequireNil(t, err, "error running Find for expected files: %s", err)

	for expectedCursor.Next(ctx) {
		if !actualCursor.Next(ctx) {
			t.Fatalf("files has fewer documents than expectedFiles")
		}

		var actualFile *bson.Document
		var expectedFile *bson.Document

		err = actualCursor.Decode(&actualFile)
		testhelpers.RequireNil(t, err, "error decoding actual file: %s", err)
		err = expectedCursor.Decode(&expectedFile)
		testhelpers.RequireNil(t, err, "error decoding expected file: %s", err)

		compareGfsDoc(t, expectedFile, actualFile, objectid.ObjectID{})
	}
}

func convertHexToBytes(t *testing.T, hexStr string) []byte {
	hexBytes, err := hex.DecodeString(hexStr)
	testhelpers.RequireNil(t, err, "error decoding hex for %s: %s", t.Name(), err)
	return hexBytes
}

func msgToDoc(t *testing.T, msg json.RawMessage) *bson.Document {
	rawBytes, err := msg.MarshalJSON()
	testhelpers.RequireNil(t, err, "error marshalling message: %s", err)

	doc := bson.NewDocument()
	err = bson.UnmarshalExtJSON(rawBytes, true, &doc)
	testhelpers.RequireNil(t, err, "error creating BSON doc: %s", err)

	return doc
}

func runUploadAssert(t *testing.T, test test, fileID objectid.ObjectID) {
	assert := test.Assert

	for _, assertData := range assert.Data {
		// each assertData section is a single command that modifies an expected collection
		if assertData.Insert != "" {
			var err error
			docs := make([]interface{}, len(assertData.Documents))

			for i, docInterface := range assertData.Documents {
				rdr, err := bson.Marshal(docInterface)
				testhelpers.RequireNil(t, err, "error marshaling doc: %s", err)
				doc, err := bson.ReadDocument(rdr)
				testhelpers.RequireNil(t, err, "error reading doc: %s", err)

				if id, err := doc.LookupErr("_id"); err == nil {
					idStr := id.StringValue()
					if idStr == "*result" || idStr == "*actual" {
						// server will create _id
						doc.Delete("_id")
					}
				}

				if data, err := doc.LookupErr("data"); err == nil {
					hexBytes := convertHexToBytes(t, data.MutableDocument().Lookup("$hex").StringValue())
					doc.Delete("data")
					doc.Append(bson.EC.Binary("data", hexBytes))
				}

				docs[i] = doc
			}

			switch assertData.Insert {
			case "expected.files":
				_, err = expectedFiles.InsertMany(ctx, docs)
			case "expected.chunks":
				_, err = expectedChunks.InsertMany(ctx, docs)
			}

			testhelpers.RequireNil(t, err, "error modifying expected collections: %s", err)
		}

		compareFiles(t)
		compareChunks(t, fileID)
	}
}

func parseUploadOptions(args *bson.Document) []UploadOptioner {
	var opts []UploadOptioner

	if optionsVal, err := args.LookupErr("options"); err == nil {
		options := optionsVal.MutableDocument()
		iter := options.Iterator()

		for iter.Next() {
			elem := iter.Element()
			val := elem.Value()

			switch elem.Key() {
			case "chunkSizeBytes":
				size := val.Int32()
				opts = append(opts, ChunkSizeBytes(size))
			case "metadata":
				opts = append(opts, Metadata(val.MutableDocument()))
			}
		}
	}

	return opts
}

func runUploadFromStreamTest(t *testing.T, test test, bucket *Bucket) {
	args := msgToDoc(t, test.Act.Arguments)
	opts := parseUploadOptions(args)
	hexBytes := convertHexToBytes(t, args.Lookup("source", "$hex").StringValue())

	fileID, err := bucket.UploadFromStream(args.Lookup("filename").StringValue(), bytes.NewBuffer(hexBytes), opts...)
	testhelpers.RequireNil(t, err, "error uploading from stream: %s", err)

	runUploadAssert(t, test, fileID)
}

func runUploadTest(t *testing.T, test test, bucket *Bucket) {
	// run operation from act section
	args := msgToDoc(t, test.Act.Arguments)

	opts := parseUploadOptions(args)
	hexBytes := convertHexToBytes(t, args.Lookup("source", "$hex").StringValue())
	stream, err := bucket.OpenUploadStream(args.Lookup("filename").StringValue(), opts...)
	testhelpers.RequireNil(t, err, "error opening upload stream for %s: %s", t.Name(), err)

	err = stream.SetWriteDeadline(deadline)
	testhelpers.RequireNil(t, err, "error setting write deadline: %s", err)
	n, err := stream.Write(hexBytes)
	if n != len(hexBytes) {
		t.Fatalf("all bytes not written for %s. expected %d got %d", t.Name(), len(hexBytes), n)
	}

	err = stream.Close()
	testhelpers.RequireNil(t, err, "error closing upload stream for %s: %s", t.Name(), err)

	// assert section is laid out as a series of commands that modify expected.files and expected.chunks
	runUploadAssert(t, test, stream.FileID)
}

// run a series of delete operations that are already BSON documents
func runDeletes(t *testing.T, deletes *bson.Array, coll *mongo.Collection) {
	iter, err := deletes.Iterator()
	testhelpers.RequireNil(t, err, "error creating iterator for deletes: %s", err)

	for iter.Next() {
		doc := iter.Value().MutableDocument() // has q and limit
		filter := doc.Lookup("q").MutableDocument()

		_, err := coll.DeleteOne(ctx, filter)
		testhelpers.RequireNil(t, err, "error running deleteOne for %s: %s", t.Name(), err)
	}
}

// run a series of updates that are already BSON documents
func runUpdates(t *testing.T, updates *bson.Array, coll *mongo.Collection) {
	iter, err := updates.Iterator()
	testhelpers.RequireNil(t, err, "error creating iterator for updates: %s", err)

	for iter.Next() {
		updateDoc := iter.Value().MutableDocument()
		filter := updateDoc.Lookup("q").MutableDocument()
		update := updateDoc.Lookup("u").MutableDocument()

		// update has $set -> data -> $hex
		if hexStr, err := update.LookupErr("$set", "data", "$hex"); err == nil {
			hexBytes := convertHexToBytes(t, hexStr.StringValue())
			update.Delete("$set")
			err = update.Concat(bson.NewDocument(
				bson.EC.SubDocument("$set", bson.NewDocument(
					bson.EC.Binary("data", hexBytes),
				)),
			))
			testhelpers.RequireNil(t, err, "error concatenating data bytes to update: %s", err)
		}

		_, err = coll.UpdateOne(ctx, filter, update)
		testhelpers.RequireNil(t, err, "error running updateOne for test %s: %s", t.Name(), err)
	}
}

func compareDownloadAssertResult(t *testing.T, assert assertSection, copied int64) {
	assertResult, err := assert.Result.MarshalJSON() // json.RawMessage
	testhelpers.RequireNil(t, err, "error marshalling assert result: %s", err)
	assertDoc := bson.NewDocument()
	err = bson.UnmarshalExtJSON(assertResult, true, &assertDoc)
	testhelpers.RequireNil(t, err, "error constructing result doc: %s", err)

	if hexStr, err := assertDoc.LookupErr("result", "$hex"); err == nil {
		hexBytes := convertHexToBytes(t, hexStr.StringValue())

		if copied != int64(len(hexBytes)) {
			t.Fatalf("bytes missing. expected %d bytes, got %d", len(hexBytes), copied)
		}

		if !bytes.Equal(hexBytes, downloadBuffer[:copied]) {
			t.Fatalf("downloaded bytes mismatch. expected %v, got %v", hexBytes, downloadBuffer[:copied])
		}
	}
}

func compareDownloadAssert(t *testing.T, assert assertSection, stream *DownloadStream, streamErr error) {
	var copied int
	var copiedErr error

	if streamErr == nil {
		// files are small enough to read into memory once
		err := stream.SetReadDeadline(deadline)
		testhelpers.RequireNil(t, err, "error setting read deadline: %s", err)
		copied, copiedErr = stream.Read(downloadBuffer)
		testhelpers.RequireNil(t, err, "error reading from stream: %s", err)
	}

	// assert section
	if assert.Result != nil {
		testhelpers.RequireNil(t, streamErr, "error downloading to stream: %s", streamErr)
		compareDownloadAssertResult(t, assert, int64(copied))
	} else if assert.Error != "" {
		var errToCompare error
		var expectedErr error

		switch assert.Error {
		case "FileNotFound":
			fallthrough
		case "RevisionNotFound":
			errToCompare = streamErr
			expectedErr = ErrFileNotFound
		case "ChunkIsMissing":
			errToCompare = copiedErr
			expectedErr = ErrWrongIndex
		case "ChunkIsWrongSize":
			errToCompare = copiedErr
			expectedErr = ErrWrongSize
		}

		testhelpers.RequireNotNil(t, errToCompare, "errToCompare is nil")
		if errToCompare != expectedErr {
			t.Fatalf("err mismatch. expected %s got %s", expectedErr, errToCompare)
		}
	}
}

func compareDownloadToStreamAssert(t *testing.T, assert assertSection, n int64, err error) {
	if assert.Result != nil {
		testhelpers.RequireNil(t, err, "error downloading to stream: %s", err)
		compareDownloadAssertResult(t, assert, n)
	} else if assert.Error != "" {
		var compareErr error

		switch assert.Error {
		case "FileNotFound":
			fallthrough
		case "RevisionNotFound":
			compareErr = ErrFileNotFound
		case "ChunkIsMissing":
			compareErr = ErrWrongIndex
		case "ChunkIsWrongSize":
			compareErr = ErrWrongSize
		}

		testhelpers.RequireNotNil(t, err, "no error when downloading to stream. expected %s", compareErr)
		if err != compareErr {
			t.Fatalf("download to stream error mismatch. expected %s got %s", compareErr, err)
		}
	}
}

func runArrangeSection(t *testing.T, test test, coll *mongo.Collection) {
	for _, msg := range test.Arrange.Data {
		msgBytes, err := msg.MarshalJSON()
		testhelpers.RequireNil(t, err, "error marshalling arrange data for test %s: %s", t.Name(), err)

		msgDoc := bson.NewDocument()
		err = bson.UnmarshalExtJSON(msgBytes, true, &msgDoc)
		testhelpers.RequireNil(t, err, "error creating arrange data doc for test %s: %s", t.Name(), err)

		if _, err = msgDoc.LookupErr("delete"); err == nil {
			// all arrange sections in the current spec tests operate on the fs.chunks collection
			runDeletes(t, msgDoc.Lookup("deletes").MutableArray(), coll)
		} else if _, err = msgDoc.LookupErr("update"); err == nil {
			runUpdates(t, msgDoc.Lookup("updates").MutableArray(), coll)
		}
	}
}

func runDownloadTest(t *testing.T, test test, bucket *Bucket) {
	runArrangeSection(t, test, chunks)

	args := msgToDoc(t, test.Act.Arguments)
	stream, streamErr := bucket.OpenDownloadStream(args.Lookup("id").ObjectID())
	compareDownloadAssert(t, test.Assert, stream, streamErr)
}

func runDownloadToStreamTest(t *testing.T, test test, bucket *Bucket) {
	runArrangeSection(t, test, chunks)
	args := msgToDoc(t, test.Act.Arguments)

	downloadStream := bytes.NewBuffer(downloadBuffer)
	n, err := bucket.DownloadToStream(args.Lookup("id").ObjectID(), downloadStream)

	compareDownloadToStreamAssert(t, test.Assert, n, err)
}

func parseDownloadByNameOpts(t *testing.T, args *bson.Document) []NameOptioner {
	opts := []NameOptioner{}

	if optsVal, err := args.LookupErr("options"); err == nil {
		optsDoc := optsVal.MutableDocument()

		if revVal, err := optsDoc.LookupErr("revision"); err == nil {
			opts = append(opts, Revision(revVal.Int32()))
		}
	}

	return opts
}

func runDownloadByNameTest(t *testing.T, test test, bucket *Bucket) {
	// act section
	args := msgToDoc(t, test.Act.Arguments)
	opts := parseDownloadByNameOpts(t, args)
	stream, streamErr := bucket.OpenDownloadStreamByName(args.Lookup("filename").StringValue(), opts...)
	compareDownloadAssert(t, test.Assert, stream, streamErr)
}

func runDownloadByNameToStreamTest(t *testing.T, test test, bucket *Bucket) {
	args := msgToDoc(t, test.Act.Arguments)
	opts := parseDownloadByNameOpts(t, args)
	downloadStream := bytes.NewBuffer(downloadBuffer)
	n, err := bucket.DownloadToStreamByName(args.Lookup("filename").StringValue(), downloadStream, opts...)

	compareDownloadToStreamAssert(t, test.Assert, n, err)
}

func runDeleteTest(t *testing.T, test test, bucket *Bucket) {
	runArrangeSection(t, test, files)
	args := msgToDoc(t, test.Act.Arguments)

	err := bucket.Delete(args.Lookup("id").ObjectID())
	if test.Assert.Error != "" {
		var errToCompare error
		switch test.Assert.Error {
		case "FileNotFound":
			errToCompare = ErrFileNotFound
		}

		if err != errToCompare {
			t.Fatalf("error mismatch for delete. expected %s got %s", errToCompare, err)
		}
	}

	if len(test.Assert.Data) != 0 {
		for _, data := range test.Assert.Data {
			deletes := bson.NewArray()

			for _, deleteMsg := range data.Deletes {
				deletes.Append(bson.VC.Document(
					msgToDoc(t, deleteMsg),
				))
			}

			runDeletes(t, deletes, expectedFiles)
			compareFiles(t)
		}
	}
}
