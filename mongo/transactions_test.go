// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"context"

	"strings"

	"bytes"
	"os"
	"path"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/network/command"
	"go.mongodb.org/mongo-driver/x/network/description"
)

const transactionTestsDir = "../data/transactions"

type transTestFile struct {
	DatabaseName   string           `json:"database_name"`
	CollectionName string           `json:"collection_name"`
	Data           json.RawMessage  `json:"data"`
	Tests          []*transTestCase `json:"tests"`
}

type transTestCase struct {
	Description    string                 `json:"description"`
	FailPoint      *failPoint             `json:"failPoint"`
	ClientOptions  map[string]interface{} `json:"clientOptions"`
	SessionOptions map[string]interface{} `json:"sessionOptions"`
	Operations     []*transOperation      `json:"operations"`
	Outcome        *transOutcome          `json:"outcome"`
	Expectations   []*transExpectation    `json:"expectations"`
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
		Errmsg string `json:"errmsg"`
	} `json:"writeConcernError"`
}

type transOperation struct {
	Name              string                 `json:"name"`
	Object            string                 `json:"object"`
	CollectionOptions map[string]interface{} `json:"collectionOptions"`
	Result            json.RawMessage        `json:"result"`
	Arguments         json.RawMessage        `json:"arguments"`
	ArgMap            map[string]interface{}
}

type transOutcome struct {
	Collection struct {
		Data json.RawMessage `json:"data"`
	} `json:"collection"`
}

type transExpectation struct {
	CommandStartedEvent struct {
		CommandName  string          `json:"command_name"`
		DatabaseName string          `json:"database_name"`
		Command      json.RawMessage `json:"command"`
	} `json:"command_started_event"`
}

type transError struct {
	ErrorContains      string   `bson:"errorContains"`
	ErrorCodeName      string   `bson:"errorCodeName"`
	ErrorLabelsContain []string `bson:"errorLabelsContain"`
	ErrorLabelsOmit    []string `bson:"errorLabelsOmit"`
}

var transStartedChan = make(chan *event.CommandStartedEvent, 100)

var transMonitor = &event.CommandMonitor{
	Started: func(ctx context.Context, cse *event.CommandStartedEvent) {
		//fmt.Printf("STARTED: %v\n", cse)
		transStartedChan <- cse
	},
}

// test case for all TransactionSpec tests
func TestTransactionSpec(t *testing.T) {
	for _, file := range testhelpers.FindJSONFilesInDir(t, transactionTestsDir) {
		runTransactionTestFile(t, path.Join(transactionTestsDir, file))
	}
}

func runTransactionTestFile(t *testing.T, filepath string) {
	content, err := ioutil.ReadFile(filepath)
	require.NoError(t, err)

	var testfile transTestFile
	require.NoError(t, json.Unmarshal(content, &testfile))

	dbName := "admin"
	dbAdmin := createTestDatabase(t, &dbName)

	version, err := getServerVersion(dbAdmin)
	require.NoError(t, err)
	if shouldSkipTransactionsTest(t, version) {
		t.Skip()
	}

	for _, test := range testfile.Tests {
		runTransactionsTestCase(t, test, testfile, dbAdmin)
	}

}

