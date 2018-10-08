// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"reflect"
	"testing"

	"fmt"
	"os"
	"time"

	"bytes"
	"strings"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/connection"
	"github.com/mongodb/mongo-go-driver/core/connstring"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/event"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/core/uuid"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/collectionopt"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"github.com/stretchr/testify/require"
)

var sessionStarted *event.CommandStartedEvent
var sessionSucceeded *event.CommandSucceededEvent
var sessionsMonitoredTop *topology.Topology

var sessionsMonitor = &event.CommandMonitor{
	Started: func(ctx context.Context, cse *event.CommandStartedEvent) {
		sessionStarted = cse
	},
	Succeeded: func(ctx context.Context, cse *event.CommandSucceededEvent) {
		sessionSucceeded = cse
	},
}

type CollFunction struct {
	name string
	coll *Collection
	iv   *IndexView
	f    func(SessionContext) error
}

var ctx = context.Background()
var emptyDoc = bson.NewDocument()
var updateDoc = bson.NewDocument(
	bson.EC.SubDocument("$inc", bson.NewDocument(
		bson.EC.Int32("x", 1),
	)),
)
var doc = bson.NewDocument(
	bson.EC.Int32("x", 1),
)
var doc2 = bson.NewDocument(
	bson.EC.Int32("y", 1),
)

var fooIndex = IndexModel{
	Keys: bson.NewDocument(
		bson.EC.Int32("foo", -1),
	),
	Options: NewIndexOptionsBuilder().Name("fooIndex").Build(),
}

var barIndex = IndexModel{
	Keys: bson.NewDocument(
		bson.EC.Int32("bar", -1),
	),
	Options: NewIndexOptionsBuilder().Name("barIndex").Build(),
}

var bazIndex = IndexModel{
	Keys: bson.NewDocument(
		bson.EC.Int32("baz", -1),
	),
	Options: NewIndexOptionsBuilder().Name("bazIndex").Build(),
}

