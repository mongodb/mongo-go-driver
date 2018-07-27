package mongo

import (
	"os"
	"reflect"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/event"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/sessionopt"
)

var ccStarted *event.CommandStartedEvent
var ccSucceeded *event.CommandSucceededEvent

var ccMonitor = &event.CommandMonitor{
	Started: func(cse *event.CommandStartedEvent) {
		ccStarted = cse
	},
	Succeeded: func(cse *event.CommandSucceededEvent) {
		ccSucceeded = cse
	},
}

var startingDoc = bson.NewDocument(
	bson.EC.Int32("hello", 5),
)

func compareOperationTimes(t *testing.T, expected *bson.Timestamp, actual *bson.Timestamp) {
	if expected.T != actual.T {
		t.Fatalf("T value mismatch; expected %d got %d", expected.T, actual.T)
	}

	if expected.I != actual.I {
		t.Fatalf("I value mismatch; expected %d got %d", expected.I, actual.I)
	}
}

func checkOperationTime(t *testing.T, cmd *bson.Document, shouldInclude bool) {
	rc, err := cmd.LookupErr("readConcern")
	testhelpers.RequireNil(t, err, "key read concern not found")

	_, err = rc.MutableDocument().LookupErr("afterClusterTime")
	if shouldInclude {
		testhelpers.RequireNil(t, err, "afterClusterTime not found")
	} else {
		testhelpers.RequireNotNil(t, err, "afterClusterTime found")
	}
}

func getOperationTime(t *testing.T, cmd *bson.Document) *bson.Timestamp {
	rc, err := cmd.LookupErr("readConcern")
	testhelpers.RequireNil(t, err, "key read concern not found")

	ct, err := rc.MutableDocument().LookupErr("afterClusterTime")
	testhelpers.RequireNil(t, err, "key afterClusterTime not found")

	timeT, timeI := ct.Timestamp()
	return &bson.Timestamp{
		T: timeT,
		I: timeI,
	}
}

func createReadFuncMap(t *testing.T, dbName string, collName string) (*Client, *Database, *Collection, []CollFunction) {
	client := createSessionsMonitoredClient(t, ccMonitor)
	db := client.Database(dbName)
	err := db.Drop(ctx)
	testhelpers.RequireNil(t, err, "error dropping database after creation: %s", err)

	coll := db.Collection(collName)
	coll.writeConcern = writeconcern.New(writeconcern.WMajority())

	functions := []CollFunction{
		{"Aggregate", reflect.ValueOf(coll.Aggregate), []interface{}{ctx, emptyDoc}},
		{"Count", reflect.ValueOf(coll.Count), []interface{}{ctx, emptyDoc}},
		{"Distinct", reflect.ValueOf(coll.Distinct), []interface{}{ctx, "field", emptyDoc}},
		{"Find", reflect.ValueOf(coll.Find), []interface{}{ctx, emptyDoc}},
		{"FindOne", reflect.ValueOf(coll.FindOne), []interface{}{ctx, emptyDoc}},
	}

	_, err = coll.InsertOne(ctx, startingDoc)
	testhelpers.RequireNil(t, err, "error inserting starting doc: %s", err)
	coll.writeConcern = nil
	return client, db, coll, functions
}

func checkReadConcern(t *testing.T, cmd *bson.Document, levelIncluded bool, expectedLevel string, optimeIncluded bool, expectedTime *bson.Timestamp) {
	rc, err := cmd.LookupErr("readConcern")
	testhelpers.RequireNil(t, err, "key readConcern not found")

	rcDoc := rc.MutableDocument()
	levelVal, err := rcDoc.LookupErr("level")
	if levelIncluded {
		testhelpers.RequireNil(t, err, "key level not found")
		if levelVal.StringValue() != expectedLevel {
			t.Fatalf("level mismatch. expected %s got %s", expectedLevel, levelVal.StringValue())
		}
	} else {
		testhelpers.RequireNotNil(t, err, "key level found")
	}

	ct, err := rcDoc.LookupErr("afterClusterTime")
	if optimeIncluded {
		testhelpers.RequireNil(t, err, "key afterClusterTime not found")
		ctT, ctI := ct.Timestamp()
		compareOperationTimes(t, expectedTime, &bson.Timestamp{ctT, ctI})
	} else {
		testhelpers.RequireNotNil(t, err, "key afterClusterTime found")
	}
}

