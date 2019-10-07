// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/testutil"
	testhelpers "go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
	"go.mongodb.org/mongo-driver/x/mongo/driver/uuid"
)

// temporary file to hold test helpers as we are porting tests to the new integration testing framework.

var wcMajority = writeconcern.New(writeconcern.WMajority())

var doc1 = bsonx.Doc{
	{"x", bsonx.Int32(1)},
}

var sessionStarted *event.CommandStartedEvent
var sessionSucceeded *event.CommandSucceededEvent
var sessionsMonitoredTop *topology.Topology
var ctx = context.Background()
var emptyDoc = bsonx.Doc{}
var emptyArr = bsonx.Arr{}
var updateDoc = bsonx.Doc{{"$inc", bsonx.Document(bsonx.Doc{{"x", bsonx.Int32(1)}})}}
var doc = bsonx.Doc{{"x", bsonx.Int32(1)}}
var doc2 = bsonx.Doc{{"y", bsonx.Int32(1)}}

var fooIndex = IndexModel{
	Keys:    bsonx.Doc{{"foo", bsonx.Int32(-1)}},
	Options: options.Index().SetName("fooIndex"),
}

var barIndex = IndexModel{
	Keys:    bsonx.Doc{{"bar", bsonx.Int32(-1)}},
	Options: options.Index().SetName("barIndex"),
}

var bazIndex = IndexModel{
	Keys:    bsonx.Doc{{"baz", bsonx.Int32(-1)}},
	Options: options.Index().SetName("bazIndex"),
}

var sessionsMonitor = &event.CommandMonitor{
	Started: func(ctx context.Context, cse *event.CommandStartedEvent) {
		sessionStarted = cse
	},
	Succeeded: func(ctx context.Context, cse *event.CommandSucceededEvent) {
		sessionSucceeded = cse
	},
}

var startedChan = make(chan *event.CommandStartedEvent, 100)
var succeededChan = make(chan *event.CommandSucceededEvent, 100)
var failedChan = make(chan *event.CommandFailedEvent, 100)

var monitor = &event.CommandMonitor{
	Started: func(ctx context.Context, cse *event.CommandStartedEvent) {
		startedChan <- cse
	},
	Succeeded: func(ctx context.Context, cse *event.CommandSucceededEvent) {
		succeededChan <- cse
	},
	Failed: func(ctx context.Context, cfe *event.CommandFailedEvent) {
		failedChan <- cfe
	},
}

type CollFunction struct {
	name string
	coll *Collection
	iv   *IndexView
	f    func(SessionContext) error
}

func createTestClient(t *testing.T) *Client {
	id, _ := uuid.New()
	return &Client{
		id:             id,
		deployment:     testutil.Topology(t),
		connString:     testutil.ConnString(t),
		readPreference: readpref.Primary(),
		clock:          &session.ClusterClock{},
		registry:       bson.DefaultRegistry,
		retryWrites:    true,
		sessionPool:    testutil.SessionPool(),
	}
}

func createTestClientWithConnstring(t *testing.T, cs connstring.ConnString) *Client {
	id, _ := uuid.New()
	return &Client{
		id:             id,
		deployment:     testutil.TopologyWithConnString(t, cs),
		connString:     cs,
		readPreference: readpref.Primary(),
		clock:          &session.ClusterClock{},
		registry:       bson.DefaultRegistry,
	}
}

func createTestDatabase(t *testing.T, name *string, opts ...*options.DatabaseOptions) *Database {
	if name == nil {
		db := testutil.DBName(t)
		name = &db
	}

	client := createTestClient(t)

	dbOpts := []*options.DatabaseOptions{options.Database().SetWriteConcern(writeconcern.New(writeconcern.WMajority()))}
	dbOpts = append(dbOpts, opts...)
	return client.Database(*name, dbOpts...)
}

func createTestCollection(t *testing.T, dbName *string, collName *string, opts ...*options.CollectionOptions) *Collection {
	if collName == nil {
		coll := testutil.ColName(t)
		collName = &coll
	}

	db := createTestDatabase(t, dbName)
	db.RunCommand(
		context.Background(),
		bsonx.Doc{{"create", bsonx.String(*collName)}},
	)

	collOpts := []*options.CollectionOptions{options.Collection().SetWriteConcern(writeconcern.New(writeconcern.WMajority()))}
	collOpts = append(collOpts, opts...)
	return db.Collection(*collName, collOpts...)
}

