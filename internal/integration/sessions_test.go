// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/mongoutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
	"golang.org/x/sync/errgroup"
)

func TestSessionPool(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().MinServerVersion("3.6").CreateClient(false))

	mt.Run("last use time updated", func(mt *mtest.T) {
		sess, err := mt.Client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		defer sess.EndSession(context.Background())
		initialLastUsedTime := sess.ClientSession().LastUsed

		err = mongo.WithSession(context.Background(), sess, func(sc context.Context) error {
			return mt.Client.Ping(sc, readpref.Primary())
		})
		assert.Nil(mt, err, "WithSession error: %v", err)

		newLastUsedTime := sess.ClientSession().LastUsed
		assert.True(mt, newLastUsedTime.After(initialLastUsedTime),
			"last used time %s is not after the initial last used time %s", newLastUsedTime, initialLastUsedTime)
	})
}

func TestSessions(t *testing.T) {
	mtOpts := mtest.NewOptions().MinServerVersion("3.6").Topologies(mtest.ReplicaSet, mtest.Sharded).
		CreateClient(false)
	mt := mtest.New(t, mtOpts)

	mt.Run("imperative API", func(mt *mtest.T) {
		mt.Run("round trip Session object", func(mt *mtest.T) {
			// Roundtrip a Session object through NewSessionContext/ContextFromSession and assert that it is correctly
			// stored/retrieved.

			sess, err := mt.Client.StartSession()
			assert.Nil(mt, err, "StartSession error: %v", err)
			defer sess.EndSession(context.Background())

			ctx := mongo.NewSessionContext(context.Background(), sess)

			gotSess := mongo.SessionFromContext(ctx)
			assert.NotNil(mt, gotSess, "expected SessionFromContext to return non-nil value, got nil")
			assert.Equal(mt, sess.ID(), gotSess.ID(), "expected Session ID %v, got %v", sess.ID(), gotSess.ID())
		})

		txnOpts := mtest.NewOptions().RunOn(
			mtest.RunOnBlock{Topology: []mtest.TopologyKind{mtest.ReplicaSet}, MinServerVersion: "4.0"},
			mtest.RunOnBlock{Topology: []mtest.TopologyKind{mtest.Sharded}, MinServerVersion: "4.2"},
		)
		mt.RunOpts("run transaction", txnOpts, func(mt *mtest.T) {
			// Test that the imperative sessions API can be used to run a transaction.

			createSessionContext := func(mt *mtest.T) context.Context {
				sess, err := mt.Client.StartSession()
				assert.Nil(mt, err, "StartSession error: %v", err)

				return mongo.NewSessionContext(context.Background(), sess)
			}

			ctx := createSessionContext(mt)
			sess := mongo.SessionFromContext(ctx)
			assert.NotNil(mt, sess, "expected SessionFromContext to return non-nil value, got nil")
			defer sess.EndSession(context.Background())

			err := sess.StartTransaction()
			assert.Nil(mt, err, "StartTransaction error: %v", err)

			numDocs := 2
			for i := 0; i < numDocs; i++ {
				_, err = mt.Coll.InsertOne(ctx, bson.D{{"x", 1}})
				assert.Nil(mt, err, "InsertOne error at index %d: %v", i, err)
			}

			// Assert that the collection count is 0 before committing and numDocs after. This tests that the InsertOne
			// calls were actually executed in the transaction because the pre-commit count does not include them.
			assertCollectionCount(mt, 0)
			err = sess.CommitTransaction(ctx)
			assert.Nil(mt, err, "CommitTransaction error: %v", err)
			assertCollectionCount(mt, int64(numDocs))
		})
	})

	unackWcOpts := options.Collection().SetWriteConcern(writeconcern.Unacknowledged())
	mt.RunOpts("unacknowledged write", mtest.NewOptions().CollectionOptions(unackWcOpts), func(mt *mtest.T) {
		// unacknowledged write during a session should result in an error
		sess, err := mt.Client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		defer sess.EndSession(context.Background())

		var res *mongo.InsertOneResult

		err = mongo.WithSession(context.Background(), sess, func(sc context.Context) error {
			res, err = mt.Coll.InsertOne(sc, bson.D{{"x", 1}})

			return err
		})

		assert.NoError(mt, err)
		assert.False(mt, res.Acknowledged)
	})

	// Regression test for GODRIVER-2533. Note that this test assumes the race
	// detector is enabled (GODRIVER-2072).
	mt.Run("NumberSessionsInProgress data race", func(mt *mtest.T) {
		// Use two goroutines to execute a few simultaneous runs of NumberSessionsInProgress
		// and a basic collection operation (CountDocuments).
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()

			for i := 0; i < 100; i++ {
				time.Sleep(100 * time.Microsecond)
				_ = mt.Client.NumberSessionsInProgress()
			}
		}()
		go func() {
			defer wg.Done()

			for i := 0; i < 100; i++ {
				time.Sleep(100 * time.Microsecond)
				_, err := mt.Coll.CountDocuments(context.Background(), bson.D{})
				assert.Nil(mt, err, "CountDocument error: %v", err)
			}
		}()

		wg.Wait()
	})
}