func createWriteFuncMap(t *testing.T, dbName string, collName string) (*Client, *Database, *Collection, []CollFunction) {
	client := createSessionsMonitoredClient(t, ccMonitor)
	db := client.Database(dbName)
	err := db.Drop(ctx)
	testhelpers.RequireNil(t, err, "error dropping database after creation: %s", err)

	coll := db.Collection(collName)
	coll.writeConcern = writeconcern.New(writeconcern.WMajority())
	_, err = coll.InsertOne(ctx, startingDoc)
	testhelpers.RequireNil(t, err, "error inserting starting doc: %s", err)
	coll.writeConcern = nil

	iv := coll.Indexes()

	manyIndexes := []IndexModel{barIndex, bazIndex}

	functions := []CollFunction{
		{"InsertOne", reflect.ValueOf(coll.InsertOne), []interface{}{ctx, doc}},
		{"InsertMany", reflect.ValueOf(coll.InsertMany), []interface{}{ctx, []interface{}{doc2}}},
		{"DeleteOne", reflect.ValueOf(coll.DeleteOne), []interface{}{ctx, emptyDoc}},
		{"DeleteMany", reflect.ValueOf(coll.DeleteMany), []interface{}{ctx, emptyDoc}},
		{"UpdateOne", reflect.ValueOf(coll.UpdateOne), []interface{}{ctx, emptyDoc, updateDoc}},
		{"UpdateMany", reflect.ValueOf(coll.UpdateMany), []interface{}{ctx, emptyDoc, updateDoc}},
		{"ReplaceOne", reflect.ValueOf(coll.ReplaceOne), []interface{}{ctx, emptyDoc, emptyDoc}},
		{"FindOneAndDelete", reflect.ValueOf(coll.FindOneAndDelete), []interface{}{ctx, emptyDoc}},
		{"FindOneAndReplace", reflect.ValueOf(coll.FindOneAndReplace), []interface{}{ctx, emptyDoc, emptyDoc}},
		{"FindOneAndUpdate", reflect.ValueOf(coll.FindOneAndUpdate), []interface{}{ctx, emptyDoc, updateDoc}},
		{"DropCollection", reflect.ValueOf(coll.Drop), []interface{}{ctx}},
		{"DropDatabase", reflect.ValueOf(db.Drop), []interface{}{ctx}},
		{"ListCollections", reflect.ValueOf(db.ListCollections), []interface{}{ctx, emptyDoc}},
		{"ListDatabases", reflect.ValueOf(client.ListDatabases), []interface{}{ctx, emptyDoc}},
		{"CreateOneIndex", reflect.ValueOf(iv.CreateOne), []interface{}{ctx, fooIndex}},
		{"CreateManyIndexes", reflect.ValueOf(iv.CreateMany), []interface{}{ctx, manyIndexes}},
		{"DropOneIndex", reflect.ValueOf(iv.DropOne), []interface{}{ctx, "barIndex"}},
		{"DropAllIndexes", reflect.ValueOf(iv.DropAll), []interface{}{ctx}},
		{"ListIndexes", reflect.ValueOf(iv.List), []interface{}{ctx}},
	}

	return client, db, coll, functions
}

func skipIfSessionsSupported(t *testing.T, db *Database) {
	if os.Getenv("TOPOLOGY") != "server" {
		t.Skip("skipping topology")
	}

	serverVersion, err := getServerVersion(db)
	testhelpers.RequireNil(t, err, "error getting server version: %s", err)

	if compareVersions(t, serverVersion, "3.6") >= 0 {
		t.Skip("skipping server version")
	}
}

