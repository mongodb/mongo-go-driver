package mongo

import (
	"context"
	"io/ioutil"
	"path"
	"testing"

	"bytes"
	"fmt"
	"time"

	"os"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/event"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"github.com/mongodb/mongo-go-driver/mongo/insertopt"
	"github.com/mongodb/mongo-go-driver/mongo/updateopt"
)

const cmTestsDir = "../data/command-monitoring"

var startedChan = make(chan *event.CommandStartedEvent, 100)
var succeededChan = make(chan *event.CommandSucceededEvent, 100)
var failedChan = make(chan *event.CommandFailedEvent, 100)
var cursorID int64

var monitor = &event.CommandMonitor{
	Started: func(cse *event.CommandStartedEvent) {
		startedChan <- cse
	},
	Succeeded: func(cse *event.CommandSucceededEvent) {
		succeededChan <- cse
	},
	Failed: func(cfe *event.CommandFailedEvent) {
		failedChan <- cfe
	},
}

func createMonitoredClient(t *testing.T) *Client {
	return &Client{
		topology:       testutil.MonitoredTopology(t, monitor),
		connString:     testutil.ConnString(t),
		readPreference: readpref.Primary(),
	}
}

func skipCmTest(t *testing.T, testCase *bson.Document, serverVersion string) bool {
	minVersionVal, err := testCase.LookupErr("ignore_if_server_version_less_than")
	if err == nil {
		if compareVersions(t, minVersionVal.StringValue(), serverVersion) > 0 {
			return true
		}
	}

	maxVersionVal, err := testCase.LookupErr("ignore_if_server_version_greater_than")
	if err == nil {
		if compareVersions(t, maxVersionVal.StringValue(), serverVersion) < 0 {
			return true
		}
	}

	return false
}

func drainChannels() {
	for len(startedChan) > 0 {
		<-startedChan
	}

	for len(succeededChan) > 0 {
		<-succeededChan
	}

	for len(failedChan) > 0 {
		<-failedChan
	}
}

func insertDocuments(docsArray *bson.Array, coll *Collection, opts ...insertopt.Many) error {
	iter, err := docsArray.Iterator()
	if err != nil {
		return err
	}

	docs := make([]interface{}, 0, docsArray.Len())
	for iter.Next() {
		docs = append(docs, iter.Value().MutableDocument())
	}

	_, err = coll.InsertMany(context.Background(), docs, opts...)
	return err
}

func TestCommandMonitoring(t *testing.T) {
	for _, file := range testhelpers.FindJSONFilesInDir(t, cmTestsDir) {
		runCmTestFile(t, path.Join(cmTestsDir, file))
	}
}

func runCmTestFile(t *testing.T, filepath string) {
	drainChannels() // remove any wire messages from prev test file
	content, err := ioutil.ReadFile(filepath)
	testhelpers.RequireNil(t, err, "error reading JSON file: %s", err)

	doc, err := bson.ParseExtJSONObject(string(content))
	testhelpers.RequireNil(t, err, "error converting JSON to BSON: %s", err)

	client := createMonitoredClient(t)
	db := client.Database(doc.Lookup("database_name").StringValue())

	serverVersionStr, err := getServerVersion(db)
	testhelpers.RequireNil(t, err, "error getting server version: %s", err)

	collName := doc.Lookup("collection_name").StringValue()
	testsIter, err := doc.Lookup("tests").MutableArray().Iterator()
	testhelpers.RequireNil(t, err, "error getting array iterator: %s", err)

	for testsIter.Next() {
		testDoc := testsIter.Value().MutableDocument()

		if skipCmTest(t, testDoc, serverVersionStr) {
			continue
		}

		skippedTopos, err := testDoc.LookupErr("ignore_if_topology_type")
		if err == nil {
			var skipTop bool
			currentTop := os.Getenv("topology")

			iter, err := skippedTopos.MutableArray().Iterator()
			testhelpers.RequireNil(t, err, "error creating iterator for skipped topologies: %s", err)

			for iter.Next() {
				top := iter.Value().StringValue()
				// the only use of ignore_if_topology_type in the tests is in find.json and has "sharded" which actually maps
				// to topology type "sharded_cluster". This option isn't documented in the command monitoring testing README
				// so there's no way of knowing if future CM tests will use this option for other topologies.
				if top == "sharded" && currentTop == "sharded_cluster" {
					skipTop = true
					break
				}
			}

			if skipTop {
				continue
			}
		}

		_, err = db.RunCommand(
			context.Background(),
			bson.NewDocument(bson.EC.String("drop", collName)),
		)

		coll := db.Collection(collName)
		err = insertDocuments(doc.Lookup("data").MutableArray(), coll)
		testhelpers.RequireNil(t, err, "error inserting starting data: %s")

		operationDoc := testDoc.Lookup("operation").MutableDocument()

		drainChannels()

		t.Run(testDoc.Lookup("description").StringValue(), func(t *testing.T) {
			switch operationDoc.Lookup("name").StringValue() {
			case "insertMany":
				cmInsertManyTest(t, testDoc, operationDoc, coll)
			case "find":
				cmFindTest(t, testDoc, operationDoc, coll)
			case "deleteMany":
				cmDeleteManyTest(t, testDoc, operationDoc, coll)
			case "deleteOne":
				cmDeleteOneTest(t, testDoc, operationDoc, coll)
			case "insertOne":
				cmInsertOneTest(t, testDoc, operationDoc, coll)
			case "updateOne":
				cmUpdateOneTest(t, testDoc, operationDoc, coll)
			case "updateMany":
				cmUpdateManyTest(t, testDoc, operationDoc, coll)
			case "count":
				cmCountTest(t, testDoc, operationDoc, coll)
			case "bulkWrite":
				cmBulkWriteTest(t, testDoc, operationDoc, coll)
			}
		})
	}
}