func skipIfBelow34(t *testing.T, db *Database) {
	versionStr, err := getServerVersion(db)
	if err != nil {
		t.Fatalf("error getting server version: %s", err)
	}
	if compareVersions(t, versionStr, "3.4") < 0 {
		t.Skip("skipping collation test for server version < 3.4")
	}
}

func initCollection(t *testing.T, coll *Collection) {
	docs := []interface{}{}
	var i int32
	for i = 1; i <= 5; i++ {
		docs = append(docs, bsonx.Doc{{"x", bsonx.Int32(i)}})
	}

	_, err := coll.InsertMany(ctx, docs)
	require.Nil(t, err)
}

func skipIfBelow36(t *testing.T) {
	serverVersion, err := getServerVersion(createTestDatabase(t, nil))
	require.NoError(t, err, "unable to get server version of database")

	if compareVersions(t, serverVersion, "3.6") < 0 {
		t.Skip()
	}
}

func skipIfBelow32(t *testing.T) {
	serverVersion, err := getServerVersion(createTestDatabase(t, nil))
	require.NoError(t, err)

	if compareVersions(t, serverVersion, "3.2") < 0 {
		t.Skip()
	}
}

func noerr(t *testing.T, err error) {
	if err != nil {
		t.Helper()
		t.Errorf("Unexpected error: (%T)%v", err, err)
		t.FailNow()
	}
}

type runOn struct {
	MinServerVersion string   `json:"minServerVersion" bson:"minServerVersion"`
	MaxServerVersion string   `json:"maxServerVersion" bson:"maxServerVersion"`
	Topology         []string `json:"topology" bson:"topology"`
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
		Name   string `json:"codeName"`
		Errmsg string `json:"errmsg"`
	} `json:"writeConcernError"`
}

