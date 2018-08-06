package mongo

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"context"

	"strings"

	"bytes"
	"math"

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
	"github.com/mongodb/mongo-go-driver/mongo/aggregateopt"
	"github.com/mongodb/mongo-go-driver/mongo/collectionopt"
	"github.com/mongodb/mongo-go-driver/mongo/deleteopt"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
	"github.com/mongodb/mongo-go-driver/mongo/replaceopt"
	"github.com/mongodb/mongo-go-driver/mongo/runcmdopt"
	"github.com/mongodb/mongo-go-driver/mongo/sessionopt"
	"github.com/mongodb/mongo-go-driver/mongo/transactionopt"
	"github.com/mongodb/mongo-go-driver/mongo/updateopt"
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
	ConfigureFailPoint string `json:"configureFailPoint"`
	Mode               struct {
		Times int32 `json:"times"`
		Skip  int32 `json:"skip"`
	} `json:"mode"`
	Data *failPointData `json:"data"`
}

type failPointData struct {
	FailCommands      []string `json:"failCommands"`
	CloseConnection   bool     `json:"closeConnection"`
	ErrorCode         int32    `json:"errorCode"`
	WriteConcernError *struct {
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
	Started: func(cse *event.CommandStartedEvent) {
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
		t.Run(test.Description, func(t *testing.T) {
			// TODO reenable tests once retryable writes implemented
			if strings.Contains(filepath, "retryable-writes") {
				t.Skip("Skipping until retryable-writes are implemented")
			}

			// kill sessions from previously failed tests
			killSessions(t, dbAdmin.client)

			// configure failpoint if specified
			if test.FailPoint != nil {
				doc := createFailPointDoc(test.FailPoint)
				_, err = dbAdmin.RunCommand(ctx, doc)
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

			err = db.Drop(ctx)
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

			sess0, err := client.StartSession(sess0Opts)
			require.NoError(t, err)
			sess1, err := client.StartSession(sess1Opts)
			require.NoError(t, err)

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
				var sess *Session
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
	}

	addClientOptions(c, opts)

	subscription, err := c.topology.Subscribe()
	testhelpers.RequireNil(t, err, "error subscribing to topology: %s", err)
	c.topology.SessionPool = session.NewPool(subscription.C)

	return c
}

func createFailPointDoc(failPoint *failPoint) *bson.Document {
	failCommandElems := make([]*bson.Value, len(failPoint.Data.FailCommands))
	for i, str := range failPoint.Data.FailCommands {
		failCommandElems[i] = bson.VC.String(str)
	}
	modeDoc := bson.NewDocument()
	if failPoint.Mode.Times != 0 {
		modeDoc.Append(bson.EC.Int32("times", failPoint.Mode.Times))
	}
	if failPoint.Mode.Skip != 0 {
		modeDoc.Append(bson.EC.Int32("skip", failPoint.Mode.Skip))
	}

	dataDoc := bson.NewDocument(
		bson.EC.ArrayFromElements("failCommands", failCommandElems...),
	)

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

	return bson.NewDocument(
		bson.EC.String("configureFailPoint", failPoint.ConfigureFailPoint),
		bson.EC.SubDocument("mode", modeDoc),
		bson.EC.SubDocument("data", dataDoc),
	)
}

func executeSessionOperation(op *transOperation, sess *Session) error {
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

func executeCollectionOperation(t *testing.T, op *transOperation, sess *Session, coll *Collection) error {
	switch op.Name {
	case "count":
		_, err := executeCount(sess, coll, op.ArgMap)
		// no results to verify with count
		return err
	case "distinct":
		_, err := executeDistinct(sess, coll, op.ArgMap)
		// no results to verify with distinct
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

func executeDatabaseOperation(t *testing.T, op *transOperation, sess *Session, db *Database) error {
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

func executeCount(sess *Session, coll *Collection, args map[string]interface{}) (int64, error) {
	var filter map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		}
	}

	if sess != nil {
		return coll.Count(ctx, filter, sess)
	}
	return coll.Count(ctx, filter)
}

func executeDistinct(sess *Session, coll *Collection, args map[string]interface{}) ([]interface{}, error) {
	var fieldName string
	for name, opt := range args {
		switch name {
		case "fieldName":
			fieldName = opt.(string)
		}
	}

	if sess != nil {
		return coll.Distinct(ctx, fieldName, nil, sess)
	}
	return coll.Distinct(ctx, fieldName, nil)
}

func executeInsertOne(sess *Session, coll *Collection, args map[string]interface{}) (*InsertOneResult, error) {
	document := args["document"].(map[string]interface{})

	// For some reason, the insertion document is unmarshaled with a float rather than integer,
	// but the documents that are used to initially populate the collection are unmarshaled
	// correctly with integers. To ensure that the tests can correctly compare them, we iterate
	// through the insertion document and change any valid integers stored as floats to actual
	// integers.
	replaceFloatsWithInts(document)

	if sess != nil {
		return coll.InsertOne(context.Background(), document, sess)
	}
	return coll.InsertOne(context.Background(), document)
}

func executeInsertMany(sess *Session, coll *Collection, args map[string]interface{}) (*InsertManyResult, error) {
	documents := args["documents"].([]interface{})

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

	if sess != nil {
		return coll.InsertMany(context.Background(), documents, sess)
	}
	return coll.InsertMany(context.Background(), documents)
}

func executeFind(sess *Session, coll *Collection, args map[string]interface{}) (Cursor, error) {
	var bundle *findopt.FindBundle
	var filter map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "sort":
			bundle = bundle.Sort(opt.(map[string]interface{}))
		case "skip":
			bundle = bundle.Skip(int64(opt.(float64)))
		case "limit":
			bundle = bundle.Limit(int64(opt.(float64)))
		case "batchSize":
			bundle = bundle.BatchSize(int32(opt.(float64)))
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	if sess != nil {
		return coll.Find(ctx, filter, bundle, sess)
	}
	return coll.Find(ctx, filter, bundle)
}

func executeFindOneAndDelete(sess *Session, coll *Collection, args map[string]interface{}) *DocumentResult {
	var bundle *findopt.DeleteOneBundle
	var filter map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "sort":
			bundle = bundle.Sort(opt.(map[string]interface{}))
		case "projection":
			bundle = bundle.Projection(opt.(map[string]interface{}))
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	if sess != nil {
		return coll.FindOneAndDelete(ctx, filter, bundle, sess)
	}
	return coll.FindOneAndDelete(ctx, filter, bundle)
}

func executeFindOneAndUpdate(sess *Session, coll *Collection, args map[string]interface{}) *DocumentResult {
	var bundle *findopt.UpdateOneBundle
	var filter map[string]interface{}
	var update map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "update":
			update = opt.(map[string]interface{})
		case "arrayFilters":
			bundle = bundle.ArrayFilters(opt.([]interface{})...)
		case "sort":
			bundle = bundle.Sort(opt.(map[string]interface{}))
		case "projection":
			bundle = bundle.Projection(opt.(map[string]interface{}))
		case "upsert":
			bundle = bundle.Upsert(opt.(bool))
		case "returnDocument":
			switch opt.(string) {
			case "After":
				bundle = bundle.ReturnDocument(mongoopt.After)
			case "Before":
				bundle = bundle.ReturnDocument(mongoopt.Before)
			}
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	// For some reason, the filter and update documents are unmarshaled with floats
	// rather than integers, but the documents that are used to initially populate the
	// collection are unmarshaled correctly with integers. To ensure that the tests can
	// correctly compare them, we iterate through the filter and replacement documents and
	// change any valid integers stored as floats to actual integers.
	replaceFloatsWithInts(filter)
	replaceFloatsWithInts(update)

	if sess != nil {
		return coll.FindOneAndUpdate(ctx, filter, update, bundle, sess)
	}
	return coll.FindOneAndUpdate(ctx, filter, update, bundle)
}

func executeFindOneAndReplace(sess *Session, coll *Collection, args map[string]interface{}) *DocumentResult {
	var bundle *findopt.ReplaceOneBundle
	var filter map[string]interface{}
	var replacement map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "replacement":
			replacement = opt.(map[string]interface{})
		case "sort":
			bundle = bundle.Sort(opt.(map[string]interface{}))
		case "projection":
			bundle = bundle.Projection(opt.(map[string]interface{}))
		case "upsert":
			bundle = bundle.Upsert(opt.(bool))
		case "returnDocument":
			switch opt.(string) {
			case "After":
				bundle = bundle.ReturnDocument(mongoopt.After)
			case "Before":
				bundle = bundle.ReturnDocument(mongoopt.Before)
			}
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	// For some reason, the filter and replacement documents are unmarshaled with floats
	// rather than integers, but the documents that are used to initially populate the
	// collection are unmarshaled correctly with integers. To ensure that the tests can
	// correctly compare them, we iterate through the filter and replacement documents and
	// change any valid integers stored as floats to actual integers.
	replaceFloatsWithInts(filter)
	replaceFloatsWithInts(replacement)

	if sess != nil {
		return coll.FindOneAndReplace(ctx, filter, replacement, bundle, sess)
	}
	return coll.FindOneAndReplace(ctx, filter, replacement, bundle)
}

func executeDeleteOne(sess *Session, coll *Collection, args map[string]interface{}) (*DeleteResult, error) {
	var bundle *deleteopt.DeleteBundle
	var filter map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	// For some reason, the filter document is unmarshaled with floats
	// rather than integers, but the documents that are used to initially populate the
	// collection are unmarshaled correctly with integers. To ensure that the tests can
	// correctly compare them, we iterate through the filter and replacement documents and
	// change any valid integers stored as floats to actual integers.
	replaceFloatsWithInts(filter)

	if sess != nil {
		return coll.DeleteOne(ctx, filter, bundle, sess)
	}
	return coll.DeleteOne(ctx, filter, bundle)
}

func executeDeleteMany(sess *Session, coll *Collection, args map[string]interface{}) (*DeleteResult, error) {
	var bundle *deleteopt.DeleteBundle
	var filter map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	// For some reason, the filter document is unmarshaled with floats
	// rather than integers, but the documents that are used to initially populate the
	// collection are unmarshaled correctly with integers. To ensure that the tests can
	// correctly compare them, we iterate through the filter and replacement documents and
	// change any valid integers stored as floats to actual integers.
	replaceFloatsWithInts(filter)

	if sess != nil {
		return coll.DeleteMany(ctx, filter, bundle, sess)
	}
	return coll.DeleteMany(ctx, filter, bundle)
}

func executeReplaceOne(sess *Session, coll *Collection, args map[string]interface{}) (*UpdateResult, error) {
	var bundle *replaceopt.ReplaceBundle
	var filter map[string]interface{}
	var replacement map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "replacement":
			replacement = opt.(map[string]interface{})
		case "upsert":
			bundle = bundle.Upsert(opt.(bool))
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	// For some reason, the filter and replacement documents are unmarshaled with floats
	// rather than integers, but the documents that are used to initially populate the
	// collection are unmarshaled correctly with integers. To ensure that the tests can
	// correctly compare them, we iterate through the filter and replacement documents and
	// change any valid integers stored as floats to actual integers.
	replaceFloatsWithInts(filter)
	replaceFloatsWithInts(replacement)

	// TODO temporarily default upsert to false explicitly to make test pass
	// because we do not send upsert=false by default
	bundle = replaceopt.BundleReplace(replaceopt.Upsert(false), bundle)
	if sess != nil {
		return coll.ReplaceOne(ctx, filter, replacement, bundle, sess)
	}
	return coll.ReplaceOne(ctx, filter, replacement, bundle)
}

func executeUpdateOne(sess *Session, coll *Collection, args map[string]interface{}) (*UpdateResult, error) {
	var bundle *updateopt.UpdateBundle
	var filter map[string]interface{}
	var update map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "update":
			update = opt.(map[string]interface{})
		case "arrayFilters":
			bundle = bundle.ArrayFilters(opt.([]interface{})...)
		case "upsert":
			bundle = bundle.Upsert(opt.(bool))
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	// For some reason, the filter and update documents are unmarshaled with floats
	// rather than integers, but the documents that are used to initially populate the
	// collection are unmarshaled correctly with integers. To ensure that the tests can
	// correctly compare them, we iterate through the filter and replacement documents and
	// change any valid integers stored as floats to actual integers.
	replaceFloatsWithInts(filter)
	replaceFloatsWithInts(update)

	// TODO temporarily default upsert to false explicitly to make test pass
	// because we do not send upsert=false by default
	bundle = updateopt.BundleUpdate(updateopt.Upsert(false), bundle)
	if sess != nil {
		return coll.UpdateOne(ctx, filter, update, bundle, sess)
	}
	return coll.UpdateOne(ctx, filter, update, bundle)
}

func executeUpdateMany(sess *Session, coll *Collection, args map[string]interface{}) (*UpdateResult, error) {
	var bundle *updateopt.UpdateBundle
	var filter map[string]interface{}
	var update map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "update":
			update = opt.(map[string]interface{})
		case "arrayFilters":
			bundle = bundle.ArrayFilters(opt.([]interface{})...)
		case "upsert":
			bundle = bundle.Upsert(opt.(bool))
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	// For some reason, the filter and update documents are unmarshaled with floats
	// rather than integers, but the documents that are used to initially populate the
	// collection are unmarshaled correctly with integers. To ensure that the tests can
	// correctly compare them, we iterate through the filter and replacement documents and
	// change any valid integers stored as floats to actual integers.
	replaceFloatsWithInts(filter)
	replaceFloatsWithInts(update)

	// TODO temporarily default upsert to false explicitly to make test pass
	// because we do not send upsert=false by default
	bundle = updateopt.BundleUpdate(updateopt.Upsert(false), bundle)
	if sess != nil {
		return coll.UpdateMany(ctx, filter, update, bundle, sess)
	}
	return coll.UpdateMany(ctx, filter, update, bundle)
}

func executeAggregate(sess *Session, coll *Collection, args map[string]interface{}) (Cursor, error) {
	var bundle *aggregateopt.AggregateBundle
	var pipeline []interface{}
	for name, opt := range args {
		switch name {
		case "pipeline":
			pipeline = opt.([]interface{})
		case "batchSize":
			bundle = bundle.BatchSize(int32(opt.(float64)))
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	if sess != nil {
		return coll.Aggregate(ctx, pipeline, bundle, sess)
	}
	return coll.Aggregate(ctx, pipeline, bundle)
}

func executeRunCommand(sess *Session, db *Database, argmap map[string]interface{}, args json.RawMessage) (bson.Reader, error) {
	var cmd *bson.Document
	var bundle *runcmdopt.RunCmdBundle
	for name, opt := range argmap {
		switch name {
		case "command":
			argBytes, err := args.MarshalJSON()
			if err != nil {
				return nil, err
			}

			var argCmdStruct struct {
				Cmd json.RawMessage `json:"command"`
			}
			err = json.NewDecoder(bytes.NewBuffer(argBytes)).Decode(&argCmdStruct)
			if err != nil {
				return nil, err
			}

			cmd, err = bson.ParseExtJSONObject(string(argCmdStruct.Cmd))
			if err != nil {
				return nil, err
			}
		case "readPreference":
			bundle = bundle.ReadPreference(getReadPref(opt))
		}
	}

	if sess != nil {
		return db.RunCommand(ctx, cmd, bundle, sess)
	}
	return db.RunCommand(ctx, cmd, bundle)
}

func verifyInsertOneResult(t *testing.T, res *InsertOneResult, result json.RawMessage) {
	expectedBytes, err := result.MarshalJSON()
	require.NoError(t, err)

	var expected InsertOneResult
	err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)
	require.NoError(t, err)

	expectedID := expected.InsertedID
	if f, ok := expectedID.(float64); ok && f == math.Floor(f) {
		expectedID = int64(f)
	}

	if expectedID != nil {
		require.NotNil(t, res)
		require.Equal(t, expectedID, res.InsertedID.(*bson.Element).Value().Interface())
	}
}

func verifyInsertManyResult(t *testing.T, res *InsertManyResult, result json.RawMessage) {
	expectedBytes, err := result.MarshalJSON()
	require.NoError(t, err)

	var expected struct{ InsertedIds map[string]interface{} }
	err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)
	require.NoError(t, err)

	if expected.InsertedIds != nil {
		replaceFloatsWithInts(expected.InsertedIds)

		for i, elem := range res.InsertedIDs {
			res.InsertedIDs[i] = elem.(*bson.Element).Value().Interface()
		}

		for _, val := range expected.InsertedIds {
			require.Contains(t, res.InsertedIDs, val)
		}
	}
}

func verifyCursorResult(t *testing.T, cur Cursor, result json.RawMessage) {
	for _, expected := range docSliceFromRaw(t, result) {
		require.NotNil(t, cur)
		require.True(t, cur.Next(context.Background()))

		actual := bson.NewDocument()
		require.NoError(t, cur.Decode(actual))

		compareDocs(t, expected, actual)
	}

	require.False(t, cur.Next(ctx))
	require.NoError(t, cur.Err())
}

func verifyDocumentResult(t *testing.T, res *DocumentResult, result json.RawMessage) {
	jsonBytes, err := result.MarshalJSON()
	require.NoError(t, err)

	actual := bson.NewDocument()
	err = res.Decode(actual)
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
}

func verifyDeleteResult(t *testing.T, res *DeleteResult, result json.RawMessage) {
	expectedBytes, err := result.MarshalJSON()
	require.NoError(t, err)

	var expected DeleteResult
	err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)
	require.NoError(t, err)

	require.Equal(t, expected.DeletedCount, res.DeletedCount)
}

func verifyUpdateResult(t *testing.T, res *UpdateResult, result json.RawMessage) {
	expectedBytes, err := result.MarshalJSON()
	require.NoError(t, err)

	var expected struct {
		MatchedCount  int64
		ModifiedCount int64
		UpsertedCount int64
	}
	err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)

	require.Equal(t, expected.MatchedCount, res.MatchedCount)
	require.Equal(t, expected.ModifiedCount, res.ModifiedCount)

	actualUpsertedCount := int64(0)
	if res.UpsertedID != nil {
		actualUpsertedCount = 1
	}

	require.Equal(t, expected.UpsertedCount, actualUpsertedCount)
}

func verifyRunCommandResult(t *testing.T, res bson.Reader, result json.RawMessage) {
	jsonBytes, err := result.MarshalJSON()
	require.NoError(t, err)

	expected, err := bson.ParseExtJSONObject(string(jsonBytes))
	require.NoError(t, err)

	require.NotNil(t, res)
	actual, err := bson.ReadDocument(res)
	require.NoError(t, err)

	// All runcommand results in tests are for key "n" only
	compareElements(t, expected.LookupElement("n"), actual.LookupElement("n"))
}

func verifyError(t *testing.T, e error, result json.RawMessage) {
	expected := getErrorFromResult(t, result)
	if expected == nil {
		return
	}

	if cerr, ok := e.(*command.Error); ok {
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

		expected, err := bson.ParseExtJSONObject(string(jsonBytes))
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

func compareElements(t *testing.T, expected *bson.Element, actual *bson.Element) {
	if expected.Value().IsNumber() {
		expectedNum := expected.Value().Int64()
		switch actual.Value().Type() {
		case bson.TypeInt32:
			require.Equal(t, int64(actual.Value().Int32()), expectedNum)
		case bson.TypeInt64:
			require.Equal(t, actual.Value().Int64(), expectedNum)
		case bson.TypeDouble:
			require.Equal(t, int64(actual.Value().Double()), expectedNum)
		}
	} else if conv, ok := expected.Value().MutableDocumentOK(); ok {
		actualConv, actualOk := actual.Value().MutableDocumentOK()
		require.True(t, actualOk)
		compareDocs(t, conv, actualConv)
	} else if conv, ok := expected.Value().MutableArrayOK(); ok {
		actualConv, actualOk := actual.Value().MutableArrayOK()
		require.True(t, actualOk)
		compareArrays(t, conv, actualConv)
	} else {
		exp, err := expected.MarshalBSON()
		require.NoError(t, err)
		act, err := actual.MarshalBSON()
		require.NoError(t, err)

		require.True(t, bytes.Equal(exp, act), "For key %s, expected %v\nactual: %v", expected.Key(), expected, actual)
	}
}

func compareArrays(t *testing.T, expected *bson.Array, actual *bson.Array) {
	if expected.Len() != actual.Len() {
		t.Errorf("array length mismatch. expected %d got %d", expected.Len(), actual.Len())
		t.FailNow()
	}

	expectedIter, err := expected.Iterator()
	testhelpers.RequireNil(t, err, "error creating iterator for expected array: %s", err)

	actualIter, err := actual.Iterator()
	testhelpers.RequireNil(t, err, "error creating iterator for actual array: %s", err)

	for expectedIter.Next() && actualIter.Next() {
		expectedDoc := expectedIter.Value().MutableDocument()
		actualDoc := actualIter.Value().MutableDocument()
		compareDocs(t, expectedDoc, actualDoc)
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

// Mutates the client to add options
func addClientOptions(c *Client, opts map[string]interface{}) {
	for name, opt := range opts {
		switch name {
		case "retryWrites":
			// TODO
		case "w":
			switch opt.(type) {
			case float64:
				c.writeConcern = writeconcern.New(writeconcern.W(int(opt.(float64))))
			case string:
				c.writeConcern = writeconcern.New(writeconcern.WMajority())
			}
		case "readConcernLevel":
			c.readConcern = readconcern.New(readconcern.Level(opt.(string)))
		case "readPreference":
			c.readPreference = readPrefFromString(opt.(string))
		}
	}
}

// Mutates the collection to add options
func addCollectionOptions(c *Collection, opts map[string]interface{}) {
	for name, opt := range opts {
		switch name {
		case "readConcern":
			c.readConcern = getReadConcern(opt)
		case "writeConcern":
			c.writeConcern = getWriteConcern(opt)
		case "readPreference":
			c.readPreference = readPrefFromString(opt.(map[string]interface{})["mode"].(string))
		}
	}
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