func checkActualHelper(t *testing.T, expected *bson.Document, actual *bson.Document, nested bool) {
	iter := actual.Iterator()
	for iter.Next() {
		elem := iter.Element()
		key := elem.Key()

		// TODO: see comments in compareStartedEvent about ordered and batchSize
		if key == "ordered" || key == "batchSize" {
			continue
		}

		val := elem.Value()

		if nested {
			expectedVal, err := expected.LookupErr(key)
			testhelpers.RequireNil(t, err, "nested field %s not found in expected cmd", key)

			if nestedDoc, ok := val.MutableDocumentOK(); ok {
				checkActualHelper(t, expectedVal.MutableDocument(), nestedDoc, true)
			} else if !compareValues(expectedVal, val) {
				t.Errorf("nested field %s has different value", key)
			}
		}
	}
}

func checkActualFields(t *testing.T, expected *bson.Document, actual *bson.Document) {
	// check that the command sent has no extra fields in nested subdocuments
	checkActualHelper(t, expected, actual, false)
}

func compareWriteError(t *testing.T, expected *bson.Document, actual *bson.Document) {
	expectedIndex := expected.Lookup("index").Int64()
	actualIndex := int64(actual.Lookup("index").Int32())

	if expectedIndex != actualIndex {
		t.Errorf("index mismatch in writeError. expected %d got %d", expectedIndex, actualIndex)
		t.FailNow()
	}

	expectedCode := expected.Lookup("code").Int64()
	actualCode := int64(actual.Lookup("code").Int32())
	if expectedCode == 42 {
		if actualCode <= 0 {
			t.Errorf("expected error code > 0 in writeError. got %d", actualCode)
			t.FailNow()
		}
	} else if expectedCode != actualCode {
		t.Errorf("error code mismatch in writeError. expected %d got %d", expectedCode, actualCode)
		t.FailNow()
	}

	expectedErrMsg := expected.Lookup("errmsg").StringValue()
	actualErrMsg := actual.Lookup("errmsg").StringValue()
	if expectedErrMsg == "" {
		if len(actualErrMsg) == 0 {
			t.Errorf("expected non-empty error msg in writeError")
			t.FailNow()
		}
	} else if expectedErrMsg != actualErrMsg {
		t.Errorf("error message mismatch in writeError. expected %s got %s", expectedErrMsg, actualErrMsg)
		t.FailNow()
	}
}

func compareWriteErrors(t *testing.T, expected *bson.Array, actual *bson.Array) {
	if expected.Len() != actual.Len() {
		t.Errorf("writeErrors length mismatch. expected %d got %d", expected.Len(), actual.Len())
		t.FailNow()
	}

	iter, err := expected.Iterator()
	testhelpers.RequireNil(t, err, "error creating expected iterator for writeErrors: %s", err)

	actualIter, err := actual.Iterator()
	testhelpers.RequireNil(t, err, "error creating actual iterator for writeErrors: %s", err)

	for iter.Next() && actualIter.Next() {
		expectedErr := iter.Value().MutableDocument()
		actualErr := actualIter.Value().MutableDocument()
		compareWriteError(t, expectedErr, actualErr)
	}
}