func runTransactionsTestCase(t *testing.T, test *transTestCase, testfile transTestFile, dbAdmin *Database) {
	t.Run(test.Description, func(t *testing.T) {

		// kill sessions from previously failed tests
		killSessions(t, dbAdmin.client)

		// configure failpoint if specified
		if test.FailPoint != nil {
			doc := createFailPointDoc(t, test.FailPoint)
			err := dbAdmin.RunCommand(ctx, doc).Err()
			require.NoError(t, err)

			defer func() {
				// disable failpoint if specified
				_ = dbAdmin.RunCommand(ctx, bsonx.Doc{
					{"configureFailPoint", bsonx.String(test.FailPoint.ConfigureFailPoint)},
					{"mode", bsonx.String("off")},
				})
			}()
		}

		client := createTransactionsMonitoredClient(t, transMonitor, test.ClientOptions)
		addClientOptions(client, test.ClientOptions)

		db := client.Database(testfile.DatabaseName)

		collName := sanitizeCollectionName(testfile.DatabaseName, testfile.CollectionName)

		err := db.Drop(ctx)
		require.NoError(t, err)

		err = db.RunCommand(
			context.Background(),
			bsonx.Doc{{"create", bsonx.String(collName)}},
		).Err()
		require.NoError(t, err)

		// insert data if present
		coll := db.Collection(collName)
		docsToInsert := docSliceToInterfaceSlice(docSliceFromRaw(t, testfile.Data))
		if len(docsToInsert) > 0 {
			coll2, err := coll.Clone(options.Collection().SetWriteConcern(writeconcern.New(writeconcern.WMajority())))
			require.NoError(t, err)
			_, err = coll2.InsertMany(context.Background(), docsToInsert)
			require.NoError(t, err)
		}

		var sess0Opts *options.SessionOptions
		var sess1Opts *options.SessionOptions
		if test.SessionOptions != nil {
			if test.SessionOptions["session0"] != nil {
				sess0Opts = getSessionOptions(test.SessionOptions["session0"].(map[string]interface{}))
			} else if test.SessionOptions["session1"] != nil {
				sess1Opts = getSessionOptions(test.SessionOptions["session1"].(map[string]interface{}))
			}
		}

		session0, err := client.StartSession(sess0Opts)
		require.NoError(t, err)
		session1, err := client.StartSession(sess1Opts)
		require.NoError(t, err)

		sess0 := session0.(*sessionImpl)
		sess1 := session1.(*sessionImpl)

		lsid0 := sess0.SessionID
		lsid1 := sess1.SessionID

		defer func() {
			sess0.EndSession(ctx)
			sess1.EndSession(ctx)
		}()

		// Drain the channel so we only capture events for this test.
		for len(transStartedChan) > 0 {
			<-transStartedChan
		}

		for _, op := range test.Operations {
			if op.Name == "count" {
				t.Skip("count has been deprecated")
			}

			// create collection with default read preference Primary (needed to prevent server selection fail)
			coll = db.Collection(collName, options.Collection().SetReadPreference(readpref.Primary()))
			addCollectionOptions(coll, op.CollectionOptions)

			// Arguments aren't marshaled directly into a map because runcommand
			// needs to convert them into BSON docs.  We convert them to a map here
			// for getting the session and for all other collection operations
			op.ArgMap = getArgMap(t, op.Arguments)

			// Get the session if specified in arguments
			var sess *sessionImpl
			if sessStr, ok := op.ArgMap["session"]; ok {
				switch sessStr.(string) {
				case "session0":
					sess = sess0
				case "session1":
					sess = sess1
				}
			}

			// execute the command on given object
			switch op.Object {
			case "session0":
				err = executeSessionOperation(op, sess0)
			case "session1":
				err = executeSessionOperation(op, sess1)
			case "collection":
				err = executeCollectionOperation(t, op, sess, coll)
			case "database":
				err = executeDatabaseOperation(t, op, sess, db)
			}

			// ensure error is what we expect
			verifyError(t, err, op.Result)
		}

		// Needs to be done here (in spite of defer) because some tests
		// require end session to be called before we check expectation
		sess0.EndSession(ctx)
		sess1.EndSession(ctx)

		checkExpectations(t, test.Expectations, lsid0, lsid1)

		if test.Outcome != nil {
			// Verify with primary read pref
			coll2, err := coll.Clone(options.Collection().SetReadPreference(readpref.Primary()))
			require.NoError(t, err)
			verifyCollectionContents(t, coll2, test.Outcome.Collection.Data)
		}

	})
}

func killSessions(t *testing.T, client *Client) {
	s, err := client.topology.SelectServer(ctx, description.WriteSelector())
	require.NoError(t, err)

	vals := make(bsonx.Arr, 0, 0)
	cmd := command.Write{
		DB:      "admin",
		Command: bsonx.Doc{{"killAllSessions", bsonx.Array(vals)}},
	}
	conn, err := s.Connection(ctx)
	require.NoError(t, err)
	defer testhelpers.RequireNoErrorOnClose(t, conn)
	// ignore the error because command kills its own implicit session
	_, _ = cmd.RoundTrip(context.Background(), s.SelectedDescription(), conn)
}

func createTransactionsMonitoredClient(t *testing.T, monitor *event.CommandMonitor, opts map[string]interface{}) *Client {
	clock := &session.ClusterClock{}

	c := &Client{
		topology:       createMonitoredTopology(t, clock, monitor),
		connString:     testutil.ConnString(t),
		readPreference: readpref.Primary(),
		clock:          clock,
		registry:       bson.NewRegistryBuilder().Build(),
	}

	addClientOptions(c, opts)

	subscription, err := c.topology.Subscribe()
	testhelpers.RequireNil(t, err, "error subscribing to topology: %s", err)
	c.topology.SessionPool = session.NewPool(subscription.C)

	return c
}

