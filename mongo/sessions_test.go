package mongo

import (
	"context"
	"reflect"
	"testing"

	"fmt"
	"os"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/connection"
	"github.com/mongodb/mongo-go-driver/core/connstring"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/event"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"github.com/stretchr/testify/require"
)

var sessionStarted *event.CommandStartedEvent
var sessionSucceeded *event.CommandSucceededEvent
var sessionsMonitoredTop *topology.Topology

var sessionsMonitor = &event.CommandMonitor{
	Started: func(cse *event.CommandStartedEvent) {
		sessionStarted = cse
	},
	Succeeded: func(cse *event.CommandSucceededEvent) {
		sessionSucceeded = cse
	},
}

type CollFunction struct {
	name string
	f    reflect.Value
	opts []interface{}
}

var ctx = context.Background()
var emptyDoc = bson.NewDocument()
var updateDoc = bson.NewDocument(
	bson.EC.String("$query", "test"),
)
var doc = bson.NewDocument(
	bson.EC.Int32("x", 1),
)
var secondDoc = bson.NewDocument(
	bson.EC.Int32("y", 1),
)

func createFuncMap(t *testing.T, dbName string, collName string) (*Client, *Database, *Collection, []CollFunction) {
	client := createTestClient(t)

	db := client.Database(dbName)
	coll := db.Collection(collName)

	functions := []CollFunction{
		{"InsertOne", reflect.ValueOf(coll.InsertOne), []interface{}{ctx, doc}},
		{"InsertMany", reflect.ValueOf(coll.InsertMany), []interface{}{ctx, []interface{}{doc}}},
		{"DeleteOne", reflect.ValueOf(coll.DeleteOne), []interface{}{ctx, emptyDoc}},
		{"DeleteMany", reflect.ValueOf(coll.DeleteMany), []interface{}{ctx, emptyDoc}},
		{"UpdateOne", reflect.ValueOf(coll.UpdateOne), []interface{}{ctx, emptyDoc, updateDoc}},
		{"UpdateMany", reflect.ValueOf(coll.UpdateMany), []interface{}{ctx, emptyDoc, updateDoc}},
		{"ReplaceOne", reflect.ValueOf(coll.ReplaceOne), []interface{}{ctx, emptyDoc, emptyDoc}},
		{"Aggregate", reflect.ValueOf(coll.Aggregate), []interface{}{ctx, emptyDoc}},
		{"Count", reflect.ValueOf(coll.Count), []interface{}{ctx, emptyDoc}},
		{"Distinct", reflect.ValueOf(coll.Distinct), []interface{}{ctx, "field", emptyDoc}},
		{"Find", reflect.ValueOf(coll.Find), []interface{}{ctx, emptyDoc}},
		{"FindOne", reflect.ValueOf(coll.FindOne), []interface{}{ctx, emptyDoc}},
		{"FindOneAndDelete", reflect.ValueOf(coll.FindOneAndDelete), []interface{}{ctx, emptyDoc}},
		{"FindOneAndReplace", reflect.ValueOf(coll.FindOneAndReplace), []interface{}{ctx, emptyDoc, emptyDoc}},
		{"FindOneAndUpdate", reflect.ValueOf(coll.FindOneAndUpdate), []interface{}{ctx, emptyDoc, updateDoc}},
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

func createSessionsMonitoredTopology(t *testing.T, clock *session.ClusterClock) *topology.Topology {
	if sessionsMonitoredTop != nil {
		return sessionsMonitoredTop
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
							return sessionsMonitor
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

func createSessionsMonitoredClient(t *testing.T) *Client {
	clock := &session.ClusterClock{}

	c := &Client{
		topology:       createSessionsMonitoredTopology(t, clock),
		connString:     testutil.ConnString(t),
		readPreference: readpref.Primary(),
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

func TestPoolLifo(t *testing.T) {
	skipIfBelow36(t) // otherwise no sessiontimeout is given and sessions auto expire

	client := createTestClient(t)

	a, err := client.StartSession()
	testhelpers.RequireNil(t, err, "error starting session a: %s", err)
	b, err := client.StartSession()
	testhelpers.RequireNil(t, err, "error starting session b: %s", err)

	a.EndSession()
	b.EndSession()

	first, err := client.StartSession()
	testhelpers.RequireNil(t, err, "error starting first session: %s", err)

	if !sessionIDsEqual(t, first.SessionID, b.SessionID) {
		t.Errorf("expected first session ID to be %#v. got %#v", first.SessionID, b.SessionID)
	}

	second, err := client.StartSession()
	if !sessionIDsEqual(t, second.SessionID, a.SessionID) {
		t.Errorf("expected second session ID to be %#v. got %#v", second.SessionID, a.SessionID)
	}
}

func TestClusterTime(t *testing.T) {
	if os.Getenv("TOPOLOGY") == "server" {
		t.Skip("skipping for non-session supporting topology")
	}

	client := createSessionsMonitoredClient(t)
	db := client.Database("SessionsTestClusterTime")
	serverVersionStr, err := getServerVersion(db)
	compRes := compareVersions(t, serverVersionStr, "3.6")
	validVersion := compRes >= 0

	testhelpers.RequireNil(t, err, "error getting server version: %s", err)
	coll := db.Collection("SessionsTestClusterTimeColl")
	serverStatusDoc := bson.NewDocument(bson.EC.Int32("serverStatus", 1))

	functions := []struct {
		name    string
		f       reflect.Value
		params1 []interface{}
		params2 []interface{}
	}{
		{"ServerStatus", reflect.ValueOf(db.RunCommand), []interface{}{ctx, serverStatusDoc}, []interface{}{ctx, serverStatusDoc}},
		{"InsertOne", reflect.ValueOf(coll.InsertOne), []interface{}{ctx, doc}, []interface{}{ctx, secondDoc}},
		{"Aggregate", reflect.ValueOf(coll.Aggregate), []interface{}{ctx, emptyDoc}, []interface{}{ctx, emptyDoc}},
		{"Find", reflect.ValueOf(coll.Find), []interface{}{ctx, emptyDoc}, []interface{}{ctx, emptyDoc}},
	}

	for _, tc := range functions {
		// TODO: nil checks for events

		t.Run(tc.name, func(t *testing.T) {
			returnVals := tc.f.Call(getOptValues(tc.params1))
			errVal := returnVals[len(returnVals)-1]
			require.Nil(t, errVal.Interface(), "got error running %s: %s", tc.name, errVal)

			_, err := sessionStarted.Command.LookupErr("$clusterTime")
			if validVersion {
				testhelpers.RequireNil(t, err, "key $clusterTime not found in first command for %s", tc.name)
			} else {
				testhelpers.RequireNotNil(t, err, "key $clusterTime found in first command for %s with version <3.6", tc.name)
				return // don't run rest of test because cluster times don't apply
			}

			// get ct from reply
			replyCtVal, err := sessionSucceeded.Reply.LookupErr("$clusterTime")
			testhelpers.RequireNil(t, err, "key $clusterTime not found in reply")

			returnVals = tc.f.Call(getOptValues(tc.params2))
			errVal = returnVals[len(returnVals)-1]
			require.Nil(t, errVal.Interface(), "got error running %s: %s", tc.name, errVal)

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
}

func TestExplicitImplicitSessionArgs(t *testing.T) {
	// TODO(GODRIVER-19) - use APM to monitor commands and responses
	t.Skip("skipping for lack of APM")
}

func TestSessionArgsForClient(t *testing.T) {
	_, _, _, funcMap := createFuncMap(t, "sessionArgsDb", "sessionArgsColl")

	client2 := createTestClient(t)
	sess, err := client2.StartSession()
	testhelpers.RequireNil(t, err, "error starting session: %s", err)

	for _, tc := range funcMap {
		t.Run(tc.name, func(t *testing.T) {
			opts := append(tc.opts, sess)
			valOpts := make([]reflect.Value, 0, len(opts))
			for _, opt := range opts {
				valOpts = append(valOpts, reflect.ValueOf(opt))
			}
			returnVals := tc.f.Call(valOpts)
			require.NotNil(t, returnVals[len(returnVals)-1], "expected error, received nil for function %s", tc.f.String())
			switch v := returnVals[len(returnVals)-1].Interface().(type) {
			case *DocumentResult:
				if v.err != ErrWrongClient {
					t.Errorf("expected wrong client error for function %s\nRecieved: %v", tc.f.String(), v.err)
				}
			case error:
				if v != ErrWrongClient {
					t.Errorf("expected wrong client error for function %s\nReceived: %v", tc.f.String(), v)
				}
			}
		})
	}
}

func TestEndSession(t *testing.T) {
	client, _, _, funcMap := createFuncMap(t, "endSessionsDb", "endSessionsDb")

	for _, tc := range funcMap {
		t.Run(tc.name, func(t *testing.T) {
			sess, err := client.StartSession()
			testhelpers.RequireNil(t, err, "error starting session: %s", err)
			sess.EndSession()

			opts := append(tc.opts, sess)
			valOpts := make([]reflect.Value, 0, len(opts))
			for _, opt := range opts {
				valOpts = append(valOpts, reflect.ValueOf(opt))
			}

			returnVals := tc.f.Call(valOpts)
			require.NotNil(t, returnVals[len(returnVals)-1], "expected error, received nil for function %s", tc.f.String())
			switch v := returnVals[len(returnVals)-1].Interface().(type) {
			case *DocumentResult:
				if v.err != session.ErrSessionEnded {
					t.Errorf("expected error using ended session for function %s\nRecieved: %v", tc.f.String(), v.err)
				}
			case error:
				if v != session.ErrSessionEnded {
					t.Errorf("expected error using ended session for function %s\nReceived: %v", tc.f.String(), v)
				}
			}
		})
	}
}

func TestImplicitSessionReturned(t *testing.T) {
	client := createTestClient(t)

	db := client.Database("ImplicitSessionReturnedDB")
	coll := db.Collection("ImplicitSessionReturnedColl")

	_, err := coll.InsertOne(ctx, bson.NewDocument(bson.EC.Int32("x", 1)))
	require.Nil(t, err, "Error on insert")
	_, err = coll.InsertOne(ctx, bson.NewDocument(bson.EC.Int32("y", 2)))
	require.Nil(t, err, "Error on insert")

	cur, err := coll.Find(ctx, emptyDoc)
	require.Nil(t, err, "Error on find")

	cur.Next(ctx)

	// TODO(GODRIVER-19) - use APM to monitor commands and responses
	t.Skip("skipping for lack of APM")

}

func TestImplicitSessionReturnedFromGetMore(t *testing.T) {
	client := createTestClient(t)

	db := client.Database("ImplicitSessionReturnedDB")
	coll := db.Collection("ImplicitSessionReturnedColl")

	docs := []interface{}{
		bson.NewDocument(bson.EC.Int32("a", 1)),
		bson.NewDocument(bson.EC.Int32("a", 2)),
		bson.NewDocument(bson.EC.Int32("a", 3)),
		bson.NewDocument(bson.EC.Int32("a", 4)),
		bson.NewDocument(bson.EC.Int32("a", 5)),
	}
	_, err := coll.InsertMany(ctx, docs)
	require.Nil(t, err, "Error on insert")

	cur, err := coll.Find(ctx, emptyDoc, findopt.BatchSize(3))
	require.Nil(t, err, "Error on find")

	cur.Next(ctx)
	cur.Next(ctx)
	cur.Next(ctx)
	cur.Next(ctx)

	// TODO(GODRIVER-19) - use APM to monitor commands and responses
	t.Skip("skipping for lack of APM")

}

func TestFindAndGetMoreSessionIDs(t *testing.T) {
	// TODO(GODRIVER-19) - use APM to monitor commands and responses
	t.Skip("skipping for lack of APM")

}