func TestSessionsProse(t *testing.T) {
	mtOpts := mtest.
		NewOptions().
		MinServerVersion("3.6").
		Topologies(mtest.ReplicaSet, mtest.Sharded).
		CreateClient(false)

	mt := mtest.New(t, mtOpts)

	hosts, err := mongoutil.HostsFromURI(mtest.ClusterURI())
	require.NoError(t, err)

	mt.Run("1 setting both snapshot and causalConsistency to true is not allowed", func(mt *mtest.T) {
		// causalConsistency and snapshot are mutually exclusive
		sessOpts := options.Session().SetCausalConsistency(true).SetSnapshot(true)
		_, err := mt.Client.StartSession(sessOpts)
		assert.NotNil(mt, err, "expected StartSession error, got nil")
		expectedErr := errors.New("causal consistency and snapshot cannot both be set for a session")
		assert.Equal(mt, expectedErr, err, "expected error %v, got %v", expectedErr, err)
	})

	mt.Run("2 pool is LIFO", func(mt *mtest.T) {
		aSess, err := mt.Client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		bSess, err := mt.Client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)

		// end the sessions to return them to the pool
		aSess.EndSession(context.Background())
		bSess.EndSession(context.Background())

		firstSess, err := mt.Client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		defer firstSess.EndSession(context.Background())
		want := bSess.ID()
		got := firstSess.ID()
		assert.True(mt, sessionIDsEqual(mt, want, got), "expected session ID %v, got %v", want, got)

		secondSess, err := mt.Client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		defer secondSess.EndSession(context.Background())
		want = aSess.ID()
		got = secondSess.ID()
		assert.True(mt, sessionIDsEqual(mt, want, got), "expected session ID %v, got %v", want, got)
	})

	// Pin to a single mongos so heartbeats/handshakes to other mongoses
	// won't cause errors.
	clusterTimeOpts := mtest.NewOptions().
		ClientOptions(options.Client().SetHeartbeatInterval(50 * time.Second)).
		ClientType(mtest.Pinned).
		CreateClient(false)

	mt.RunOpts("3 clusterTime in commands", clusterTimeOpts, func(mt *mtest.T) {
		serverStatus := sessionFunction{"server status", "database", "RunCommand", []interface{}{bson.D{{"serverStatus", 1}}}}
		insert := sessionFunction{"insert one", "collection", "InsertOne", []interface{}{bson.D{{"x", 1}}}}
		agg := sessionFunction{"aggregate", "collection", "Aggregate", []interface{}{mongo.Pipeline{}}}
		find := sessionFunction{"find", "collection", "Find", []interface{}{bson.D{}}}

		sessionFunctions := []sessionFunction{serverStatus, insert, agg, find}
		for _, sf := range sessionFunctions {
			mt.Run(sf.name, func(mt *mtest.T) {
				err := sf.execute(mt, nil)
				assert.Nil(mt, err, "%v error: %v", sf.name, err)

				// assert $clusterTime was sent to server
				started := mt.GetStartedEvent()
				assert.NotNil(mt, started, "expected started event, got nil")
				_, err = started.Command.LookupErr("$clusterTime")
				assert.Nil(mt, err, "$clusterTime not sent")

				// record response cluster time
				succeeded := mt.GetSucceededEvent()
				assert.NotNil(mt, succeeded, "expected succeeded event, got nil")
				replyClusterTimeVal, err := succeeded.Reply.LookupErr("$clusterTime")
				assert.Nil(mt, err, "$clusterTime not found in response")

				// call function again
				err = sf.execute(mt, nil)
				assert.Nil(mt, err, "%v error: %v", sf.name, err)

				// find cluster time sent to server and assert it is the same as the one in the previous response
				sentClusterTimeVal, err := mt.GetStartedEvent().Command.LookupErr("$clusterTime")
				assert.Nil(mt, err, "$clusterTime not sent")
				replyClusterTimeDoc := replyClusterTimeVal.Document()
				sentClusterTimeDoc := sentClusterTimeVal.Document()
				assert.Equal(mt, replyClusterTimeDoc, sentClusterTimeDoc,
					"expected cluster time %v, got %v", replyClusterTimeDoc, sentClusterTimeDoc)
			})
		}
	})

	mt.RunOpts("4 explicit and implicit session arguments", noClientOpts, func(mt *mtest.T) {
		// lsid is included in commands with explicit and implicit sessions

		sessionFunctions := createFunctionsSlice()
		for _, sf := range sessionFunctions {
			mt.Run(sf.name, func(mt *mtest.T) {
				// explicit session
				sess, err := mt.Client.StartSession()
				assert.Nil(mt, err, "StartSession error: %v", err)
				defer sess.EndSession(context.Background())
				mt.ClearEvents()

				_ = sf.execute(mt, sess) // don't check error because we only care about lsid
				_, wantID := sess.ID().Lookup("id").Binary()
				gotID := extractSentSessionID(mt)
				assert.True(mt, bytes.Equal(wantID, gotID), "expected session ID %v, got %v", wantID, gotID)

				// implicit session
				_ = sf.execute(mt, nil)
				gotID = extractSentSessionID(mt)
				assert.NotNil(mt, gotID, "expected lsid, got nil")
			})
		}
	})

	mt.Run("5 session argument is for the right client", func(mt *mtest.T) {
		// a session can only be used in commands associated with the client that created it

		sessionFunctions := createFunctionsSlice()
		sess, err := mt.Client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		defer sess.EndSession(context.Background())

		for _, sf := range sessionFunctions {
			mt.Run(sf.name, func(mt *mtest.T) {
				err = sf.execute(mt, sess)
				assert.Equal(mt, mongo.ErrWrongClient, err, "expected error %v, got %v", mongo.ErrWrongClient, err)
			})
		}
	})

	const proseTest6 = "6 no further operations can be performed using a session after endSession has been called"
	mt.RunOpts(proseTest6, noClientOpts, func(mt *mtest.T) {
		// an ended session cannot be used in commands

		sessionFunctions := createFunctionsSlice()
		for _, sf := range sessionFunctions {
			mt.Run(sf.name, func(mt *mtest.T) {
				sess, err := mt.Client.StartSession()
				assert.Nil(mt, err, "StartSession error: %v", err)
				sess.EndSession(context.Background())

				err = sf.execute(mt, sess)
				assert.Equal(mt, session.ErrSessionEnded, err, "expected error %v, got %v", session.ErrSessionEnded, err)
			})
		}
	})

	mt.Run("7 authenticating as multiple users suppresses implicit sessions", func(mt *mtest.T) {
		mt.Skip("Go Driver does not allow simultaneous authentication with multiple users.")
	})

	mt.Run("8 client side cursor that exhausts the results on the initial query immediately returns the implicit session to the pool",
		func(mt *mtest.T) {
			// implicit sessions are returned to the server session pool

			doc := bson.D{{"x", 1}}
			_, err := mt.Coll.InsertOne(context.Background(), doc)
			assert.Nil(mt, err, "InsertOne error: %v", err)
			_, err = mt.Coll.InsertOne(context.Background(), doc)
			assert.Nil(mt, err, "InsertOne error: %v", err)

			// create a cursor that will hold onto an implicit session and record the sent session ID
			mt.ClearEvents()
			cursor, err := mt.Coll.Find(context.Background(), bson.D{})
			assert.Nil(mt, err, "Find error: %v", err)
			findID := extractSentSessionID(mt)
			assert.True(mt, cursor.Next(context.Background()), "expected Next true, got false")

			// execute another operation and verify the find session ID was reused
			_, err = mt.Coll.DeleteOne(context.Background(), bson.D{})
			assert.Nil(mt, err, "DeleteOne error: %v", err)
			deleteID := extractSentSessionID(mt)
			assert.Equal(mt, findID, deleteID, "expected session ID %v, got %v", findID, deleteID)
		})

	mt.Run("9 client side cursor that exhausts the results after a getMore immediately returns the implicit session to the pool",
		func(mt *mtest.T) {
			// Client-side cursor that exhausts the results after a getMore immediately returns the implicit session to the pool.

			var docs []interface{}
			for i := 0; i < 5; i++ {
				docs = append(docs, bson.D{{"x", i}})
			}

			_, err := mt.Coll.InsertMany(context.Background(), docs)
			assert.Nil(mt, err, "InsertMany error: %v", err)

			// run a find that will hold onto the implicit session and record the session ID
			mt.ClearEvents()
			cursor, err := mt.Coll.Find(context.Background(), bson.D{}, options.Find().SetBatchSize(3))
			assert.Nil(mt, err, "Find error: %v", err)
			findID := extractSentSessionID(mt)

			// iterate past 4 documents, forcing a getMore. session should be returned to pool after getMore
			for i := 0; i < 4; i++ {
				assert.True(mt, cursor.Next(context.Background()), "Next returned false on iteration %v", i)
			}

			// execute another operation and verify the find session ID was reused
			_, err = mt.Coll.DeleteOne(context.Background(), bson.D{})
			assert.Nil(mt, err, "DeleteOne error: %v", err)
			deleteID := extractSentSessionID(mt)
			assert.Equal(mt, findID, deleteID, "expected session ID %v, got %v", findID, deleteID)
		})

	mt.Run("10 no remaining sessions are checked out after each functional test", func(mt *mtest.T) {
		mt.Skip("This is tested individually in each functional test.")
	})

	mt.Run("11 for every combination of topology and readPreference, ensure that find and getMore both send the same session id", func(mt *mtest.T) {
		var docs []interface{}
		for i := 0; i < 3; i++ {
			docs = append(docs, bson.D{{"x", i}})
		}
		_, err := mt.Coll.InsertMany(context.Background(), docs)
		assert.Nil(mt, err, "InsertMany error: %v", err)

		// run a find that will hold onto an implicit session and record the session ID
		mt.ClearEvents()
		cursor, err := mt.Coll.Find(context.Background(), bson.D{}, options.Find().SetBatchSize(2))
		assert.Nil(mt, err, "Find error: %v", err)
		findID := extractSentSessionID(mt)
		assert.NotNil(mt, findID, "expected session ID for find, got nil")

		// iterate over all documents and record the session ID of the getMore
		for i := 0; i < 3; i++ {
			assert.True(mt, cursor.Next(context.Background()), "Next returned false on iteration %v", i)
		}
		getMoreID := extractSentSessionID(mt)
		assert.Equal(mt, findID, getMoreID, "expected session ID %v, got %v", findID, getMoreID)
	})

	sessallocopts := mtest.NewOptions().ClientOptions(options.Client().SetMaxPoolSize(1).SetRetryWrites(true).
		SetHosts(hosts[:1]))
	mt.RunOpts("14 implicit session allocation", sessallocopts, func(mt *mtest.T) {
		// TODO(GODRIVER-2844): Fix and unskip this test case.
		mt.Skip("Test fails frequently, skipping. See GODRIVER-2844")

		ops := map[string]func(ctx context.Context) error{
			"insert": func(ctx context.Context) error {
				_, err := mt.Coll.InsertOne(ctx, bson.D{})
				return err
			},
			"delete": func(ctx context.Context) error {
				_, err := mt.Coll.DeleteOne(ctx, bson.D{})
				return err
			},
			"update": func(ctx context.Context) error {
				_, err := mt.Coll.UpdateOne(ctx, bson.D{}, bson.D{{"$set", bson.D{{"a", 1}}}})
				return err
			},
			"bulkWrite": func(ctx context.Context) error {
				model := mongo.NewUpdateOneModel().
					SetFilter(bson.D{}).
					SetUpdate(bson.D{{"$set", bson.D{{"a", 1}}}})
				_, err := mt.Coll.BulkWrite(ctx, []mongo.WriteModel{model})
				return err
			},
			"findOneAndDelete": func(ctx context.Context) error {
				result := mt.Coll.FindOneAndDelete(ctx, bson.D{})
				if err := result.Err(); err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
					return err
				}
				return nil
			},
			"findOneAndUpdate": func(ctx context.Context) error {
				result := mt.Coll.FindOneAndUpdate(ctx, bson.D{},
					bson.D{{"$set", bson.D{{"a", 1}}}})

				if err := result.Err(); err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
					return err
				}
				return nil
			},
			"findOneAndReplace": func(ctx context.Context) error {
				result := mt.Coll.FindOneAndReplace(ctx, bson.D{}, bson.D{{"a", 1}})
				if err := result.Err(); err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
					return err
				}
				return nil
			},
			"find": func(ctx context.Context) error {
				cursor, err := mt.Coll.Find(ctx, bson.D{})
				if err != nil {
					return err
				}
				return cursor.All(ctx, &bson.A{})
			},
		}

		// maintainedOneSession asserts that exactly one session is used for all operations at least once
		// across the retries of this test.
		var maintainedOneSession bool

		// minimumSessionCount asserts the least amount of sessions used over all the retries of the
		// operations. For example, if we retry 5 times we could result in session use { 1, 2, 1, 1, 6 }. In
		// this case, minimumSessionCount should be 1.
		var minimumSessionCount int

		// limitedSessionUse asserts that the number of allocated sessions is strictly less than the number of
		// concurrent operations in every retry of this test. In this instance it would be less than (but NOT
		// equal to the number of operations).
		limitedSessionUse := true

		retrycount := 5
		for i := 1; i <= retrycount; i++ {
			errs, ctx := errgroup.WithContext(context.Background())

			// Execute the ops list concurrently.
			for cmd, op := range ops {
				op := op
				cmd := cmd
				errs.Go(func() error {
					if err := op(ctx); err != nil {
						return fmt.Errorf("error running %s operation: %w", cmd, err)
					}
					return nil
				})
			}
			err := errs.Wait()
			assert.Nil(mt, err, "expected no error, got: %v", err)

			// Get all started events and collect them by the session ID.
			set := make(map[string]bool)
			for _, event := range mt.GetAllStartedEvents() {
				lsid := event.Command.Lookup("lsid")
				set[lsid.String()] = true
			}

			setSize := len(set)
			if setSize == 1 {
				maintainedOneSession = true
			} else if setSize < minimumSessionCount || minimumSessionCount == 0 {
				// record the minimum number of sessions we used over all retries.
				minimumSessionCount = setSize
			}

			if setSize >= len(ops) {
				limitedSessionUse = false
			}
		}

		oneSessMsg := "expected one session across all %v operations for at least 1/%v retries, got: %v"
		assert.True(mt, maintainedOneSession, oneSessMsg, len(ops), retrycount, minimumSessionCount)

		limitedSessMsg := "expected session count to be less than the number of operations: %v"
		assert.True(mt, limitedSessionUse, limitedSessMsg, len(ops))

	})
}