func createFailPointDoc(t *testing.T, failPoint *failPoint) bsonx.Doc {
	failDoc := bsonx.Doc{{"configureFailPoint", bsonx.String(failPoint.ConfigureFailPoint)}}

	modeBytes, err := failPoint.Mode.MarshalJSON()
	require.NoError(t, err)

	var modeStruct struct {
		Times int32 `json:"times"`
		Skip  int32 `json:"skip"`
	}
	err = json.Unmarshal(modeBytes, &modeStruct)
	if err != nil {
		failDoc = append(failDoc, bsonx.Elem{"mode", bsonx.String("alwaysOn")})
	} else {
		modeDoc := bsonx.Doc{}
		if modeStruct.Times != 0 {
			modeDoc = append(modeDoc, bsonx.Elem{"times", bsonx.Int32(modeStruct.Times)})
		}
		if modeStruct.Skip != 0 {
			modeDoc = append(modeDoc, bsonx.Elem{"skip", bsonx.Int32(modeStruct.Skip)})
		}
		failDoc = append(failDoc, bsonx.Elem{"mode", bsonx.Document(modeDoc)})
	}

	if failPoint.Data != nil {
		dataDoc := bsonx.Doc{}

		if failPoint.Data.FailCommands != nil {
			failCommandElems := make(bsonx.Arr, len(failPoint.Data.FailCommands))
			for i, str := range failPoint.Data.FailCommands {
				failCommandElems[i] = bsonx.String(str)
			}
			dataDoc = append(dataDoc, bsonx.Elem{"failCommands", bsonx.Array(failCommandElems)})
		}

		if failPoint.Data.CloseConnection {
			dataDoc = append(dataDoc, bsonx.Elem{"closeConnection", bsonx.Boolean(failPoint.Data.CloseConnection)})
		}

		if failPoint.Data.ErrorCode != 0 {
			dataDoc = append(dataDoc, bsonx.Elem{"errorCode", bsonx.Int32(failPoint.Data.ErrorCode)})
		}

		if failPoint.Data.WriteConcernError != nil {
			dataDoc = append(dataDoc,
				bsonx.Elem{"writeConcernError", bsonx.Document(bsonx.Doc{
					{"code", bsonx.Int32(failPoint.Data.WriteConcernError.Code)},
					{"errmsg", bsonx.String(failPoint.Data.WriteConcernError.Errmsg)},
				})},
			)
		}

		if failPoint.Data.FailBeforeCommitExceptionCode != 0 {
			dataDoc = append(dataDoc, bsonx.Elem{"failBeforeCommitExceptionCode", bsonx.Int32(failPoint.Data.FailBeforeCommitExceptionCode)})
		}

		failDoc = append(failDoc, bsonx.Elem{"data", bsonx.Document(dataDoc)})
	}

	return failDoc
}

func executeSessionOperation(op *transOperation, sess *sessionImpl) error {
	switch op.Name {
	case "startTransaction":
		// options are only argument
		var transOpts *options.TransactionOptions
		if op.ArgMap["options"] != nil {
			transOpts = getTransactionOptions(op.ArgMap["options"].(map[string]interface{}))
		}
		return sess.StartTransaction(transOpts)
	case "commitTransaction":
		return sess.CommitTransaction(ctx)
	case "abortTransaction":
		return sess.AbortTransaction(ctx)
	}
	return nil
}

