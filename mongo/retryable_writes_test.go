// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"strings"

	"time"

	"sync"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/mongo/driverlegacy/session"
	"go.mongodb.org/mongo-driver/x/mongo/driverlegacy/topology"
	"go.mongodb.org/mongo-driver/x/network/connection"
	"go.mongodb.org/mongo-driver/x/network/connstring"
)

const retryWritesDir = "../data/retryable-writes"

type retryTestFile struct {
	Data             json.RawMessage  `json:"data"`
	MinServerVersion string           `json:"minServerVersion"`
	MaxServerVersion string           `json:"maxServerVersion"`
	Tests            []*retryTestCase `json:"tests"`
}

type retryTestCase struct {
	Description   string                 `json:"description"`
	FailPoint     *failPoint             `json:"failPoint"`
	ClientOptions map[string]interface{} `json:"clientOptions"`
	Operation     *retryOperation        `json:"operation"`
	Outcome       *retryOutcome          `json:"outcome"`
}

type retryOperation struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type retryOutcome struct {
	Error      bool            `json:"error"`
	Result     json.RawMessage `json:"result"`
	Collection struct {
		Name string          `json:"name"`
		Data json.RawMessage `json:"data"`
	} `json:"collection"`
}

var retryMonitoredTopology *topology.Topology
var retryMonitoredTopologyOnce sync.Once

var retryStartedChan = make(chan *event.CommandStartedEvent, 100)

var retryMonitor = &event.CommandMonitor{
	Started: func(ctx context.Context, cse *event.CommandStartedEvent) {
		retryStartedChan <- cse
	},
}

func TestTxnNumberIncluded(t *testing.T) {
	client := createRetryMonitoredClient(t, retryMonitor)
	client.retryWrites = true

	db := client.Database("retry-writes")

	version, err := getServerVersion(db)
	require.NoError(t, err)
	if shouldSkipRetryTest(t, version) {
		t.Skip()
	}

	doc1 := map[string]interface{}{"x": 1}
	doc2 := map[string]interface{}{"y": 2}
	update := map[string]interface{}{"$inc": 1}
	var cases = []struct {
		op          *retryOperation
		includesTxn bool
	}{
		{&retryOperation{Name: "deleteOne"}, true},
		{&retryOperation{Name: "deleteMany"}, false},
		{&retryOperation{Name: "updateOne", Arguments: map[string]interface{}{"update": update}}, true},
		{&retryOperation{Name: "updateMany", Arguments: map[string]interface{}{"update": update}}, false},
		{&retryOperation{Name: "replaceOne"}, true},
		{&retryOperation{Name: "insertOne", Arguments: map[string]interface{}{"document": doc1}}, true},
		{&retryOperation{Name: "insertMany", Arguments: map[string]interface{}{
			"ordered": true, "documents": []interface{}{doc1, doc2}}}, true},
		{&retryOperation{Name: "insertMany", Arguments: map[string]interface{}{
			"ordered": false, "documents": []interface{}{doc1, doc2}}}, true},
		{&retryOperation{Name: "findOneAndReplace"}, true},
		{&retryOperation{Name: "findOneAndUpdate", Arguments: map[string]interface{}{"update": update}}, true},
		{&retryOperation{Name: "findOneAndDelete"}, true},
	}

	err = db.Drop(ctx)
	require.NoError(t, err)

	for _, tc := range cases {
		t.Run(tc.op.Name, func(t *testing.T) {
			coll := db.Collection(tc.op.Name)
			err = coll.Drop(ctx)
			require.NoError(t, err)

			// insert sample data
			_, err = coll.InsertOne(ctx, doc1)
			require.NoError(t, err)
			_, err = coll.InsertOne(ctx, doc2)
			require.NoError(t, err)

			for len(retryStartedChan) > 0 {
				<-retryStartedChan
			}

			executeRetryOperation(t, tc.op, nil, coll)

			var evt *event.CommandStartedEvent
			select {
			case evt = <-retryStartedChan:
			default:
				require.Fail(t, "Expected command started event")
			}

			if tc.includesTxn {
				require.NotNil(t, evt.Command.Lookup("txnNumber"))
			} else {
				require.Equal(t, evt.Command.Lookup("txnNumber"), bson.RawValue{})
			}
		})
	}
}

// test case for all RetryableWritesSpec tests
func TestRetryableWritesSpec(t *testing.T) {
	for _, file := range testhelpers.FindJSONFilesInDir(t, retryWritesDir) {
		runRetryTestFile(t, path.Join(retryWritesDir, file))
	}
}

func runRetryTestFile(t *testing.T, filepath string) {
	if strings.Contains(filepath, "bulk") {
		return
	}
	content, err := ioutil.ReadFile(filepath)
	require.NoError(t, err)

	var testfile retryTestFile
	require.NoError(t, json.Unmarshal(content, &testfile))

	dbName := "admin"
	dbAdmin := createTestDatabase(t, &dbName)

	version, err := getServerVersion(dbAdmin)
	require.NoError(t, err)

	// check if we should skip all retry tests
	if shouldSkipRetryTest(t, version) || os.Getenv("TOPOLOGY") == "sharded_cluster" {
		t.Skip()
	}

	// check if we should skip individual test file
	if shouldSkip(t, testfile.MinServerVersion, testfile.MaxServerVersion, dbAdmin) {
		return
	}

	for _, test := range testfile.Tests {
		runRetryTestCase(t, test, testfile.Data, dbAdmin)
	}

}

