package mongo

import (
	"context"
	"reflect"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

const URI = "mongodb://localhost:27017"

func sessionIdsEqual(t *testing.T, sessionId1 *bson.Document, sessionId2 *bson.Document) bool {
	firstId, err := sessionId1.LookupErr("id")
	testhelpers.RequireNil(t, err, "error extracting ID 1: %s", err)

	secondId, err := sessionId1.LookupErr("id")
	testhelpers.RequireNil(t, err, "error extracting ID 2: %s", err)

	_, firstUuid := firstId.Binary()
	_, secondUuid := secondId.Binary()

	return reflect.DeepEqual(firstUuid, secondUuid)
}

func TestPoolLifo(t *testing.T) {
	client, err := NewClient(URI)
	testhelpers.RequireNil(t, err, "error creating client: %s", err)

	a, err := client.StartSession()
	testhelpers.RequireNil(t, err, "error starting session a: %s", err)
	b, err := client.StartSession()
	testhelpers.RequireNil(t, err, "error starting session b: %s", err)

	a.EndSession()
	b.EndSession()

	first, err := client.StartSession()
	testhelpers.RequireNil(t, err, "error starting first session: %s", err)

	if !sessionIdsEqual(t, first.SessionID, b.SessionID) {
		t.Errorf("expected first session ID to be %#v. got %#v", first.SessionID, b.SessionID)
	}

	second, err := client.StartSession()
	if !sessionIdsEqual(t, second.SessionID, a.SessionID) {
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
var doc = bson.NewDocument(
	bson.EC.Int32("x", 1),
)

type CollFunction struct {
	name string
	f    reflect.Value
	opts []interface{}
}

func createFuncMap(t *testing.T, dbName string, collName string) (*Client, *Database, *Collection, []CollFunction) {
	client, err := NewClient(URI)
	testhelpers.RequireNil(t, err, "error creating client: %s", err)

	db := client.Database(dbName)
	coll := db.Collection(collName)

	functions := []CollFunction{
		{"InsertOne", reflect.ValueOf(coll.InsertOne), []interface{}{ctx, doc}},
		{"InsertMany", reflect.ValueOf(coll.InsertMany), []interface{}{ctx, []interface{}{doc}}},
		{"DeleteOne", reflect.ValueOf(coll.DeleteOne), []interface{}{ctx, emptyDoc}},
		{"DeleteMany", reflect.ValueOf(coll.DeleteMany), []interface{}{ctx, []interface{}{doc}}},
		{"UpdateOne", reflect.ValueOf(coll.UpdateOne), []interface{}{ctx, emptyDoc, emptyDoc}},
		{"UpdateMany", reflect.ValueOf(coll.UpdateMany), []interface{}{ctx, emptyDoc, emptyDoc}},
		{"ReplaceOne", reflect.ValueOf(coll.ReplaceOne), []interface{}{ctx, emptyDoc, emptyDoc}},
		{"Aggregate", reflect.ValueOf(coll.Aggregate), []interface{}{ctx, emptyDoc}},
		{"Count", reflect.ValueOf(coll.Count), []interface{}{ctx, emptyDoc}},
		{"Distinct", reflect.ValueOf(coll.Distinct), []interface{}{ctx, "field", emptyDoc}},
		{"Find", reflect.ValueOf(coll.Find), []interface{}{ctx, emptyDoc}},
		{"FindOne", reflect.ValueOf(coll.FindOne), []interface{}{ctx, emptyDoc}},
		{"FindOneAndDelete", reflect.ValueOf(coll.FindOneAndDelete), []interface{}{ctx, emptyDoc}},
		{"FindOneAndReplace", reflect.ValueOf(coll.FindOneAndReplace), []interface{}{ctx, emptyDoc, emptyDoc}},
		{"FindOneAndUpdate", reflect.ValueOf(coll.FindOneAndUpdate), []interface{}{ctx, emptyDoc, emptyDoc}},
	}

	return client, db, coll, functions
}

func TestSessionArgsForClient(t *testing.T) {
	_, _, _, funcMap := createFuncMap(t, "sessionArgsDb", "sessionArgsColl")

	client2, err := NewClient(URI)
	testhelpers.RequireNil(t, err, "error creating client2: %s", err)
	sess, err := client2.StartSession()
	testhelpers.RequireNil(t, err, "error starting session: %s", err)

	for _, tc := range funcMap {
		t.Run(tc.name, func(t *testing.T) {
			opts := append(tc.opts, sess)
			valOpts := make([]reflect.Value, len(tc.opts))
			for _, opt := range opts {
				valOpts = append(valOpts, reflect.ValueOf(opt))
			}

			returnVals := tc.f.Call(valOpts)
			if returnVals[len(returnVals)-1].IsNil() {
				t.Errorf("expected error for function %s. got nil", tc.f.String())
			}
		})
	}
}

func TestEndSession(t *testing.T) {

}