type sessionFunction struct {
	name   string
	target string
	fnName string
	params []interface{} // should not include context
}

func (sf sessionFunction) execute(mt *mtest.T, sess *mongo.Session) error {
	var target reflect.Value
	switch sf.target {
	case "client":
		target = reflect.ValueOf(mt.Client)
	case "database":
		// use a different database for drops because any executed after the drop will get "database not found"
		// errors on sharded clusters
		if sf.name != "drop database" {
			target = reflect.ValueOf(mt.DB)
			break
		}
		target = reflect.ValueOf(mt.Client.Database("sessionsTestsDropDatabase"))
	case "collection":
		target = reflect.ValueOf(mt.Coll)
	case "indexView":
		target = reflect.ValueOf(mt.Coll.Indexes())
	default:
		mt.Fatalf("unrecognized target: %v", sf.target)
	}

	fn := target.MethodByName(sf.fnName)
	paramsValues := interfaceSliceToValueSlice(sf.params)

	if sess != nil {
		return mongo.WithSession(context.Background(), sess, func(sc context.Context) error {
			valueArgs := []reflect.Value{reflect.ValueOf(sc)}
			valueArgs = append(valueArgs, paramsValues...)
			returnValues := fn.Call(valueArgs)
			return extractReturnError(returnValues)
		})
	}
	valueArgs := []reflect.Value{reflect.ValueOf(context.Background())}
	valueArgs = append(valueArgs, paramsValues...)
	returnValues := fn.Call(valueArgs)
	return extractReturnError(returnValues)
}

