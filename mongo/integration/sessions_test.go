// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
)

func TestSessionPool(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().MinServerVersion("3.6").CreateClient(false))
	defer mt.Close()

	mt.Run("pool LIFO", func(mt *mtest.T) {
		aSess, err := mt.Client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		bSess, err := mt.Client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)

		// end the sessions to return them to the pool
		aSess.EndSession(mtest.Background)
		bSess.EndSession(mtest.Background)

		firstSess, err := mt.Client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		defer firstSess.EndSession(mtest.Background)
		want := bSess.ID()
		got := firstSess.ID()
		assert.True(mt, sessionIDsEqual(mt, want, got), "expected session ID %v, got %v", want, got)

		secondSess, err := mt.Client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		defer secondSess.EndSession(mtest.Background)
		want = aSess.ID()
		got = secondSess.ID()
		assert.True(mt, sessionIDsEqual(mt, want, got), "expected session ID %v, got %v", want, got)
	})
	mt.Run("last use time updated", func(mt *mtest.T) {
		sess, err := mt.Client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		defer sess.EndSession(mtest.Background)
		initialLastUsedTime := getSessionLastUsedTime(mt, sess)

		err = mongo.WithSession(mtest.Background, sess, func(sc mongo.SessionContext) error {
			return mt.Client.Ping(sc, readpref.Primary())
		})
		assert.Nil(mt, err, "WithSession error: %v", err)

		newLastUsedTime := getSessionLastUsedTime(mt, sess)
		assert.True(mt, newLastUsedTime.After(initialLastUsedTime),
			"last used time %s is not after the initial last used time %s", newLastUsedTime, initialLastUsedTime)
	})
}