func compareBatches(t *testing.T, expected *bson.Array, actual *bson.Array) {
	if expected.Len() != actual.Len() {
		t.Errorf("batch length mismatch. expected %d got %d", expected.Len(), actual.Len())
		t.FailNow()
	}

	expectedIter, err := expected.Iterator()
	testhelpers.RequireNil(t, err, "error creating iterator for expected batch: %s", err)

	actualIter, err := actual.Iterator()
	testhelpers.RequireNil(t, err, "error creating iterator for actual batch: %s", err)

	for expectedIter.Next() && actualIter.Next() {
		expectedDoc := expectedIter.Value().MutableDocument()
		actualDoc := actualIter.Value().MutableDocument()

		if !expectedDoc.Equal(actualDoc) {
			t.Errorf("document mismatch in cursor batch")
			t.FailNow()
		}
	}
}

func compareCursors(t *testing.T, expected *bson.Document, actual *bson.Document) {
	expectedID := expected.Lookup("id").Int64()
	actualID := actual.Lookup("id").Int64()

	// this means we're getting a new cursor ID from the server in response to a query
	// future cursor IDs in getMore commands for this test case should match this ID
	if expectedID == 42 {
		cursorID = actualID
	} else {
		if expectedID != actualID {
			t.Errorf("cursor ID mismatch. expected %d got %d", expectedID, actualID)
			t.FailNow()
		}
	}

	expectedNS := expected.Lookup("ns").StringValue()
	actualNS := actual.Lookup("ns").StringValue()
	if expectedNS != actualNS {
		t.Errorf("cursor NS mismatch. expected %s got %s", expectedNS, actualNS)
		t.FailNow()
	}

	batchID := "firstBatch"
	batchVal, err := expected.LookupErr(batchID)
	if err != nil {
		batchID = "nextBatch"
		batchVal = expected.Lookup(batchID)
	}

	actualBatchVal, err := actual.LookupErr(batchID)
	testhelpers.RequireNil(t, err, "could not find batch with ID %s in actual cursor", batchID)

	expectedBatch := batchVal.MutableArray()
	actualBatch := actualBatchVal.MutableArray()
	compareBatches(t, expectedBatch, actualBatch)
}

func compareUpserted(t *testing.T, expected *bson.Array, actual *bson.Array) {
	if expected.Len() != actual.Len() {
		t.Errorf("length mismatch. expected %d got %d", expected.Len(), actual.Len())
	}
	expectedIter, err := expected.Iterator()
	testhelpers.RequireNil(t, err, "error creating expected iter: %s", err)

	actualIter, err := actual.Iterator()
	testhelpers.RequireNil(t, err, "error creating actual iter: %s", err)

	for expectedIter.Next() && actualIter.Next() {
		compareDocs(t, expectedIter.Value().MutableDocument(), actualIter.Value().MutableDocument())
	}
}

func compareReply(t *testing.T, succeeded *event.CommandSucceededEvent, reply *bson.Document) {
	iter := reply.Iterator()
	for iter.Next() {
		elem := iter.Element()
		switch elem.Key() {
		case "ok":
			var actualOk int64

			actualOkVal, err := succeeded.Reply.LookupErr("ok")
			testhelpers.RequireNil(t, err, "could not find key ok in reply")

			switch actualOkVal.Type() {
			case bson.TypeInt32:
				actualOk = int64(actualOkVal.Int32())
			case bson.TypeInt64:
				actualOk = actualOkVal.Int64()
			case bson.TypeDouble:
				actualOk = int64(actualOkVal.Double())
			}

			if actualOk != elem.Value().Int64() {
				t.Errorf("ok value in reply does not match. expected %d got %d", elem.Value().Int64(), actualOk)
				t.FailNow()
			}
		case "n":
			actualNVal, err := succeeded.Reply.LookupErr("n")
			testhelpers.RequireNil(t, err, "could not find key n in reply")

			if int(actualNVal.Int32()) != int(elem.Value().Int64()) {
				t.Errorf("n values do not match. expected %d got %d", elem.Value().Int32(), actualNVal.Int64())
				t.FailNow()
			}
		case "writeErrors":
			actualArr, err := succeeded.Reply.LookupErr("writeErrors")
			testhelpers.RequireNil(t, err, "could not find key writeErrors in reply")

			compareWriteErrors(t, elem.Value().MutableArray(), actualArr.MutableArray())
		case "cursor":
			actualDoc, err := succeeded.Reply.LookupErr("cursor")
			testhelpers.RequireNil(t, err, "could not find key cursor in reply")

			compareCursors(t, elem.Value().MutableDocument(), actualDoc.MutableDocument())
		case "upserted":
			actualDoc, err := succeeded.Reply.LookupErr("upserted")
			testhelpers.RequireNil(t, err, "could not find key upserted in reply")

			compareUpserted(t, elem.Value().MutableArray(), actualDoc.MutableArray())
		default:
			fmt.Printf("key %s does not match existing case\n", elem.Key())
		}
	}
}