func createFuncMap(t *testing.T, dbName string, collName string, monitored bool) (*Client, *Database, *Collection, []CollFunction) {
	var client *Client

	if monitored {
		client = createSessionsMonitoredClient(t, sessionsMonitor)
	} else {
		client = createTestClient(t)
	}

	db := client.Database(dbName)
	err := db.Drop(ctx)
	testhelpers.RequireNil(t, err, "error dropping database after creation: %s", err)

	coll := db.Collection(collName)
	iv := coll.Indexes()

	manyIndexes := []IndexModel{barIndex, bazIndex}

	functions := []CollFunction{
		{"InsertOne", coll, nil, func(mctx SessionContext) error { _, err := coll.InsertOne(mctx, doc); return err }},
		{"InsertMany", coll, nil, func(mcxt SessionContext) error { _, err := coll.InsertMany(mcxt, []interface{}{doc2}); return err }},
		{"DeleteOne", coll, nil, func(mctx SessionContext) error { _, err := coll.DeleteOne(mctx, emptyDoc); return err }},
		{"DeleteMany", coll, nil, func(mctx SessionContext) error { _, err := coll.DeleteMany(mctx, emptyDoc); return err }},
		{"UpdateOne", coll, nil, func(mctx SessionContext) error { _, err := coll.UpdateOne(mctx, emptyDoc, updateDoc); return err }},
		{"UpdateMany", coll, nil, func(mctx SessionContext) error { _, err := coll.UpdateMany(mctx, emptyDoc, updateDoc); return err }},
		{"ReplaceOne", coll, nil, func(mctx SessionContext) error { _, err := coll.ReplaceOne(mctx, emptyDoc, emptyDoc); return err }},
		{"Aggregate", coll, nil, func(mctx SessionContext) error { _, err := coll.Aggregate(mctx, emptyDoc); return err }},
		{"Count", coll, nil, func(mctx SessionContext) error { _, err := coll.Count(mctx, emptyDoc); return err }},
		{"Distinct", coll, nil, func(mctx SessionContext) error { _, err := coll.Distinct(mctx, "field", emptyDoc); return err }},
		{"Find", coll, nil, func(mctx SessionContext) error { _, err := coll.Find(mctx, emptyDoc); return err }},
		{"FindOne", coll, nil, func(mctx SessionContext) error { res := coll.FindOne(mctx, emptyDoc); return res.err }},
		{"FindOneAndDelete", coll, nil, func(mctx SessionContext) error { res := coll.FindOneAndDelete(mctx, emptyDoc); return res.err }},
		{"FindOneAndReplace", coll, nil, func(mctx SessionContext) error {
			res := coll.FindOneAndReplace(mctx, emptyDoc, emptyDoc)
			return res.err
		}},
		{"FindOneAndUpdate", coll, nil, func(mctx SessionContext) error {
			res := coll.FindOneAndUpdate(mctx, emptyDoc, updateDoc)
			return res.err
		}},
		{"DropCollection", coll, nil, func(mctx SessionContext) error { err := coll.Drop(mctx); return err }},
		{"DropDatabase", coll, nil, func(mctx SessionContext) error { err := db.Drop(mctx); return err }},
		{"ListCollections", coll, nil, func(mctx SessionContext) error { _, err := db.ListCollections(mctx, emptyDoc); return err }},
		{"ListDatabases", coll, nil, func(mctx SessionContext) error { _, err := client.ListDatabases(mctx, emptyDoc); return err }},
		{"CreateOneIndex", coll, nil, func(mctx SessionContext) error { _, err := iv.CreateOne(mctx, fooIndex); return err }},
		{"CreateManyIndexes", coll, nil, func(mctx SessionContext) error { _, err := iv.CreateMany(mctx, manyIndexes); return err }},
		{"DropOneIndex", coll, &iv, func(mctx SessionContext) error { _, err := iv.DropOne(mctx, "barIndex"); return err }},
		{"DropAllIndexes", coll, nil, func(mctx SessionContext) error { _, err := iv.DropAll(mctx); return err }},
		{"ListIndexes", coll, nil, func(mctx SessionContext) error { _, err := iv.List(mctx); return err }},
	}

	return client, db, coll, functions
}

func getClusterTime(clusterTime *bson.Document) (uint32, uint32) {
	if clusterTime == nil {
		fmt.Println("is nil")
		return 0, 0
	}

	clusterTimeVal, err := clusterTime.LookupErr("$clusterTime")
	if err != nil {
		fmt.Println("could not find $clusterTime")
		return 0, 0
	}

	timestampVal, err := clusterTimeVal.MutableDocument().LookupErr("clusterTime")
	if err != nil {
		fmt.Println("could not find clusterTime")
		return 0, 0
	}

	return timestampVal.Timestamp()
}

func getOptValues(opts []interface{}) []reflect.Value {
	valOpts := make([]reflect.Value, 0, len(opts))
	for _, opt := range opts {
		valOpts = append(valOpts, reflect.ValueOf(opt))
	}

	return valOpts
}