func createFunctionsSlice() []sessionFunction {
	insertManyDocs := []interface{}{bson.D{{"x", 1}}}
	fooIndex := mongo.IndexModel{
		Keys:    bson.D{{"foo", -1}},
		Options: options.Index().SetName("fooIndex"),
	}
	manyIndexes := []mongo.IndexModel{fooIndex}
	updateDoc := bson.D{{"$inc", bson.D{{"x", 1}}}}

	return []sessionFunction{
		{"list databases", "client", "ListDatabases", []interface{}{bson.D{}}},
		{"insert one", "collection", "InsertOne", []interface{}{bson.D{{"x", 1}}}},
		{"insert many", "collection", "InsertMany", []interface{}{insertManyDocs}},
		{"delete one", "collection", "DeleteOne", []interface{}{bson.D{}}},
		{"delete many", "collection", "DeleteMany", []interface{}{bson.D{}}},
		{"update one", "collection", "UpdateOne", []interface{}{bson.D{}, updateDoc}},
		{"update many", "collection", "UpdateMany", []interface{}{bson.D{}, updateDoc}},
		{"replace one", "collection", "ReplaceOne", []interface{}{bson.D{}, bson.D{}}},
		{"aggregate", "collection", "Aggregate", []interface{}{mongo.Pipeline{}}},
		{"estimated document count", "collection", "EstimatedDocumentCount", nil},
		{"distinct", "collection", "Distinct", []interface{}{"field", bson.D{}}},
		{"find", "collection", "Find", []interface{}{bson.D{}}},
		{"find one and delete", "collection", "FindOneAndDelete", []interface{}{bson.D{}}},
		{"find one and replace", "collection", "FindOneAndReplace", []interface{}{bson.D{}, bson.D{}}},
		{"find one and update", "collection", "FindOneAndUpdate", []interface{}{bson.D{}, updateDoc}},
		{"drop collection", "collection", "Drop", nil},
		{"list collections", "database", "ListCollections", []interface{}{bson.D{}}},
		{"drop database", "database", "Drop", nil},
		{"create one index", "indexView", "CreateOne", []interface{}{fooIndex}},
		{"create many indexes", "indexView", "CreateMany", []interface{}{manyIndexes}},
		{"drop one index", "indexView", "DropOne", []interface{}{"barIndex"}},
		{"drop all indexes", "indexView", "DropAll", nil},
		{"list indexes", "indexView", "List", nil},
	}
}