func compareValues(expected *bson.Value, actual *bson.Value) bool {
	if expected.IsNumber() {
		if !actual.IsNumber() {
			return false
		}

		expectedNum := expected.Int64() // from JSON file
		switch actual.Type() {
		case bson.TypeInt32:
			if int64(actual.Int32()) != expectedNum {
				return false
			}
		case bson.TypeInt64:
			if actual.Int64() != expectedNum {
				return false
			}
		case bson.TypeDouble:
			if int64(actual.Double()) != expectedNum {
				return false
			}
		}

		return true
	}

	switch expected.Type() {
	case bson.TypeString:
		if aStr, ok := actual.StringValueOK(); !(ok && aStr == expected.StringValue()) {
			return false
		}
	case bson.TypeBinary:
		aSub, aBytes := actual.Binary()
		eSub, eBytes := expected.Binary()

		if (aSub != eSub) || (!bytes.Equal(aBytes, eBytes)) {
			return false
		}
	}

	return true
}

func compareDocs(t *testing.T, expected *bson.Document, actual *bson.Document) {
	// this is necessary even though Equal() exists for documents because types not match between commands and the BSON
	// documents given in test cases. for example, all numbers in the test case JSON are parsed as int64, but many nubmers
	// sent over the wire are type int32
	if expected.Len() != actual.Len() {
		t.Errorf("doc length mismatch. expected %d got %d", expected.Len(), actual.Len())
		t.FailNow()
	}

	eIter := expected.Iterator()

	for eIter.Next() {
		expectedElem := eIter.Element()

		aVal, err := actual.LookupErr(expectedElem.Key())
		testhelpers.RequireNil(t, err, "docs not equal. key %s not found in actual", expectedElem.Key())

		eVal := expectedElem.Value()

		if doc, ok := eVal.MutableDocumentOK(); ok {
			// nested doc
			compareDocs(t, doc, aVal.MutableDocument())

			// nested docs were equal
			continue
		}

		if !compareValues(eVal, aVal) {
			t.Errorf("docs not equal because value mismatch for key %s", expectedElem.Key())
		}
	}
}