func TestSessions(t *testing.T) {
	mtOpts := mtest.NewOptions().MinServerVersion("3.6").Topologies(mtest.ReplicaSet, mtest.Sharded).
		CreateClient(false)
	mt := mtest.New(t, mtOpts)
	hosts := options.Client().ApplyURI(mt.ConnString()).Hosts

	// Pin to a single mongos so heartbeats/handshakes to other mongoses won't cause errors.
	clusterTimeOpts := mtest.NewOptions().
		ClientOptions(options.Client().SetHeartbeatInterval(50 * time.Second)).
		ClientType(mtest.Pinned).
		CreateClient(false)
	mt.RunOpts("cluster time", clusterTimeOpts, func(mt *mtest.T) {
		// $clusterTime included in commands

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

	clusterTimeHandshakeOpts := options.Client().
		SetHosts(hosts[:1]). // Prevent handshakes to other hosts from updating the cluster time.
		SetDirect(true).
		SetHeartbeatInterval(50 * time.Second) // Prevent extra heartbeats from updating the cluster time.
	clusterTimeHandshakeMtOpts := mtest.NewOptions().
		ClientType(mtest.Proxy).
		ClientOptions(clusterTimeHandshakeOpts).
		CreateCollection(false).
		SSL(false) // The proxy dialer doesn't work for SSL connections.
	mt.RunOpts("cluster time is updated from handshakes", clusterTimeHandshakeMtOpts, func(mt *mtest.T) {
		err := mt.Client.Ping(mtest.Background, mtest.PrimaryRp)
		assert.Nil(mt, err, "Ping error: %v", err)
		msgPairs := mt.GetProxiedMessages()
		assert.True(mt, len(msgPairs) > 2, "expected more than two messages, got %d", len(msgPairs))

		for idx, pair := range mt.GetProxiedMessages() {
			// Get the $clusterTime value sent to the server. The first three messages are the handshakes for the
			// heartbeat, RTT, and application connections. These should not contain $clusterTime because they happen on
			// connections that don't know the server's wire version and therefore don't know if the server supports
			// $clusterTime.
			_, err = pair.Sent.Command.LookupErr("$clusterTime")
			if idx <= 2 {
				assert.NotNil(mt, err, "expected no $clusterTime field in command %s", pair.Sent.Command)
				continue
			}

			// All messages after the first two should contain $clusterTime.
			assert.Nil(mt, err, "expected $clusterTime field in command %s", pair.Sent.Command)
		}
	})

	mt.RunOpts("explicit implicit session arguments", noClientOpts, func(mt *mtest.T) {
		// lsid is included in commands with explicit and implicit sessions

		sessionFunctions := createFunctionsSlice()
		for _, sf := range sessionFunctions {
			mt.Run(sf.name, func(mt *mtest.T) {
				// explicit session
				sess, err := mt.Client.StartSession()
				assert.Nil(mt, err, "StartSession error: %v", err)
				defer sess.EndSession(mtest.Background)
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
	mt.Run("wrong client", func(mt *mtest.T) {
		// a session can only be used in commands associated with the client that created it

		sessionFunctions := createFunctionsSlice()
		sess, err := mt.Client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		defer sess.EndSession(mtest.Background)

		for _, sf := range sessionFunctions {
			mt.Run(sf.name, func(mt *mtest.T) {
				err = sf.execute(mt, sess)
				assert.Equal(mt, mongo.ErrWrongClient, err, "expected error %v, got %v", mongo.ErrWrongClient, err)
			})
		}
	})
	mt.RunOpts("ended session", noClientOpts, func(mt *mtest.T) {
		// an ended session cannot be used in commands

		sessionFunctions := createFunctionsSlice()
		for _, sf := range sessionFunctions {
			mt.Run(sf.name, func(mt *mtest.T) {
				sess, err := mt.Client.StartSession()
				assert.Nil(mt, err, "StartSession error: %v", err)
				sess.EndSession(mtest.Background)

				err = sf.execute(mt, sess)
				assert.Equal(mt, session.ErrSessionEnded, err, "expected error %v, got %v", session.ErrSessionEnded, err)
			})
		}
	})
	mt.Run("implicit session returned", func(mt *mtest.T) {
		// implicit sessions are returned to the server session pool

		doc := bson.D{{"x", 1}}
		_, err := mt.Coll.InsertOne(mtest.Background, doc)
		assert.Nil(mt, err, "InsertOne error: %v", err)
		_, err = mt.Coll.InsertOne(mtest.Background, doc)
		assert.Nil(mt, err, "InsertOne error: %v", err)

		// create a cursor that will hold onto an implicit session and record the sent session ID
		mt.ClearEvents()
		cursor, err := mt.Coll.Find(mtest.Background, bson.D{})
		assert.Nil(mt, err, "Find error: %v", err)
		findID := extractSentSessionID(mt)
		assert.True(mt, cursor.Next(mtest.Background), "expected Next true, got false")

		// execute another operation and verify the find session ID was reused
		_, err = mt.Coll.DeleteOne(mtest.Background, bson.D{})
		assert.Nil(mt, err, "DeleteOne error: %v", err)
		deleteID := extractSentSessionID(mt)
		assert.Equal(mt, findID, deleteID, "expected session ID %v, got %v", findID, deleteID)
	})
	mt.Run("implicit session returned from getMore", func(mt *mtest.T) {
		// Client-side cursor that exhausts the results after a getMore immediately returns the implicit session to the pool.

		var docs []interface{}
		for i := 0; i < 5; i++ {
			docs = append(docs, bson.D{{"x", i}})
		}
		_, err := mt.Coll.InsertMany(mtest.Background, docs)
		assert.Nil(mt, err, "InsertMany error: %v", err)

		// run a find that will hold onto the implicit session and record the session ID
		mt.ClearEvents()
		cursor, err := mt.Coll.Find(mtest.Background, bson.D{}, options.Find().SetBatchSize(3))
		assert.Nil(mt, err, "Find error: %v", err)
		findID := extractSentSessionID(mt)

		// iterate past 4 documents, forcing a getMore. session should be returned to pool after getMore
		for i := 0; i < 4; i++ {
			assert.True(mt, cursor.Next(mtest.Background), "Next returned false on iteration %v", i)
		}

		// execute another operation and verify the find session ID was reused
		_, err = mt.Coll.DeleteOne(mtest.Background, bson.D{})
		assert.Nil(mt, err, "DeleteOne error: %v", err)
		deleteID := extractSentSessionID(mt)
		assert.Equal(mt, findID, deleteID, "expected session ID %v, got %v", findID, deleteID)
	})
	mt.Run("find and getMore use same ID", func(mt *mtest.T) {
		var docs []interface{}
		for i := 0; i < 3; i++ {
			docs = append(docs, bson.D{{"x", i}})
		}
		_, err := mt.Coll.InsertMany(mtest.Background, docs)
		assert.Nil(mt, err, "InsertMany error: %v", err)

		// run a find that will hold onto an implicit session and record the session ID
		mt.ClearEvents()
		cursor, err := mt.Coll.Find(mtest.Background, bson.D{}, options.Find().SetBatchSize(2))
		assert.Nil(mt, err, "Find error: %v", err)
		findID := extractSentSessionID(mt)
		assert.NotNil(mt, findID, "expected session ID for find, got nil")

		// iterate over all documents and record the session ID of the getMore
		for i := 0; i < 3; i++ {
			assert.True(mt, cursor.Next(mtest.Background), "Next returned false on iteration %v", i)
		}
		getMoreID := extractSentSessionID(mt)
		assert.Equal(mt, findID, getMoreID, "expected session ID %v, got %v", findID, getMoreID)
	})

	mt.Run("imperative API", func(mt *mtest.T) {
		mt.Run("round trip Session object", func(mt *mtest.T) {
			// Rountrip a Session object through NewSessionContext/ContextFromSession and assert that it is correctly
			// stored/retrieved.

			sess, err := mt.Client.StartSession()
			assert.Nil(mt, err, "StartSession error: %v", err)
			defer sess.EndSession(mtest.Background)

			sessCtx := mongo.NewSessionContext(mtest.Background, sess)
			assert.Equal(mt, sess.ID(), sessCtx.ID(), "expected Session ID %v, got %v", sess.ID(), sessCtx.ID())

			gotSess := mongo.SessionFromContext(sessCtx)
			assert.NotNil(mt, gotSess, "expected SessionFromContext to return non-nil value, got nil")
			assert.Equal(mt, sess.ID(), gotSess.ID(), "expected Session ID %v, got %v", sess.ID(), gotSess.ID())
		})

		txnOpts := mtest.NewOptions().RunOn(
			mtest.RunOnBlock{Topology: []mtest.TopologyKind{mtest.ReplicaSet}, MinServerVersion: "4.0"},
			mtest.RunOnBlock{Topology: []mtest.TopologyKind{mtest.Sharded}, MinServerVersion: "4.2"},
		)
		mt.RunOpts("run transaction", txnOpts, func(mt *mtest.T) {
			// Test that the imperative sessions API can be used to run a transaction.

			createSessionContext := func(mt *mtest.T) mongo.SessionContext {
				sess, err := mt.Client.StartSession()
				assert.Nil(mt, err, "StartSession error: %v", err)

				return mongo.NewSessionContext(mtest.Background, sess)
			}

			sessCtx := createSessionContext(mt)
			sess := mongo.SessionFromContext(sessCtx)
			assert.NotNil(mt, sess, "expected SessionFromContext to return non-nil value, got nil")
			defer sess.EndSession(mtest.Background)

			err := sess.StartTransaction()
			assert.Nil(mt, err, "StartTransaction error: %v", err)

			numDocs := 2
			for i := 0; i < numDocs; i++ {
				_, err = mt.Coll.InsertOne(sessCtx, bson.D{{"x", 1}})
				assert.Nil(mt, err, "InsertOne error at index %d: %v", i, err)
			}

			// Assert that the collection count is 0 before committing and numDocs after. This tests that the InsertOne
			// calls were actually executed in the transaction because the pre-commit count does not include them.
			assertCollectionCount(mt, 0)
			err = sess.CommitTransaction(sessCtx)
			assert.Nil(mt, err, "CommitTransaction error: %v", err)
			assertCollectionCount(mt, int64(numDocs))
		})
	})
}

func assertCollectionCount(mt *mtest.T, expectedCount int64) {
	mt.Helper()

	count, err := mt.Coll.CountDocuments(mtest.Background, bson.D{})
	assert.Nil(mt, err, "CountDocuments error: %v", err)
	assert.Equal(mt, expectedCount, count, "expected CountDocuments result %v, got %v", expectedCount, count)
}

type sessionFunction struct {
	name   string
	target string
	fnName string
	params []interface{} // should not include context
}

func (sf sessionFunction) execute(mt *mtest.T, sess mongo.Session) error {
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
		return mongo.WithSession(mtest.Background, sess, func(sc mongo.SessionContext) error {
			valueArgs := []reflect.Value{reflect.ValueOf(sc)}
			valueArgs = append(valueArgs, paramsValues...)
			returnValues := fn.Call(valueArgs)
			return extractReturnError(returnValues)
		})
	}
	valueArgs := []reflect.Value{reflect.ValueOf(mtest.Background)}
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
	default:
		return nil
	}
}

func extractSentSessionID(mt *mtest.T) []byte {
	lsid, err := mt.GetStartedEvent().Command.LookupErr("lsid")
	if err != nil {
		return nil
	}

	_, data := lsid.Document().Lookup("id").Binary()
	return data
}

func getSessionLastUsedTime(mt *mtest.T, sess mongo.Session) time.Time {
	xsess, ok := sess.(mongo.XSession)
	assert.True(mt, ok, "expected session to implement mongo.XSession, but got %T", sess)
	return xsess.ClientSession().LastUsed
}