func TestCausalConsistency(t *testing.T) {
	t.Run("TestOperationTimeNil", func(t *testing.T) {
		// When a ClientSession is first created the operationTime has no value

		client := createSessionsMonitoredClient(t, ccMonitor)
		sess, err := client.StartSession()
		testhelpers.RequireNil(t, err, "error creating session: %s", err)
		defer sess.EndSession()

		if sess.OperationTime != nil {
			t.Fatal("operation time is not nil")
		}
	})

	t.Run("TestNoTimeOnFirstCommand", func(t *testing.T) {
		// First read in causally consistent session must not send afterClusterTime to the server

		client := createSessionsMonitoredClient(t, ccMonitor)
		sess, err := client.StartSession(sessionopt.CausalConsistency(true))
		testhelpers.RequireNil(t, err, "error creating session: %s", err)
		defer sess.EndSession()

		db := client.Database("FirstCommandDB")
		err = db.Drop(ctx)
		testhelpers.RequireNil(t, err, "error dropping db: %s", err)

		coll := db.Collection("FirstCommandColl")
		_, err = coll.Find(ctx, emptyDoc, sess)
		testhelpers.RequireNil(t, err, "error running find: %s", err)

		testhelpers.RequireNotNil(t, ccStarted, "no started command found")
		if ccStarted.CommandName != "find" {
			t.Fatalf("started command %s is not a find command", ccStarted.CommandName)
		}

		checkOperationTime(t, ccStarted.Command, false)
	})

	t.Run("TestOperationTimeUpdated", func(t *testing.T) {
		//The first read or write on a ClientSession should update the operationTime of the ClientSession, even if there is an error

		skipInvalidTopology(t)
		skipIfBelow36(t)

		client := createSessionsMonitoredClient(t, ccMonitor)
		sess, err := client.StartSession()
		testhelpers.RequireNil(t, err, "error starting session: %s", err)
		defer sess.EndSession()

		db := client.Database("OptimeUpdateDB")
		err = db.Drop(ctx)
		testhelpers.RequireNil(t, err, "error dropping db: %s", err)

		coll := db.Collection("OptimeUpdateColl")
		_, _ = coll.Find(ctx, emptyDoc, sess)

		testhelpers.RequireNotNil(t, ccSucceeded, "no succeeded command")
		serverT, serverI := ccSucceeded.Reply.Lookup("operationTime").Timestamp()

		testhelpers.RequireNotNil(t, sess.OperationTime, "operation time nil after first command")
		compareOperationTimes(t, &bson.Timestamp{serverT, serverI}, sess.OperationTime)
	})

	t.Run("TestOperationTimeSent", func(t *testing.T) {
		// findOne followed by another read operation should include operationTime returned by the server for the first
		// operation in the afterClusterTime parameter of the second operation

		skipInvalidTopology(t)
		skipIfBelow36(t)

		client, _, coll, readMap := createReadFuncMap(t, "OptimeSentDB", "OptimeSentColl")

		for _, tc := range readMap {
			t.Run(tc.name, func(t *testing.T) {
				sess, err := client.StartSession(sessionopt.CausalConsistency(true))
				testhelpers.RequireNil(t, err, "error creating session for %s: %s", tc.name, err)
				defer sess.EndSession()

				opts := append(tc.opts, sess)
				docRes := coll.FindOne(ctx, emptyDoc, sess)
				testhelpers.RequireNil(t, docRes.err, "find one error for %s: %s", tc.name, docRes.err)

				currOptime := sess.OperationTime

				returnVals := tc.f.Call(getOptValues(opts))
				err = getReturnError(returnVals)
				testhelpers.RequireNil(t, err, "error running %s: %s", tc.name, err)

				testhelpers.RequireNotNil(t, ccStarted, "no started command")
				sentOptime := getOperationTime(t, ccStarted.Command)

				compareOperationTimes(t, currOptime, sentOptime)
			})
		}
	})

	t.Run("TestWriteThenRead", func(t *testing.T) {
		// Any write operation followed by findOne should include operationTime of first op in afterClusterTime parameter of
		// second op

		skipInvalidTopology(t)
		skipIfBelow36(t)

		client, _, coll, writeMap := createWriteFuncMap(t, "WriteThenReadDB", "WriteThenReadColl")

		for _, tc := range writeMap {
			t.Run(tc.name, func(t *testing.T) {
				sess, err := client.StartSession(sessionopt.CausalConsistency(true))
				testhelpers.RequireNil(t, err, "error starting session: %s", err)
				defer sess.EndSession()

				opts := append(tc.opts, sess)
				returnVals := tc.f.Call(getOptValues(opts))
				err = getReturnError(returnVals)
				testhelpers.RequireNil(t, err, "error running %s: %s", tc.name, err)

				currentOptime := sess.OperationTime
				_ = coll.FindOne(ctx, emptyDoc, sess)

				testhelpers.RequireNotNil(t, ccStarted, "no started command")
				sentOptime := getOperationTime(t, ccStarted.Command)

				compareOperationTimes(t, currentOptime, sentOptime)
			})
		}
	})

	t.Run("TestNonConsistentRead", func(t *testing.T) {
		// Read op in a non causally-consistent session should not include afterClusterTime in cmd sent to server

		skipInvalidTopology(t)
		skipIfBelow36(t)

		client := createSessionsMonitoredClient(t, ccMonitor)
		sess, err := client.StartSession(sessionopt.CausalConsistency(false))
		testhelpers.RequireNil(t, err, "error creating session: %s", err)
		defer sess.EndSession()

		db := client.Database("NonConsistentReadDB")
		err = db.Drop(ctx)
		testhelpers.RequireNil(t, err, "error dropping db: %s", err)

		coll := db.Collection("NonConsistentReadColl")
		_, _ = coll.Find(ctx, emptyDoc, sess)

		testhelpers.RequireNotNil(t, ccStarted, "no started command")
		if ccStarted.CommandName != "find" {
			t.Fatalf("started command %s was not a find command", ccStarted.CommandName)
		}

		checkOperationTime(t, ccStarted.Command, false)
	})

	t.Run("TestInvalidTopology", func(t *testing.T) {
		// A read op in a causally consistent session does not include afterClusterTime in a deployment that does not
		// support cluster times

		client := createSessionsMonitoredClient(t, ccMonitor)
		db := client.Database("InvalidTopologyDB")
		err := db.Drop(ctx)
		testhelpers.RequireNil(t, err, "error dropping db: %s", err)

		skipIfSessionsSupported(t, db)

		sess, err := client.StartSession(sessionopt.CausalConsistency(true))
		testhelpers.RequireNil(t, err, "error starting session: %s", err)
		defer sess.EndSession()

		coll := db.Collection("InvalidTopologyColl")
		_, _ = coll.Find(ctx, emptyDoc, sess)

		testhelpers.RequireNotNil(t, ccStarted, "no started command found")
		if ccStarted.CommandName != "find" {
			t.Fatalf("started command %s was not a find command", ccStarted.CommandName)
		}

		checkOperationTime(t, ccStarted.Command, false)
	})

	t.Run("TestDefaultReadConcern", func(t *testing.T) {
		// When using the default server read concern, the readConcern parameter in the command sent to the server should
		// not include a level field

		skipInvalidTopology(t)
		skipIfBelow36(t)

		client := createSessionsMonitoredClient(t, ccMonitor)
		sess, err := client.StartSession(sessionopt.CausalConsistency(true))
		testhelpers.RequireNil(t, err, "error starting session: %s", err)

		db := client.Database("DefaultReadConcernDB")
		err = db.Drop(ctx)
		testhelpers.RequireNil(t, err, "error dropping db: %s", err)

		coll := db.Collection("DefaultReadConcernColl")
		coll.readConcern = readconcern.New()
		_ = coll.FindOne(ctx, emptyDoc, sess)

		currOptime := sess.OperationTime
		_ = coll.FindOne(ctx, emptyDoc, sess)

		testhelpers.RequireNotNil(t, ccStarted, "no started command found")
		if ccStarted.CommandName != "find" {
			t.Fatalf("started command %s was not a find command", ccStarted.CommandName)
		}

		checkReadConcern(t, ccStarted.Command, false, "", true, currOptime)
	})

	t.Run("TestCustomReadConcern", func(t *testing.T) {
		// When using a custom read concern, the readConcern field in commands sent to the server should have level
		// and afterClusterTime

		skipInvalidTopology(t)
		skipIfBelow36(t)

		client := createSessionsMonitoredClient(t, ccMonitor)
		sess, err := client.StartSession(sessionopt.CausalConsistency(true))
		testhelpers.RequireNil(t, err, "error starting session: %s", err)
		defer sess.EndSession()

		db := client.Database("CustomReadConcernDB")
		err = db.Drop(ctx)
		testhelpers.RequireNil(t, err, "error dropping db: %s", err)

		coll := db.Collection("CustomReadConcernColl")
		coll.readConcern = readconcern.Majority()

		_ = coll.FindOne(ctx, emptyDoc, sess)
		currOptime := sess.OperationTime

		_ = coll.FindOne(ctx, emptyDoc, sess)

		testhelpers.RequireNotNil(t, ccStarted, "no started command found")
		if ccStarted.CommandName != "find" {
			t.Fatalf("started command %s was not a find command", ccStarted.CommandName)
		}

		checkReadConcern(t, ccStarted.Command, true, "majority", true, currOptime)
	})

	t.Run("TestUnacknowledgedWrite", func(t *testing.T) {
		// Unacknowledged write should not update the operationTime property of a session

		client := createSessionsMonitoredClient(t, nil)
		sess, err := client.StartSession(sessionopt.CausalConsistency(true))
		testhelpers.RequireNil(t, err, "error starting session: %s", err)
		defer sess.EndSession()

		db := client.Database("UnackWriteDB")
		err = db.Drop(ctx)
		testhelpers.RequireNil(t, err, "error dropping db: %s", err)

		coll := db.Collection("UnackWriteColl")
		coll.writeConcern = writeconcern.New(writeconcern.W(0))
		_, _ = coll.InsertOne(ctx, doc)

		if sess.OperationTime != nil {
			t.Fatal("operation time updated for unacknowledged write")
		}
	})

	t.Run("TestInvalidTopologyClusterTime", func(t *testing.T) {
		// $clusterTime should not be included in commands if the deployment does not support cluster times

		client := createSessionsMonitoredClient(t, ccMonitor)
		db := client.Database("InvalidTopCTDB")
		err := db.Drop(ctx)
		testhelpers.RequireNil(t, err, "error dropping db: %s", err)

		skipIfSessionsSupported(t, db)

		coll := db.Collection("InvalidTopCTColl")
		_ = coll.FindOne(ctx, emptyDoc)

		testhelpers.RequireNotNil(t, ccStarted, "no started command found")
		if ccStarted.CommandName != "find" {
			t.Fatalf("started command %s was not a find command", ccStarted.CommandName)
		}

		_, err = ccStarted.Command.LookupErr("$clusterTime")
		testhelpers.RequireNotNil(t, err, "$clusterTime found for invalid topology")
	})

	t.Run("TestValidTopologyClusterTime", func(t *testing.T) {
		// $clusterTime should be included in commands if the deployment supports cluster times

		skipInvalidTopology(t)
		skipIfBelow36(t)

		client := createSessionsMonitoredClient(t, ccMonitor)
		db := client.Database("ValidTopCTDB")
		err := db.Drop(ctx)
		testhelpers.RequireNil(t, err, "error dropping db: %s", err)

		coll := db.Collection("ValidTopCTColl")
		_ = coll.FindOne(ctx, emptyDoc)

		testhelpers.RequireNotNil(t, ccStarted, "no started command found")
		if ccStarted.CommandName != "find" {
			t.Fatalf("started command %s was not a find command", ccStarted.CommandName)
		}

		_, err = ccStarted.Command.LookupErr("$clusterTime")
		testhelpers.RequireNil(t, err, "$clusterTime found for invalid topology")
	})
}