type expectation struct {
	CommandStartedEvent struct {
		CommandName  string          `json:"command_name"`
		DatabaseName string          `json:"database_name"`
		Command      json.RawMessage `json:"command"`
	} `json:"command_started_event"`
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

// compareVersions compares two version number strings (i.e. positive integers separated by
// periods). Comparisons are done to the lesser precision of the two versions. For example, 3.2 is
// considered equal to 3.2.11, whereas 3.2.0 is considered less than 3.2.11.
//
// Returns a positive int if version1 is greater than version2, a negative int if version1 is less
// than version2, and 0 if version1 is equal to version2.
func compareVersions(t *testing.T, v1 string, v2 string) int {
	n1 := strings.Split(v1, ".")
	n2 := strings.Split(v2, ".")

	for i := 0; i < int(math.Min(float64(len(n1)), float64(len(n2)))); i++ {
		i1, err := strconv.Atoi(n1[i])
		if err != nil {
			return 1
		}

		i2, err := strconv.Atoi(n2[i])
		if err != nil {
			return -1
		}

		difference := i1 - i2
		if difference != 0 {
			return difference
		}
	}

	return 0
}

func getServerVersion(db *Database) (string, error) {
	var serverStatus bsonx.Doc
	err := db.RunCommand(
		context.Background(),
		bsonx.Doc{{"serverStatus", bsonx.Int32(1)}},
	).Decode(&serverStatus)
	if err != nil {
		return "", err
	}

	version, err := serverStatus.LookupErr("version")
	if err != nil {
		return "", err
	}

	return version.StringValue(), nil
}

func getReadConcern(opt interface{}) *readconcern.ReadConcern {
	return readconcern.New(readconcern.Level(opt.(map[string]interface{})["level"].(string)))
}

func getWriteConcern(opt interface{}) *writeconcern.WriteConcern {
	if w, ok := opt.(map[string]interface{}); ok {
		var newTimeout time.Duration
		if conv, ok := w["wtimeout"].(float64); ok {
			newTimeout = time.Duration(int(conv)) * time.Millisecond
		}
		var newJ bool
		if conv, ok := w["j"].(bool); ok {
			newJ = conv
		}
		if conv, ok := w["w"].(string); ok && conv == "majority" {
			return writeconcern.New(writeconcern.WMajority(), writeconcern.J(newJ), writeconcern.WTimeout(newTimeout))
		} else if conv, ok := w["w"].(float64); ok {
			return writeconcern.New(writeconcern.W(int(conv)), writeconcern.J(newJ), writeconcern.WTimeout(newTimeout))
		}
	}
	return nil
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
					{"codeName", bsonx.String(failPoint.Data.WriteConcernError.Name)},
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

func createMonitoredTopology(t *testing.T, clock *session.ClusterClock, monitor *event.CommandMonitor, connstr *connstring.ConnString) *topology.Topology {
	if sessionsMonitoredTop != nil {
		return sessionsMonitoredTop // don't create the same topology twice
	}

	cs := testutil.ConnString(t)
	if connstr != nil {
		cs = *connstr
	}
	cs.HeartbeatInterval = time.Hour
	cs.HeartbeatIntervalSet = true

	opts := []topology.Option{
		topology.WithConnString(func(connstring.ConnString) connstring.ConnString { return cs }),
		topology.WithServerOptions(func(opts ...topology.ServerOption) []topology.ServerOption {
			return append(
				opts,
				topology.WithConnectionOptions(func(opts ...topology.ConnectionOption) []topology.ConnectionOption {
					return append(
						opts,
						topology.WithMonitor(func(*event.CommandMonitor) *event.CommandMonitor {
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

	err = sessionsMonitoredTop.Connect()
	if err != nil {
		t.Fatal(err)
	}

	err = operation.NewCommand(bsoncore.BuildDocument(nil, bsoncore.AppendInt32Element(nil, "dropDatabase", 1))).
		Database(testutil.DBName(t)).ServerSelector(description.WriteSelector()).Deployment(sessionsMonitoredTop).Execute(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	return sessionsMonitoredTop
}

func createSessionsMonitoredClient(t *testing.T, monitor *event.CommandMonitor) *Client {
	clock := &session.ClusterClock{}

	c := &Client{
		deployment:     createMonitoredTopology(t, clock, monitor, nil),
		connString:     testutil.ConnString(t),
		readPreference: readpref.Primary(),
		readConcern:    readconcern.Local(),
		clock:          clock,
		registry:       bson.DefaultRegistry,
		monitor:        monitor,
	}

	subscription, err := c.deployment.(driver.Subscriber).Subscribe()
	testhelpers.RequireNil(t, err, "error subscribing to topology: %s", err)
	c.sessionPool = session.NewPool(subscription.Updates)

	return c
}

// skip if the topology doesn't support sessions
func skipInvalidTopology(t *testing.T) {
	if os.Getenv("TOPOLOGY") == "server" {
		t.Skip("skipping for non-session supporting topology")
	}
}

type aggregator interface {
	Aggregate(context.Context, interface{}, ...*options.AggregateOptions) (*Cursor, error)
}

func createMonitoredClient(t *testing.T, monitor *event.CommandMonitor) *Client {
	client, err := NewClient()
	testhelpers.RequireNil(t, err, "unable to create client")
	client.deployment = testutil.GlobalMonitoredTopology(t, monitor)
	client.sessionPool = testutil.GlobalMonitoredSessionPool()
	client.connString = testutil.ConnString(t)
	client.readPreference = readpref.Primary()
	client.clock = &session.ClusterClock{}
	client.registry = bson.DefaultRegistry
	client.monitor = monitor
	return client
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

func getInt64(val bsonx.Val) int64 {
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

func compareValues(expected bsonx.Val, actual bsonx.Val) bool {
	if expected.IsNumber() {
		if !actual.IsNumber() {
			return false
		}

		return getInt64(expected) == getInt64(actual)
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

func compareDocs(t *testing.T, expected bsonx.Doc, actual bsonx.Doc) {
	// this is necessary even though Equal() exists for documents because types not match between commands and the BSON
	// documents given in test cases. for example, all numbers in the test case JSON are parsed as int64, but many nubmers
	// sent over the wire are type int32
	for _, expectedElem := range expected {
		aElem, err := actual.LookupElementErr(expectedElem.Key)
		testhelpers.RequireNil(t, err, "docs not equal. key %s not found in actual", expectedElem.Key)
		aVal := aElem.Value

		eVal := expectedElem.Value

		if doc, ok := eVal.DocumentOK(); ok {
			// special $$type assertion
			if typeVal, err := doc.LookupErr("$$type"); err == nil {
				// e.g. field: {$$type: "binData"} should assert that "field" is an element of type binary
				assertType(t, aElem.Value.Type(), typeVal.StringValue())
				continue
			}

			// nested doc
			compareDocs(t, doc, aVal.Document())

			// nested docs were equal
			continue
		}

		if !compareValues(eVal, aVal) {
			t.Fatalf("docs not equal because value mismatch for key %s", expectedElem.Key)
		}
	}
}