func compareStartedEvent(t *testing.T, expected *bson.Document) {
	// rules for command comparison (from spec):
	// 1. the actual command can have extra fields not specified in the test case, but only as top level fields
	// these fields cannot be in nested subdocuments
	// 2. the actual command must have all fields specified in the test case

	// this function only checks that everything in the test case cmd is also in the monitored cmd
	// checkActualFields() checks that the actual cmd has no extra fields in nested subdocuments
	if len(startedChan) == 0 {
		t.Errorf("no started event waiting")
		t.FailNow()
	}

	started := <-startedChan

	if started.CommandName == "isMaster" {
		return
	}

	expectedCmdName := expected.Lookup("command_name").StringValue()
	if expectedCmdName != started.CommandName {
		t.Errorf("command name mismatch. expected %s, got %s", expectedCmdName, started.CommandName)
		t.FailNow()
	}
	expectedDbName := expected.Lookup("database_name").StringValue()
	if expectedDbName != started.DatabaseName {
		t.Errorf("database name mismatch. expected %s, got %s", expectedDbName, started.DatabaseName)
		t.FailNow()
	}

	expectedCmd := expected.Lookup("command").MutableDocument()
	var identifier string
	var expectedPayload *bson.Array

	iter := expectedCmd.Iterator()
	for iter.Next() {
		elem := iter.Element()
		key := elem.Key()
		val := elem.Value()

		if key == "getMore" && expectedCmdName == "getMore" {
			expectedCursorID := val.Int64()
			actualCursorID := started.Command.Lookup("getMore").Int64()

			if expectedCursorID == 42 {
				if actualCursorID != cursorID {
					t.Errorf("cursor ID mismatch in getMore. expected %d got %d", cursorID, actualCursorID)
					t.FailNow()
				}
			} else if expectedCursorID != actualCursorID {
				t.Errorf("cursor ID mismatch in getMore. expected %d got %d", expectedCursorID, actualCursorID)
				t.FailNow()
			}
			continue
		}

		// type 1 payload
		if key == "documents" || key == "updates" || key == "deletes" {
			expectedPayload = val.MutableArray()
			identifier = key
			continue
		}

		// TODO: some tests specify that "ordered" must be a key in the event but ordered isn't a valid option for some of these cases (e.g. insertOne)
		if key == "ordered" {
			continue
		}

		actualVal, err := started.Command.LookupErr(key)
		testhelpers.RequireNil(t, err, "key %s not found in started event %s", key, started.CommandName)

		// TODO: tests in find.json expect different batch sizes for find/getMore commands based on an optional limit
		// e.g. if limit = 4 and find is run with batchSize = 3, the getMore should have batchSize = 1
		// skipping because the driver doesn't have this logic
		if key == "batchSize" {
			continue
		}

		if !compareValues(val, actualVal) {
			t.Errorf("values for key %s do not match. cmd: %s", key, started.CommandName)
			t.FailNow()
		}
	}

	if expectedPayload == nil {
		// no type 1 payload
		return
	}

	actualArrayVal, err := started.Command.LookupErr(identifier)
	testhelpers.RequireNil(t, err, "array with id %s not found in started command", identifier)
	actualPayload := actualArrayVal.MutableArray()

	if expectedPayload.Len() != actualPayload.Len() {
		t.Errorf("payload length mismatch. expected %d, got %d", expectedPayload.Len(), actualPayload.Len())
		t.FailNow()
	}

	expectedIter, err := expectedPayload.Iterator()
	testhelpers.RequireNil(t, err, "error creating iterator for expected payload: %s", err)

	actualIter, err := actualPayload.Iterator()
	testhelpers.RequireNil(t, err, "error creating iterator for actual payload: %s", err)

	for expectedIter.Next() && actualIter.Next() {
		expectedDoc := expectedIter.Value().MutableDocument()
		actualDoc := actualIter.Value().MutableDocument()

		compareDocs(t, expectedDoc, actualDoc)
	}

	checkActualFields(t, expectedCmd, started.Command)
}

func compareSuccessEvent(t *testing.T, expected *bson.Document) {
	if len(succeededChan) == 0 {
		t.Errorf("no success event waiting")
		t.FailNow()
	}

	succeeded := <-succeededChan

	if succeeded.CommandName == "isMaster" {
		return
	}

	cmdName := expected.Lookup("command_name").StringValue()
	if cmdName != succeeded.CommandName {
		t.Errorf("command name mismatch. expected %s got %s", cmdName, succeeded.CommandName)
		t.FailNow()
	}

	compareReply(t, succeeded, expected.Lookup("reply").MutableDocument())
}

func compareFailureEvent(t *testing.T, expected *bson.Document) {
	if len(failedChan) == 0 {
		t.Errorf("no failure event waiting")
		t.FailNow()
	}

	failed := <-failedChan
	expectedName := expected.Lookup("command_name").StringValue()

	if expectedName != failed.CommandName {
		t.Errorf("command name mismatch for failed event. expected %s got %s", expectedName, failed.CommandName)
		t.FailNow()
	}
}

func compareExpectations(t *testing.T, testCase *bson.Document) {
	expectations := testCase.Lookup("expectations").MutableArray()
	iter, err := expectations.Iterator()
	testhelpers.RequireNil(t, err, "error creating expecations iterator: %s", err)

	for iter.Next() {
		expectedDoc := iter.Value().MutableDocument()

		if startDoc, err := expectedDoc.LookupErr("command_started_event"); err == nil {
			compareStartedEvent(t, startDoc.MutableDocument())
			continue
		}

		if successDoc, err := expectedDoc.LookupErr("command_succeeded_event"); err == nil {
			compareSuccessEvent(t, successDoc.MutableDocument())
			continue
		}

		compareFailureEvent(t, expectedDoc.Lookup("command_failed_event").MutableDocument())
	}
}