func createMonitoredTopology(t *testing.T, clock *session.ClusterClock, monitor *event.CommandMonitor) *topology.Topology {
	if sessionsMonitoredTop != nil {
		return sessionsMonitoredTop // don't create the same topology twice
	}

	cs := testutil.ConnString(t)
	cs.HeartbeatInterval = time.Hour
	cs.HeartbeatIntervalSet = true

	opts := []topology.Option{
		topology.WithConnString(func(connstring.ConnString) connstring.ConnString { return cs }),
		topology.WithServerOptions(func(opts ...topology.ServerOption) []topology.ServerOption {
			return append(
				opts,
				topology.WithConnectionOptions(func(opts ...connection.Option) []connection.Option {
					return append(
						opts,
						connection.WithMonitor(func(*event.CommandMonitor) *event.CommandMonitor {
							return monitor
						}),
					)
				}),
				topology.WithClock(func(c *session.ClusterClock) *session.ClusterClock {
					return clock
				}),
			)
		}),
	}

	sessionsMonitoredTop, err := topology.New(opts...)
	if err != nil {
		t.Fatal(err)
	}

	err = sessionsMonitoredTop.Connect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	s, err := sessionsMonitoredTop.SelectServer(context.Background(), description.WriteSelector())
	if err != nil {
		t.Fatal(err)
	}

	c, err := s.Connection(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	_, err = (&command.Write{
		DB:      testutil.DBName(t),
		Command: bson.NewDocument(bson.EC.Int32("dropDatabase", 1)),
	}).RoundTrip(context.Background(), s.SelectedDescription(), c)
	if err != nil {
		t.Fatal(err)
	}

	return sessionsMonitoredTop
}

func createSessionsMonitoredClient(t *testing.T, monitor *event.CommandMonitor) *Client {
	clock := &session.ClusterClock{}

	c := &Client{
		topology:       createMonitoredTopology(t, clock, monitor),
		connString:     testutil.ConnString(t),
		readPreference: readpref.Primary(),
		readConcern:    readconcern.Local(),
		clock:          clock,
	}

	subscription, err := c.topology.Subscribe()
	testhelpers.RequireNil(t, err, "error subscribing to topology: %s", err)
	c.topology.SessionPool = session.NewPool(subscription.C)

	return c
}

func sessionIDsEqual(t *testing.T, sessionID1 *bson.Document, sessionID2 *bson.Document) bool {
	firstID, err := sessionID1.LookupErr("id")
	testhelpers.RequireNil(t, err, "error extracting ID 1: %s", err)

	secondID, err := sessionID2.LookupErr("id")
	testhelpers.RequireNil(t, err, "error extracting ID 2: %s", err)

	_, firstUUID := firstID.Binary()
	_, secondUUID := secondID.Binary()

	return reflect.DeepEqual(firstUUID, secondUUID)
}

// skip if the topology doesn't support sessions
func skipInvalidTopology(t *testing.T) {
	if os.Getenv("TOPOLOGY") == "server" {
		t.Skip("skipping for non-session supporting topology")
	}
}

func validSessionsVersion(t *testing.T, db *Database) bool {
	serverVersionStr, err := getServerVersion(db)
	testhelpers.RequireNil(t, err, "error getting server version: %s", err)
	return compareVersions(t, serverVersionStr, "3.6") >= 0
}

func getReturnError(returnVals []reflect.Value) error {
	errVal := returnVals[len(returnVals)-1]
	switch converted := errVal.Interface().(type) {
	case error:
		return converted
	case *DocumentResult:
		return converted.err
	default:
		return nil
	}
}

func getSessionUUID(t *testing.T, cmd *bson.Document) []byte {
	lsid, err := cmd.LookupErr("lsid")
	testhelpers.RequireNil(t, err, "key lsid not found in command")
	sessID, err := lsid.MutableDocument().LookupErr("id")
	testhelpers.RequireNil(t, err, "key id not found in lsid doc")

	_, data := sessID.Binary()
	return data
}

func getTestName(t *testing.T) string {
	fullName := t.Name()
	s := strings.Split(fullName, "/")
	return s[len(s)-1]
}

func verifySessionsReturned(t *testing.T, client *Client) {
	checkedOut := client.topology.SessionPool.CheckedOut()
	if checkedOut != 0 {
		t.Fatalf("%d sessions not returned for %s", checkedOut, t.Name())
	}
}

func checkLsidIncluded(t *testing.T, shouldInclude bool) {
	testhelpers.RequireNotNil(t, sessionStarted, "started event was nil")
	_, err := sessionStarted.Command.LookupErr("lsid")

	if shouldInclude {
		testhelpers.RequireNil(t, err, "key lsid not found in command for test %s", t.Name())
	} else {
		testhelpers.RequireNotNil(t, err, "key lsid found in command for test %s", t.Name())
	}
}

func checkUnbundle(t *testing.T, s1 *session.Client, s2 *sessionImpl, cid uuid.UUID, err error) {
	testhelpers.RequireNil(t, err, "Unexpected error unbundling: %s", err)
	if s1.ClientID != cid {
		t.Fatalf("expected client ID: %s, received client ID: %s", s1.ClientID, cid)
	}
	if s1.SessionID != s2.SessionID {
		t.Fatalf("expected session ID: %s, received session ID: %s", s1.SessionID, s2.SessionID)
	}
}

func drainHelper(c Cursor) {
	for c.Next(ctx) {
	}
}

func drainCursor(returnVals []reflect.Value) {
	if c, ok := returnVals[0].Interface().(Cursor); ok {
		drainHelper(c)
	}
}

func testCheckedOut(t *testing.T, client *Client, expected int) {
	actual := client.topology.SessionPool.CheckedOut()
	if actual != expected {
		t.Fatalf("checked out mismatch. expected %d got %d", expected, actual)
	}
}

func TestSessions(t *testing.T) {
	t.Run("TestPoolLifo", func(t *testing.T) {
		skipIfBelow36(t) // otherwise no session timeout is given and sessions auto expire

		client := createTestClient(t)
		defer verifySessionsReturned(t, client)

		aSess, err := client.StartSession()
		testhelpers.RequireNil(t, err, "error starting session a: %s", err)
		bSess, err := client.StartSession()
		testhelpers.RequireNil(t, err, "error starting session b: %s", err)
		a := aSess.(*sessionImpl)
		b := bSess.(*sessionImpl)

		a.EndSession(ctx)
		b.EndSession(ctx)

		firstSess, err := client.StartSession()
		testhelpers.RequireNil(t, err, "error starting first session: %s", err)
		defer firstSess.EndSession(ctx)
		first := firstSess.(*sessionImpl)

		if !sessionIDsEqual(t, first.SessionID, b.SessionID) {
			t.Errorf("expected first session ID to be %#v. got %#v", first.SessionID, b.SessionID)
		}

		secondSess, err := client.StartSession()
		testhelpers.RequireNil(t, err, "error starting second session: %s", err)
		defer secondSess.EndSession(ctx)
		second := secondSess.(*sessionImpl)

		if !sessionIDsEqual(t, second.SessionID, a.SessionID) {
			t.Errorf("expected second session ID to be %#v. got %#v", second.SessionID, a.SessionID)
		}
	})

	t.Run("TestClusterTime", func(t *testing.T) {
		// Test to see if $clusterTime is included in commands

		skipInvalidTopology(t)

		client := createSessionsMonitoredClient(t, sessionsMonitor)
		db := client.Database("SessionsTestClusterTime")
		err := db.Drop(ctx)
		testhelpers.RequireNil(t, err, "error dropping database: %s", err)

		coll := db.Collection("SessionsTestClusterTimeColl")
		serverStatusDoc := bson.NewDocument(bson.EC.Int32("serverStatus", 1))

		functions := []struct {
			name    string
			f       reflect.Value
			params1 []interface{}
			params2 []interface{}
		}{
			{"ServerStatus", reflect.ValueOf(db.RunCommand), []interface{}{ctx, serverStatusDoc}, []interface{}{ctx, serverStatusDoc}},
			{"InsertOne", reflect.ValueOf(coll.InsertOne), []interface{}{ctx, doc}, []interface{}{ctx, doc2}},
			{"Aggregate", reflect.ValueOf(coll.Aggregate), []interface{}{ctx, emptyDoc}, []interface{}{ctx, emptyDoc}},
			{"Find", reflect.ValueOf(coll.Find), []interface{}{ctx, emptyDoc}, []interface{}{ctx, emptyDoc}},
		}

		validVersion := validSessionsVersion(t, db)
		for _, tc := range functions {
			t.Run(tc.name, func(t *testing.T) {
				returnVals := tc.f.Call(getOptValues(tc.params1))
				defer verifySessionsReturned(t, client)
				defer drainCursor(returnVals)

				err := getReturnError(returnVals)
				testhelpers.RequireNil(t, err, "err running %s: %s", tc.name, err)

				testhelpers.RequireNotNil(t, sessionStarted, "started event was nil")
				_, err = sessionStarted.Command.LookupErr("$clusterTime")
				if validVersion {
					testhelpers.RequireNil(t, err, "key $clusterTime not found in first command for %s", tc.name)
				} else {
					testhelpers.RequireNotNil(t, err, "key $clusterTime found in first command for %s with version <3.6", tc.name)
					return // don't run rest of test because cluster times don't apply
				}

				// get ct from reply
				testhelpers.RequireNotNil(t, sessionSucceeded, "succeeded event was nil")
				replyCtVal, err := sessionSucceeded.Reply.LookupErr("$clusterTime")
				testhelpers.RequireNil(t, err, "key $clusterTime not found in reply")

				returnVals = tc.f.Call(getOptValues(tc.params2))
				err = getReturnError(returnVals)
				testhelpers.RequireNil(t, err, "err running %s: %s", tc.name, err)

				testhelpers.RequireNotNil(t, sessionStarted, "second started event was nil")
				nextCtVal, err := sessionStarted.Command.LookupErr("$clusterTime")
				testhelpers.RequireNil(t, err, "key $clusterTime not found in first command for %s", tc.name)

				epoch1, ord1 := getClusterTime(bson.NewDocument(bson.EC.SubDocument("$clusterTime", replyCtVal.MutableDocument())))
				epoch2, ord2 := getClusterTime(bson.NewDocument(bson.EC.SubDocument("$clusterTime", nextCtVal.MutableDocument())))

				if epoch1 == 0 {
					t.Fatal("epoch1 is 0")
				} else if epoch2 == 0 {
					t.Fatal("epoch2 is 0")
				}

				if epoch1 != epoch2 {
					t.Fatalf("epoch mismatch. epoch1 = %d, epoch2 = %d", epoch1, epoch2)
				}

				if ord1 != ord2 {
					t.Fatalf("ord mismatch. ord1 = %d, ord2 = %d", ord1, ord2)
				}
			})
		}
	})

	t.Run("TestExplicitImplicitSessionArgs", func(t *testing.T) {
		// Test to see if lsid is included in commands with explicit and implicit sessions

		skipInvalidTopology(t)
		skipIfBelow36(t)

		name := getTestName(t)
		client, db, _, funcMap := createFuncMap(t, name+"DB", name+"Coll", true)

		for _, tc := range funcMap {
			t.Run(tc.name, func(t *testing.T) {
				defer verifySessionsReturned(t, client)

				s, err := client.StartSession()
				testhelpers.RequireNil(t, err, "error creating session for %s: %s", tc.name, err)
				defer s.EndSession(ctx)
				sess := s.(*sessionImpl)

				// check to see if lsid included with explicit session
				err = WithSession(ctx, sess, tc.f)
				testhelpers.RequireNil(t, err, "error running %s: %s", tc.name, err)

				_, sessID := sess.SessionID.Lookup("id").Binary()
				if !bytes.Equal(getSessionUUID(t, sessionStarted.Command), sessID) {
					t.Fatal("included UUID does not match session UUID")
				}

				// can't insert same document again
				if tc.name == "InsertOne" {
					tc.f = func(mctx SessionContext) error {
						_, err := tc.coll.InsertOne(mctx, bson.NewDocument(bson.EC.Int32("InsertOneNewDoc", 1)))
						return err
					}
				} else if tc.name == "InsertMany" {
					tc.f = func(mctx SessionContext) error {
						_, err := tc.coll.InsertMany(mctx, []interface{}{bson.NewDocument(bson.EC.Int32("InsertManyNewDoc", 2))})
						return err
					}
				} else if tc.name == "DropOneIndex" {
					tc.f = func(mctx SessionContext) error {
						_, err := tc.iv.DropOne(mctx, "bazIndex")
						return err
					}
				}

				err = WithSession(ctx, sess, tc.f)
				testhelpers.RequireNil(t, err, "error running %s: %s", tc.name, err)
				//defer drainCursor(returnVals)

				// check to see if lsid included with implicit session
				shouldInclude := validSessionsVersion(t, db)
				checkLsidIncluded(t, shouldInclude)
			})
		}
	})

	t.Run("TestSessionArgsForClient", func(t *testing.T) {
		// test to make sure a session can only be used in commands associated with the client that created it

		client, _, _, funcMap := createFuncMap(t, "sessionArgsDb", "sessionArgsColl", false)

		client2 := createTestClient(t)

		for _, tc := range funcMap {
			t.Run(tc.name, func(t *testing.T) {
				defer verifySessionsReturned(t, client)
				defer verifySessionsReturned(t, client2)

				sess, err := client2.StartSession()
				testhelpers.RequireNil(t, err, "error starting session: %s", err)
				defer sess.EndSession(ctx)

				err = WithSession(ctx, sess, tc.f)
				testhelpers.RequireNotNil(t, err, "expected err for %s got nil", tc.name)

				if err != ErrWrongClient {
					t.Errorf("expected error using wrong client for function %s; got: %s", reflect.ValueOf(tc.f).String(), err)
				}
			})
		}
	})

	t.Run("TestEndSession", func(t *testing.T) {
		// test to make sure that an ended session cannot be used in commands

		client, _, _, funcMap := createFuncMap(t, "endSessionsDb", "endSessionsDb", false)

		for _, tc := range funcMap {
			t.Run(tc.name, func(t *testing.T) {
				defer verifySessionsReturned(t, client)

				sess, err := client.StartSession()
				testhelpers.RequireNil(t, err, "error starting session: %s", err)
				sess.EndSession(ctx)

				err = WithSession(ctx, sess, tc.f)
				testhelpers.RequireNotNil(t, err, "expected error for %s got nil", tc.name)

				if err != session.ErrSessionEnded {
					t.Errorf("expected error using ended session for function %s; got: %s", reflect.ValueOf(tc.f).String(), err)
				}
			})
		}
	})

	t.Run("TestImplicitSessionReturned", func(t *testing.T) {
		// test to make sure implicit sessions are returned to the server session pool

		skipInvalidTopology(t)
		skipIfBelow36(t)

		client := createSessionsMonitoredClient(t, sessionsMonitor)
		defer verifySessionsReturned(t, client)

		db := client.Database("ImplicitSessionReturnedDB")
		err := db.Drop(ctx)
		testhelpers.RequireNil(t, err, "error dropping database: %s", err)
		coll := db.Collection("ImplicitSessionReturnedColl")

		_, err = coll.InsertOne(ctx, bson.NewDocument(bson.EC.Int32("x", 1)))
		testhelpers.RequireNil(t, err, "error running insert: %s", err)
		_, err = coll.InsertOne(ctx, bson.NewDocument(bson.EC.Int32("y", 2)))
		testhelpers.RequireNil(t, err, "error running insert: %s", err)

		cur, err := coll.Find(ctx, emptyDoc) // should use implicit session returned by InsertOne commands
		testhelpers.RequireNil(t, err, "error running find: %s", err)

		testhelpers.RequireNotNil(t, sessionStarted, "started command was nil")
		findUUID := getSessionUUID(t, sessionStarted.Command)

		cur.Next(ctx)
		testCheckedOut(t, client, 0)

		_, err = coll.DeleteOne(ctx, emptyDoc)
		testhelpers.RequireNil(t, err, "error running delete: %s", err)
		if sessionStarted.CommandName != "delete" {
			t.Fatal("delete command not monitored")
		}
		deleteUUID := getSessionUUID(t, sessionStarted.Command)

		// check to see if Delete used same implicit session as Find
		if !bytes.Equal(findUUID, deleteUUID) {
			t.Fatal("uuid mismatch")
		}
	})

	t.Run("TestImplicitSessionReturnedFromGetMore", func(t *testing.T) {
		// test to make sure that a cursor returns a session to the session pool after running the final getMore operation

		skipInvalidTopology(t)
		skipIfBelow36(t)

		client := createTestClient(t)

		db := client.Database("ImplicitSessionReturnedGMDB")
		err := db.Drop(ctx)
		testhelpers.RequireNil(t, err, "error dropping database: %s", err)
		coll := db.Collection("ImplicitSessionReturnedGMColl")

		docs := []interface{}{
			bson.NewDocument(bson.EC.Int32("a", 1)),
			bson.NewDocument(bson.EC.Int32("a", 2)),
			bson.NewDocument(bson.EC.Int32("a", 3)),
			bson.NewDocument(bson.EC.Int32("a", 4)),
			bson.NewDocument(bson.EC.Int32("a", 5)),
		}
		_, err = coll.InsertMany(ctx, docs) // pool should have 1 session
		require.Nil(t, err, "Error on insert")

		cur, err := coll.Find(ctx, emptyDoc, findopt.BatchSize(3))
		require.Nil(t, err, "Error on find")

		testCheckedOut(t, client, 1)

		cur.Next(ctx)
		cur.Next(ctx)
		cur.Next(ctx)
		cur.Next(ctx)

		verifySessionsReturned(t, client)
	})

	t.Run("TestFindAndGetMoreSessionIDs", func(t *testing.T) {
		skipInvalidTopology(t)
		skipIfBelow36(t)

		primary := readpref.Primary()
		second := readpref.Secondary()
		primPref := readpref.PrimaryPreferred()
		secondPref := readpref.SecondaryPreferred()
		rsTop := "replica_set"
		shardedTop := "sharded_cluster"
		wcMajority := writeconcern.New(writeconcern.WMajority())
		client := createSessionsMonitoredClient(t, sessionsMonitor)
		db := client.Database("TestFindAndGetMoreSessionIDsDB")
		coll := db.Collection("TestFindAndGetMoreSessionIDsColl", collectionopt.WriteConcern(wcMajority),
			collectionopt.ReadConcern(readconcern.Majority()))
		docs := []interface{}{
			bson.NewDocument(bson.EC.Int32("a", 1)),
			bson.NewDocument(bson.EC.Int32("a", 2)),
			bson.NewDocument(bson.EC.Int32("a", 3)),
		}

		for i, doc := range docs {
			_, err := coll.InsertOne(ctx, doc)
			testhelpers.RequireNil(t, err, "err inserting doc %d: %s", i, err)
		}

		var tests = []struct {
			name        string
			expectedTop string
			rp          *readpref.ReadPref
		}{
			{"rsPrimary", rsTop, primary},
			{"rsSecondary", rsTop, second},
			{"rsPrimaryPref", rsTop, primPref},
			{"rsSecondaryPref", rsTop, secondPref},
			{"shardedPrimary", shardedTop, primary},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if os.Getenv("TOPOLOGY") != tc.expectedTop {
					t.Skip()
				}

				coll.readPreference = tc.rp
				cur, err := coll.Find(ctx, emptyDoc, findopt.BatchSize(2))
				testhelpers.RequireNil(t, err, "error running find: %s", err)

				testhelpers.RequireNotNil(t, sessionStarted, "no started command registered for find")
				if sessionStarted.CommandName != "find" {
					t.Fatalf("started command %s was not a find command", sessionStarted.CommandName)
				}
				findUUID := getSessionUUID(t, sessionStarted.Command)

				for i := 0; i < 3; i++ {
					if !cur.Next(ctx) {
						t.Fatalf("cursor Next() returned false on iteration %d", i)
					}
				}

				if sessionStarted.CommandName != "getMore" {
					t.Fatalf("started command %s was not a getMore command", sessionStarted.CommandName)
				}

				getMoreUUID := getSessionUUID(t, sessionStarted.Command)
				if !bytes.Equal(findUUID, getMoreUUID) {
					t.Fatalf("uuid mismatch for find and getMore")
				}
			})
		}
	})

	//t.Run("TestSessionUnbundling", func(t *testing.T) {
	//	client := createTestClient(t)
	//	sess1, err := client.StartSession()
	//	sess2, err := client.StartSession()
	//	testhelpers.RequireNil(t, err, "Unexpected error starting session: %s", err)
	//
	//	t.Run("TestAggregateOpt", func(t *testing.T) {
	//		_, s, err := aggregateopt.BundleAggregate(sess1, aggregateopt.BundleAggregate(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//	})
	//
	//	t.Run("TestChangeStreamOpt", func(t *testing.T) {
	//		_, s, err := changestreamopt.BundleChangeStream(sess1, changestreamopt.BundleChangeStream(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//	})
	//
	//	t.Run("TestCountOpt", func(t *testing.T) {
	//		_, s, err := countopt.BundleCount(sess1, countopt.BundleCount(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//	})
	//
	//	t.Run("TestCountOpt", func(t *testing.T) {
	//		_, s, err := deleteopt.BundleDelete(sess1, deleteopt.BundleDelete(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//	})
	//
	//	t.Run("TestDistinctOpt", func(t *testing.T) {
	//		_, s, err := distinctopt.BundleDistinct(sess1, distinctopt.BundleDistinct(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//	})
	//
	//	t.Run("TestFindOpt", func(t *testing.T) {
	//		_, s, err := findopt.BundleOne(sess1, findopt.BundleOne(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//
	//		_, s, err = findopt.BundleFind(sess1, findopt.BundleFind(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//
	//		_, s, err = findopt.BundleDeleteOne(sess1, findopt.BundleDeleteOne(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//
	//		_, s, err = findopt.BundleReplaceOne(sess1, findopt.BundleReplaceOne(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//
	//		_, s, err = findopt.BundleUpdateOne(sess1, findopt.BundleUpdateOne(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//	})
	//
	//	t.Run("TestIndexOpt", func(t *testing.T) {
	//		_, s, err := indexopt.BundleCreate(sess1, indexopt.BundleCreate(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//
	//		_, s, err = indexopt.BundleDrop(sess1, indexopt.BundleDrop(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//
	//		_, s, err = indexopt.BundleList(sess1, indexopt.BundleList(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//	})
	//
	//	t.Run("TestInsertOpt", func(t *testing.T) {
	//		_, s, err := insertopt.BundleMany(sess1, insertopt.BundleMany(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//
	//		_, s, err = insertopt.BundleOne(sess1, insertopt.BundleOne(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//	})
	//
	//	t.Run("TestListCollOpt", func(t *testing.T) {
	//		_, s, err := listcollectionopt.BundleListCollections(sess1, listcollectionopt.BundleListCollections(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//	})
	//
	//	t.Run("TestListDBOpt", func(t *testing.T) {
	//		_, s, err := listdbopt.BundleListDatabases(sess1, listdbopt.BundleListDatabases(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//	})
	//
	//	t.Run("TestReplaceOpt", func(t *testing.T) {
	//		_, s, err := replaceopt.BundleReplace(sess1, replaceopt.BundleReplace(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//	})
	//
	//	t.Run("TestUpdateOpt", func(t *testing.T) {
	//		_, s, err := updateopt.BundleUpdate(sess1, updateopt.BundleUpdate(sess2)).Unbundle(true)
	//		checkUnbundle(t, s, sess2, client.id, err)
	//	})
	//})
}