func assertCollectionCount(mt *mtest.T, expectedCount int64) {
	mt.Helper()

	count, err := mt.Coll.CountDocuments(context.Background(), bson.D{})
	require.NoError(mt, err, "CountDocuments error")
	assert.Equal(mt, expectedCount, count, "expected CountDocuments result %v, got %v", expectedCount, count)
}

func sessionIDsEqual(mt *mtest.T, id1, id2 bson.Raw) bool {
	first, err := id1.LookupErr("id")
	assert.Nil(mt, err, "id not found in document %v", id1)
	second, err := id2.LookupErr("id")
	assert.Nil(mt, err, "id not found in document %v", id2)

	_, firstUUID := first.Binary()
	_, secondUUID := second.Binary()
	return bytes.Equal(firstUUID, secondUUID)
}

func interfaceSliceToValueSlice(args []interface{}) []reflect.Value {
	vals := make([]reflect.Value, 0, len(args))
	for _, arg := range args {
		vals = append(vals, reflect.ValueOf(arg))
	}
	return vals
}

func extractReturnError(returnValues []reflect.Value) error {
	errVal := returnValues[len(returnValues)-1]
	switch converted := errVal.Interface().(type) {
	case error:
		return converted
	case *mongo.SingleResult:
		return converted.Err()
	case *mongo.DistinctResult:
		return converted.Err()
	default:
		return nil
	}
}

func extractSentSessionID(mt *mtest.T) []byte {
	event := mt.GetStartedEvent()
	if event == nil {
		return nil
	}
	lsid, err := event.Command.LookupErr("lsid")
	if err != nil {
		return nil
	}

	_, data := lsid.Document().Lookup("id").Binary()
	return data
}