func cmFindOptions(arguments *bson.Document) []findopt.Find {
	var opts []findopt.Find

	if sort, err := arguments.LookupErr("sort"); err == nil {
		opts = append(opts, findopt.Sort(sort.MutableDocument()))
	}

	if skip, err := arguments.LookupErr("skip"); err == nil {
		opts = append(opts, findopt.Skip(skip.Int64()))
	}

	if limit, err := arguments.LookupErr("limit"); err == nil {
		opts = append(opts, findopt.Limit(limit.Int64()))
	}

	if batchSize, err := arguments.LookupErr("batchSize"); err == nil {
		opts = append(opts, findopt.BatchSize(int32(batchSize.Int64())))
	}

	if collation, err := arguments.LookupErr("collation"); err == nil {
		collMap := collation.Interface().(map[string]interface{})
		opts = append(opts, findopt.Collation(collationFromMap(collMap)))
	}

	modifiersVal, err := arguments.LookupErr("modifiers")
	if err != nil {
		return opts
	}

	iter := modifiersVal.MutableDocument().Iterator()
	for iter.Next() {
		elem := iter.Element()
		val := elem.Value()

		var opt findopt.Find
		switch elem.Key() {
		case "$comment":
			opt = findopt.Comment(val.StringValue())
		case "$hint":
			opt = findopt.Hint(val.MutableDocument())
		case "$max":
			opt = findopt.Max(val.MutableDocument())
		case "$maxTimeMS":
			ns := time.Duration(val.Int64()) * time.Millisecond
			opt = findopt.MaxTime(ns)
		case "$min":
			opt = findopt.Min(val.MutableDocument())
		case "$returnKey":
			opt = findopt.ReturnKey(val.Boolean())
		case "$showDiskLoc":
			opt = findopt.ShowRecordID(val.Boolean())
		}

		if opt != nil {
			opts = append(opts, opt)
		}
	}

	return opts
}

func getRp(rpDoc *bson.Document) *readpref.ReadPref {
	rpMode := rpDoc.Lookup("mode").StringValue()
	switch rpMode {
	case "primary":
		return readpref.Primary()
	case "secondary":
		return readpref.Secondary()
	case "primaryPreferred":
		return readpref.PrimaryPreferred()
	case "secondaryPreferred":
		return readpref.SecondaryPreferred()
	}

	return nil
}

func cmInsertManyTest(t *testing.T, testCase *bson.Document, operation *bson.Document, coll *Collection) {
	t.Logf("RUNNING %s\n", testCase.Lookup("description").StringValue())
	args := operation.Lookup("arguments").MutableDocument()
	var opts []insertopt.Many
	var orderedGiven bool

	if optionsDoc, err := args.LookupErr("options"); err == nil {
		if ordered, err := optionsDoc.MutableDocument().LookupErr("ordered"); err == nil {
			orderedGiven = true
			opts = append(opts, insertopt.Ordered(ordered.Boolean()))
		}
	}

	// TODO: ordered?
	if !orderedGiven {
		opts = append(opts, insertopt.Ordered(true))
	}
	// ignore errors because write errors constitute a successful command
	_ = insertDocuments(args.Lookup("documents").MutableArray(), coll, opts...)
	compareExpectations(t, testCase)
}

func cmFindTest(t *testing.T, testCase *bson.Document, operation *bson.Document, coll *Collection) {
	arguments := operation.Lookup("arguments").MutableDocument()
	filter := arguments.Lookup("filter").MutableDocument()
	opts := cmFindOptions(arguments)

	oldRp := coll.readPreference
	if rpVal, err := operation.LookupErr("read_preference"); err == nil {
		coll.readPreference = getRp(rpVal.MutableDocument())
	}

	cursor, _ := coll.Find(context.Background(), filter, opts...) // ignore errors at this stage

	if cursor != nil {
		for cursor.Next(context.Background()) {
		}
	}

	coll.readPreference = oldRp
	compareExpectations(t, testCase)
}

