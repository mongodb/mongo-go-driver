package mongo

import (
	"context"
	"reflect"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"github.com/stretchr/testify/require"
)

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
	// TODO(GODRIVER-19) - use APM to monitor commands and responses
	t.Skip("skipping for lack of APM")
}

func TestExplicitImplicitSessionArgs(t *testing.T) {
	// TODO(GODRIVER-19) - use APM to monitor commands and responses
	t.Skip("skipping for lack of APM")
}

var ctx = context.Background()
var emptyDoc = bson.NewDocument()
var updateDoc = bson.NewDocument(
	bson.EC.String("$query", "test"),
)
var doc = bson.NewDocument(
	bson.EC.Int32("x", 1),
)

type CollFunction struct {
	name string
	f    reflect.Value
	opts []interface{}
}

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
