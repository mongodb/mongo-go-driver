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

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/event"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/collectionopt"
	"github.com/mongodb/mongo-go-driver/mongo/sessionopt"
	"github.com/mongodb/mongo-go-driver/mongo/transactionopt"
	"github.com/stretchr/testify/require"
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
			_, err := dbAdmin.RunCommand(ctx, doc)
			require.NoError(t, err)

			defer func() {
				// disable failpoint if specified
				_, _ = dbAdmin.RunCommand(ctx, bson.NewDocument(
					bson.EC.String("configureFailPoint", test.FailPoint.ConfigureFailPoint),
					bson.EC.String("mode", "off"),
				))
			}()
		}

		client := createTransactionsMonitoredClient(t, transMonitor, test.ClientOptions)
		addClientOptions(client, test.ClientOptions)

		db := client.Database(testfile.DatabaseName)

		collName := sanitizeCollectionName(testfile.DatabaseName, testfile.CollectionName)

		err := db.Drop(ctx)
		require.NoError(t, err)

		_, err = db.RunCommand(
			context.Background(),
			bson.NewDocument(
				bson.EC.String("create", collName),
			),
		)
		require.NoError(t, err)

		// insert data if present
		coll := db.Collection(collName)
		docsToInsert := docSliceToInterfaceSlice(docSliceFromRaw(t, testfile.Data))
		if len(docsToInsert) > 0 {
			coll2, err := coll.Clone(collectionopt.WriteConcern(writeconcern.New(writeconcern.WMajority())))
			require.NoError(t, err)
			_, err = coll2.InsertMany(context.Background(), docsToInsert)
			require.NoError(t, err)
		}

		var sess0Opts *sessionopt.SessionBundle
		var sess1Opts *sessionopt.SessionBundle
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
			// create collection with default read preference Primary (needed to prevent server selection fail)
			coll = db.Collection(collName, collectionopt.ReadPreference(readpref.Primary()))
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
			coll2, err := coll.Clone(collectionopt.ReadPreference(readpref.Primary()))
			require.NoError(t, err)
			verifyCollectionContents(t, coll2, test.Outcome.Collection.Data)
		}

	})
}