func executeCollectionOperation(t *testing.T, op *transOperation, sess *sessionImpl, coll *Collection) error {
	switch op.Name {
	case "countDocuments":
		_, err := executeCountDocuments(sess, coll, op.ArgMap)
		// no results to verify with count
		return err
	case "distinct":
		res, err := executeDistinct(sess, coll, op.ArgMap)
		if !resultHasError(t, op.Result) {
			verifyDistinctResult(t, res, op.Result)
		}
		return err
	case "insertOne":
		res, err := executeInsertOne(sess, coll, op.ArgMap)
		if !resultHasError(t, op.Result) {
			verifyInsertOneResult(t, res, op.Result)
		}
		return err
	case "insertMany":
		res, err := executeInsertMany(sess, coll, op.ArgMap)
		if !resultHasError(t, op.Result) {
			verifyInsertManyResult(t, res, op.Result)
		}
		return err
	case "find":
		res, err := executeFind(sess, coll, op.ArgMap)
		if !resultHasError(t, op.Result) {
			verifyCursorResult(t, res, op.Result)
		}
		return err
	case "findOneAndDelete":
		res := executeFindOneAndDelete(sess, coll, op.ArgMap)
		if !resultHasError(t, op.Result) {
			verifySingleResult(t, res, op.Result)
		}
		return res.err
	case "findOneAndUpdate":
		res := executeFindOneAndUpdate(sess, coll, op.ArgMap)
		if !resultHasError(t, op.Result) {
			verifySingleResult(t, res, op.Result)
		}
		return res.err
	case "findOneAndReplace":
		res := executeFindOneAndReplace(sess, coll, op.ArgMap)
		if !resultHasError(t, op.Result) {
			verifySingleResult(t, res, op.Result)
		}
		return res.err
	case "deleteOne":
		res, err := executeDeleteOne(sess, coll, op.ArgMap)
		if !resultHasError(t, op.Result) {
			verifyDeleteResult(t, res, op.Result)
		}
		return err
	case "deleteMany":
		res, err := executeDeleteMany(sess, coll, op.ArgMap)
		if !resultHasError(t, op.Result) {
			verifyDeleteResult(t, res, op.Result)
		}
		return err
	case "updateOne":
		res, err := executeUpdateOne(sess, coll, op.ArgMap)
		if !resultHasError(t, op.Result) {
			verifyUpdateResult(t, res, op.Result)
		}
		return err
	case "updateMany":
		res, err := executeUpdateMany(sess, coll, op.ArgMap)
		if !resultHasError(t, op.Result) {
			verifyUpdateResult(t, res, op.Result)
		}
		return err
	case "replaceOne":
		res, err := executeReplaceOne(sess, coll, op.ArgMap)
		if !resultHasError(t, op.Result) {
			verifyUpdateResult(t, res, op.Result)
		}
		return err
	case "aggregate":
		res, err := executeAggregate(sess, coll, op.ArgMap)
		if !resultHasError(t, op.Result) {
			verifyCursorResult2(t, res, op.Result)
		}
		return err
	case "bulkWrite":
		// TODO reenable when bulk writes implemented
		t.Skip("Skipping until bulk writes implemented")
	}
	return nil
}

func executeDatabaseOperation(t *testing.T, op *transOperation, sess *sessionImpl, db *Database) error {
	switch op.Name {
	case "runCommand":
		var result bsonx.Doc
		err := executeRunCommand(sess, db, op.ArgMap, op.Arguments).Decode(&result)
		if !resultHasError(t, op.Result) {
			res, err := result.MarshalBSON()
			if err != nil {
				return err
			}
			verifyRunCommandResult(t, res, op.Result)
		}
		return err
	}
	return nil
}

func verifyError(t *testing.T, e error, result json.RawMessage) {
	expected := getErrorFromResult(t, result)
	if expected == nil {
		return
	}

	if cerr, ok := e.(CommandError); ok {
		if expected.ErrorCodeName != "" {
			require.NotNil(t, cerr)
			require.Equal(t, expected.ErrorCodeName, cerr.Name)
		}
		if expected.ErrorContains != "" {
			require.NotNil(t, cerr, "Expected error %v", expected.ErrorContains)
			require.Contains(t, strings.ToLower(cerr.Message), strings.ToLower(expected.ErrorContains))
		}
		if expected.ErrorLabelsContain != nil {
			require.NotNil(t, cerr)
			for _, l := range expected.ErrorLabelsContain {
				require.True(t, cerr.HasErrorLabel(l), "Error missing error label %s", l)
			}
		}
		if expected.ErrorLabelsOmit != nil {
			require.NotNil(t, cerr)
			for _, l := range expected.ErrorLabelsOmit {
				require.False(t, cerr.HasErrorLabel(l))
			}
		}
	} else {
		require.Equal(t, expected.ErrorCodeName, "")
		require.Equal(t, len(expected.ErrorLabelsContain), 0)
		// ErrorLabelsOmit can contain anything, since they are all omitted for e not type CommandError
		// so we do not check that here

		if expected.ErrorContains != "" {
			require.NotNil(t, e, "Expected error %v", expected.ErrorContains)
			require.Contains(t, strings.ToLower(e.Error()), strings.ToLower(expected.ErrorContains))
		}
	}
}

func resultHasError(t *testing.T, result json.RawMessage) bool {
	if result == nil {
		return false
	}
	res := getErrorFromResult(t, result)
	if res == nil {
		return false
	}
	return res.ErrorLabelsOmit != nil ||
		res.ErrorLabelsContain != nil ||
		res.ErrorCodeName != "" ||
		res.ErrorContains != ""
}

func getErrorFromResult(t *testing.T, result json.RawMessage) *transError {
	expectedBytes, err := result.MarshalJSON()
	require.NoError(t, err)

	var expected transError
	err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)
	if err != nil {
		return nil
	}
	return &expected
}

