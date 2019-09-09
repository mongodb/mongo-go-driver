// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"os"
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