func runRetryTestCase(t *testing.T, test *retryTestCase, data json.RawMessage, dbAdmin *Database) {
	t.Run(test.Description, func(t *testing.T) {
		client := createTestClient(t)

		db := client.Database("retry-writes")
		collName := sanitizeCollectionName("retry-writes", test.Description)

		err := db.Drop(ctx)
		require.NoError(t, err)

		// insert data if present
		coll := db.Collection(collName)
		docsToInsert := docSliceToInterfaceSlice(docSliceFromRaw(t, data))
		if len(docsToInsert) > 0 {
			coll2, err := coll.Clone(options.Collection().SetWriteConcern(writeconcern.New(writeconcern.WMajority())))
			require.NoError(t, err)
			_, err = coll2.InsertMany(ctx, docsToInsert)
			require.NoError(t, err)
		}

		// configure failpoint if needed
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

		addClientOptions(client, test.ClientOptions)

		executeRetryOperation(t, test.Operation, test.Outcome, coll)

		verifyCollectionContents(t, coll, test.Outcome.Collection.Data)
	})

}

func executeRetryOperation(t *testing.T, op *retryOperation, outcome *retryOutcome, coll *Collection) {
	switch op.Name {
	case "deleteOne":
		res, err := executeDeleteOne(nil, coll, op.Arguments)
		if outcome == nil {
			return
		}
		if outcome.Error {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			verifyDeleteResult(t, res, outcome.Result)
		}
	case "deleteMany":
		_, _ = executeDeleteMany(nil, coll, op.Arguments)
		// no checking required for deleteMany
	case "updateOne":
		res, err := executeUpdateOne(nil, coll, op.Arguments)
		if outcome == nil {
			return
		}
		if outcome.Error {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			verifyUpdateResult(t, res, outcome.Result)
		}
	case "updateMany":
		_, _ = executeUpdateMany(nil, coll, op.Arguments)
		// no checking required for updateMany
	case "replaceOne":
		res, err := executeReplaceOne(nil, coll, op.Arguments)
		if outcome == nil {
			return
		}
		if outcome.Error {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			verifyUpdateResult(t, res, outcome.Result)
		}
	case "insertOne":
		res, err := executeInsertOne(nil, coll, op.Arguments)
		if outcome == nil {
			return
		}
		if outcome.Error {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			verifyInsertOneResult(t, res, outcome.Result)
		}
	case "insertMany":
		res, err := executeInsertMany(nil, coll, op.Arguments)
		if outcome == nil {
			return
		}
		if outcome.Error {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			verifyInsertManyResult(t, res, outcome.Result)
		}
	case "findOneAndUpdate":
		res := executeFindOneAndUpdate(nil, coll, op.Arguments)
		if outcome == nil {
			return
		}
		if outcome.Error {
			require.Error(t, res.err)
		} else {
			require.NoError(t, res.err)
			verifySingleResult(t, res, outcome.Result)
		}
	case "findOneAndDelete":
		res := executeFindOneAndDelete(nil, coll, op.Arguments)
		if outcome == nil {
			return
		}
		if outcome.Error {
			require.Error(t, res.err)
		} else {
			require.NoError(t, res.err)
			verifySingleResult(t, res, outcome.Result)
		}
	case "findOneAndReplace":
		res := executeFindOneAndReplace(nil, coll, op.Arguments)
		if outcome == nil {
			return
		}
		if outcome.Error {
			require.Error(t, res.err)
		} else {
			require.NoError(t, res.err)
			verifySingleResult(t, res, outcome.Result)
		}
	case "bulkWrite":
		// TODO reenable when bulk writes implemented
		t.Skip("Skipping until bulk writes implemented")
	}
}

func createRetryMonitoredClient(t *testing.T, monitor *event.CommandMonitor) *Client {
	clock := &session.ClusterClock{}

	c := &Client{
		topology:       createRetryMonitoredTopology(t, clock, monitor),
		connString:     testutil.ConnString(t),
		readPreference: readpref.Primary(),
		clock:          clock,
		registry:       bson.DefaultRegistry,
	}

	subscription, err := c.topology.Subscribe()
	testhelpers.RequireNil(t, err, "error subscribing to topology: %s", err)
	c.topology.SessionPool = session.NewPool(subscription.C)

	return c
}

func createRetryMonitoredTopology(t *testing.T, clock *session.ClusterClock, monitor *event.CommandMonitor) *topology.Topology {
	cs := testutil.ConnString(t)
	cs.HeartbeatInterval = time.Minute
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

	retryMonitoredTopologyOnce.Do(func() {
		retryMonitoredTopo, err := topology.New(opts...)
		if err != nil {
			t.Fatal(err)
		}
		err = retryMonitoredTopo.Connect(ctx)
		if err != nil {
			t.Fatal(err)
		}

		retryMonitoredTopology = retryMonitoredTopo
	})

	return retryMonitoredTopology
}

// skip entire test suite if server version less than 3.6 OR not a replica set
func shouldSkipRetryTest(t *testing.T, serverVersion string) bool {
	return compareVersions(t, serverVersion, "3.6") < 0 ||
		os.Getenv("TOPOLOGY") == "server"
}