func checkExpectations(t *testing.T, expectations []*transExpectation, id0 bsonx.Doc, id1 bsonx.Doc) {
	for _, expectation := range expectations {
		var evt *event.CommandStartedEvent
		select {
		case evt = <-transStartedChan:
		default:
			require.Fail(t, "Expected command started event", expectation.CommandStartedEvent.CommandName)
		}

		require.Equal(t, expectation.CommandStartedEvent.CommandName, evt.CommandName)
		require.Equal(t, expectation.CommandStartedEvent.DatabaseName, evt.DatabaseName)

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
			require.NoError(t, actualVal.Validate())
			if key == "lsid" {
				if val.StringValue() == "session0" {
					doc, err := bsonx.ReadDoc(actualVal.Document())
					require.NoError(t, err)
					require.True(t, id0.Equal(doc), "Session ID mismatch")
				}
				if val.StringValue() == "session1" {
					doc, err := bsonx.ReadDoc(actualVal.Document())
					require.NoError(t, err)
					require.True(t, id1.Equal(doc), "Session ID mismatch")
				}
			} else if key == "getMore" {
				require.NotNil(t, actualVal, "Expected %v, got nil for key: %s", elem, key)
				expectedCursorID := val.Int64()
				// ignore if equal to 42
				if expectedCursorID != 42 {
					require.Equal(t, expectedCursorID, actualVal.Int64())
				}
			} else if key == "readConcern" {
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
			} else {
				doc, err := bsonx.ReadDoc(actual)
				require.NoError(t, err)
				compareElements(t, elem, doc.LookupElement(key))
			}

		}
	}
}

// convert operation arguments from raw message into map
func getArgMap(t *testing.T, args json.RawMessage) map[string]interface{} {
	if args == nil {
		return nil
	}
	var argmap map[string]interface{}
	err := json.Unmarshal(args, &argmap)
	require.NoError(t, err)
	return argmap
}

func getSessionOptions(opts map[string]interface{}) *options.SessionOptions {
	sessOpts := options.Session()
	for name, opt := range opts {
		switch name {
		case "causalConsistency":
			sessOpts = sessOpts.SetCausalConsistency(opt.(bool))
		case "defaultTransactionOptions":
			transOpts := opt.(map[string]interface{})
			if transOpts["readConcern"] != nil {
				sessOpts = sessOpts.SetDefaultReadConcern(getReadConcern(transOpts["readConcern"]))
			}
			if transOpts["writeConcern"] != nil {
				sessOpts = sessOpts.SetDefaultWriteConcern(getWriteConcern(transOpts["writeConcern"]))
			}
			if transOpts["readPreference"] != nil {
				sessOpts = sessOpts.SetDefaultReadPreference(getReadPref(transOpts["readPreference"]))
			}
		}
	}

	return sessOpts
}

func getTransactionOptions(opts map[string]interface{}) *options.TransactionOptions {
	transOpts := options.Transaction()
	for name, opt := range opts {
		switch name {
		case "writeConcern":
			transOpts = transOpts.SetWriteConcern(getWriteConcern(opt))
		case "readPreference":
			transOpts = transOpts.SetReadPreference(getReadPref(opt))
		case "readConcern":
			transOpts = transOpts.SetReadConcern(getReadConcern(opt))
		}
	}
	return transOpts
}

func getWriteConcern(opt interface{}) *writeconcern.WriteConcern {
	if w, ok := opt.(map[string]interface{}); ok {
		if conv, ok := w["w"].(string); ok && conv == "majority" {
			return writeconcern.New(writeconcern.WMajority())
		} else if conv, ok := w["w"].(float64); ok {
			return writeconcern.New(writeconcern.W(int(conv)))
		}
	}
	return nil
}

func getReadConcern(opt interface{}) *readconcern.ReadConcern {
	return readconcern.New(readconcern.Level(opt.(map[string]interface{})["level"].(string)))
}

func getReadPref(opt interface{}) *readpref.ReadPref {
	if conv, ok := opt.(map[string]interface{}); ok {
		return readPrefFromString(conv["mode"].(string))
	}
	return nil
}

func readPrefFromString(s string) *readpref.ReadPref {
	switch strings.ToLower(s) {
	case "primary":
		return readpref.Primary()
	case "primarypreferred":
		return readpref.PrimaryPreferred()
	case "secondary":
		return readpref.Secondary()
	case "secondarypreferred":
		return readpref.SecondaryPreferred()
	case "nearest":
		return readpref.Nearest()
	}
	return readpref.Primary()
}

// skip if server version less than 4.0 OR not a replica set.
func shouldSkipTransactionsTest(t *testing.T, serverVersion string) bool {
	return compareVersions(t, serverVersion, "4.0") < 0 ||
		os.Getenv("TOPOLOGY") != "replica_set"
}