func killSessions(t *testing.T, client *Client) {
	s, err := client.topology.SelectServer(ctx, description.WriteSelector())
	require.NoError(t, err)

	vals := make([]*bson.Value, 0, 0)
	cmd := command.Write{
		DB: "admin",
		Command: bson.NewDocument(
			bson.EC.ArrayFromElements("killAllSessions", vals...),
		),
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

func createFailPointDoc(t *testing.T, failPoint *failPoint) *bson.Document {
	failDoc := bson.NewDocument(
		bson.EC.String("configureFailPoint", failPoint.ConfigureFailPoint),
	)

	modeBytes, err := failPoint.Mode.MarshalJSON()
	require.NoError(t, err)

	var modeStruct struct {
		Times int32 `json:"times"`
		Skip  int32 `json:"skip"`
	}
	err = json.Unmarshal(modeBytes, &modeStruct)
	if err != nil {
		failDoc.Append(bson.EC.String("mode", "alwaysOn"))
	} else {
		modeDoc := bson.NewDocument()
		if modeStruct.Times != 0 {
			modeDoc.Append(bson.EC.Int32("times", modeStruct.Times))
		}
		if modeStruct.Skip != 0 {
			modeDoc.Append(bson.EC.Int32("skip", modeStruct.Skip))
		}
		failDoc.Append(bson.EC.SubDocument("mode", modeDoc))
	}

	if failPoint.Data != nil {
		dataDoc := bson.NewDocument()

		if failPoint.Data.FailCommands != nil {
			failCommandElems := make([]*bson.Value, len(failPoint.Data.FailCommands))
			for i, str := range failPoint.Data.FailCommands {
				failCommandElems[i] = bson.VC.String(str)
			}
			dataDoc.Append(bson.EC.ArrayFromElements("failCommands", failCommandElems...))
		}

		if failPoint.Data.CloseConnection {
			dataDoc.Append(bson.EC.Boolean("closeConnection", failPoint.Data.CloseConnection))
		}

		if failPoint.Data.ErrorCode != 0 {
			dataDoc.Append(bson.EC.Int32("errorCode", failPoint.Data.ErrorCode))
		}

		if failPoint.Data.WriteConcernError != nil {
			dataDoc.Append(
				bson.EC.SubDocument("writeConcernError", bson.NewDocument(
					bson.EC.Int32("code", failPoint.Data.WriteConcernError.Code),
					bson.EC.String("errmsg", failPoint.Data.WriteConcernError.Errmsg),
				)),
			)
		}

		if failPoint.Data.FailBeforeCommitExceptionCode != 0 {
			dataDoc.Append(
				bson.EC.Int32("failBeforeCommitExceptionCode", failPoint.Data.FailBeforeCommitExceptionCode),
			)
		}

		failDoc.Append(bson.EC.SubDocument("data", dataDoc))
	}

	return failDoc
}

func executeSessionOperation(op *transOperation, sess *sessionImpl) error {
	switch op.Name {
	case "startTransaction":
		// options are only argument
		var transOpts transactionopt.Transaction
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
	case "count":
		_, err := executeCount(sess, coll, op.ArgMap)
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
			verifyDocumentResult(t, res, op.Result)
		}
		return res.err
	case "findOneAndUpdate":
		res := executeFindOneAndUpdate(sess, coll, op.ArgMap)
		if !resultHasError(t, op.Result) {
			verifyDocumentResult(t, res, op.Result)
		}
		return res.err
	case "findOneAndReplace":
		res := executeFindOneAndReplace(sess, coll, op.ArgMap)
		if !resultHasError(t, op.Result) {
			verifyDocumentResult(t, res, op.Result)
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
			verifyCursorResult(t, res, op.Result)
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
		res, err := executeRunCommand(sess, db, op.ArgMap, op.Arguments)
		if !resultHasError(t, op.Result) {
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

	if cerr, ok := e.(command.Error); ok {
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
		// ErrorLabelsOmit can contain anything, since they are all omitted for e not type command.Error
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

func checkExpectations(t *testing.T, expectations []*transExpectation, id0 *bson.Document, id1 *bson.Document) {
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

		expected := bson.NewDocument()
		err = bson.UnmarshalExtJSON(jsonBytes, true, &expected)
		require.NoError(t, err)

		actual := evt.Command
		iter := expected.Iterator()
		for iter.Next() {
			elem := iter.Element()
			key := elem.Key()
			val := elem.Value()

			actualVal := actual.Lookup(key)

			// Keys that may be nil
			if val.Type() == bson.TypeNull {
				require.Nil(t, actual.LookupElement(key), "Expected %s to be nil", key)
				continue
			} else if key == "ordered" {
				// TODO: some tests specify that "ordered" must be a key in the event but ordered isn't a valid option for some of these cases (e.g. insertOne)
				continue
			}

			// Keys that should not be nil
			require.NotNil(t, actualVal, "Expected %v, got nil for key: %s", elem, key)
			if key == "lsid" {
				if val.StringValue() == "session0" {
					require.True(t, id0.Equal(actualVal.MutableDocument()), "Session ID mismatch")
				}
				if val.StringValue() == "session1" {
					require.True(t, id1.Equal(actualVal.MutableDocument()), "Session ID mismatch")
				}
			} else if key == "getMore" {
				require.NotNil(t, actualVal, "Expected %v, got nil for key: %s", elem, key)
				expectedCursorID := val.Int64()
				// ignore if equal to 42
				if expectedCursorID != 42 {
					require.Equal(t, expectedCursorID, actualVal.Int64())
				}
			} else if key == "readConcern" {
				rcExpectDoc := val.MutableDocument()
				rcActualDoc := actualVal.MutableDocument()
				clusterTime := rcExpectDoc.Lookup("afterClusterTime")
				level := rcExpectDoc.Lookup("level")
				if clusterTime != nil {
					require.NotNil(t, rcActualDoc.Lookup("afterClusterTime"))
				}
				if level != nil {
					compareElements(t, rcExpectDoc.LookupElement("level"), rcActualDoc.LookupElement("level"))
				}
			} else {
				compareElements(t, elem, actual.LookupElement(key))
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

func getSessionOptions(opts map[string]interface{}) *sessionopt.SessionBundle {
	sessOpts := sessionopt.BundleSession()
	for name, opt := range opts {
		switch name {
		case "causalConsistency":
			sessOpts = sessOpts.CausalConsistency(opt.(bool))
		case "defaultTransactionOptions":
			transOpts := opt.(map[string]interface{})
			if transOpts["readConcern"] != nil {
				sessOpts = sessOpts.DefaultReadConcern(getReadConcern(transOpts["readConcern"]))
			}
			if transOpts["writeConcern"] != nil {
				sessOpts = sessOpts.DefaultWriteConcern(getWriteConcern(transOpts["writeConcern"]))
			}
			if transOpts["readPreference"] != nil {
				sessOpts = sessOpts.DefaultReadPreference(getReadPref(transOpts["readPreference"]))
			}
		}
	}

	return sessOpts
}

func getTransactionOptions(opts map[string]interface{}) *transactionopt.TransactionBundle {
	transOpts := transactionopt.BundleTransaction()
	for name, opt := range opts {
		switch name {
		case "writeConcern":
			transOpts = transOpts.WriteConcern(getWriteConcern(opt))
		case "readPreference":
			transOpts = transOpts.ReadPreference(getReadPref(opt))
		case "readConcern":
			transOpts = transOpts.ReadConcern(getReadConcern(opt))
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