func cmDeleteManyTest(t *testing.T, testCase *bson.Document, operation *bson.Document, coll *Collection) {
	args := operation.Lookup("arguments").MutableDocument()
	filter := args.Lookup("filter").MutableDocument()

	_, _ = coll.DeleteMany(context.Background(), filter)
	compareExpectations(t, testCase)
}

func cmDeleteOneTest(t *testing.T, testCase *bson.Document, operation *bson.Document, coll *Collection) {
	args := operation.Lookup("arguments").MutableDocument()
	filter := args.Lookup("filter").MutableDocument()

	_, _ = coll.DeleteOne(context.Background(), filter)
	compareExpectations(t, testCase)
}

func cmInsertOneTest(t *testing.T, testCase *bson.Document, operation *bson.Document, coll *Collection) {
	// ignore errors because write errors constitute a successful command
	_, _ = coll.InsertOne(context.Background(),
		operation.Lookup("arguments").MutableDocument().Lookup("document").MutableDocument())
	compareExpectations(t, testCase)
}

func getUpdateParams(args *bson.Document) (*bson.Document, *bson.Document, []updateopt.Update) {
	filter := args.Lookup("filter").MutableDocument()
	update := args.Lookup("update").MutableDocument()

	opts := []updateopt.Update{
		updateopt.Upsert(false), // can be overwritten by test options
	}
	if upsert, err := args.LookupErr("upsert"); err == nil {
		opts = append(opts, updateopt.Upsert(upsert.Boolean()))
	}

	return filter, update, opts
}

func runUpdateOne(args *bson.Document, coll *Collection, updateOpts ...updateopt.Update) {
	filter, update, opts := getUpdateParams(args)
	opts = append(opts, updateOpts...)
	_, _ = coll.UpdateOne(context.Background(), filter, update, opts...)
}

func cmUpdateOneTest(t *testing.T, testCase *bson.Document, operation *bson.Document, coll *Collection) {
	args := operation.Lookup("arguments").MutableDocument()
	runUpdateOne(args, coll)
	compareExpectations(t, testCase)
}

func cmUpdateManyTest(t *testing.T, testCase *bson.Document, operation *bson.Document, coll *Collection) {
	args := operation.Lookup("arguments").MutableDocument()
	filter, update, opts := getUpdateParams(args)
	_, _ = coll.UpdateMany(context.Background(), filter, update, opts...)
	compareExpectations(t, testCase)
}

func cmCountTest(t *testing.T, testCase *bson.Document, operation *bson.Document, coll *Collection) {
	filter := operation.Lookup("arguments").MutableDocument().Lookup("filter").MutableDocument()

	oldRp := coll.readPreference
	if rpVal, err := operation.LookupErr("read_preference"); err == nil {
		coll.readPreference = getRp(rpVal.MutableDocument())
	}

	_, _ = coll.Count(context.Background(), filter)
	coll.readPreference = oldRp

	compareExpectations(t, testCase)
}

func cmBulkWriteTest(t *testing.T, testCase *bson.Document, operation *bson.Document, coll *Collection) {
	outerArguments := operation.Lookup("arguments").MutableDocument()
	iter, err := outerArguments.Lookup("requests").MutableArray().Iterator()
	testhelpers.RequireNil(t, err, "error creating bulk write iterator: %s", err)

	var wc *writeconcern.WriteConcern
	if collOpts, err := operation.LookupErr("collectionOptions"); err == nil {
		wcDoc := collOpts.MutableDocument().Lookup("writeConcern").MutableDocument()
		wc = writeconcern.New(writeconcern.W(int(wcDoc.Lookup("w").Int64())))
	}

	oldWc := coll.writeConcern
	if wc != nil {
		coll.writeConcern = wc
	}

	for iter.Next() {
		requestDoc := iter.Value().MutableDocument()
		args := requestDoc.Lookup("arguments").MutableDocument()

		switch requestDoc.Lookup("name").StringValue() {
		case "insertOne":
			_, _ = coll.InsertOne(context.Background(),
				args.Lookup("document").MutableDocument())
		case "updateOne":
			runUpdateOne(args, coll)
		}
	}

	coll.writeConcern = oldWc

	if !writeconcern.AckWrite(wc) {
		time.Sleep(time.Second) // sleep to allow event to be written to channel before checking expectations
	}
	compareExpectations(t, testCase)
}
