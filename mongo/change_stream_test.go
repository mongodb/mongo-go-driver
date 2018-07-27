package mongo

import (
	"context"
	"os"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/dispatch"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/stretchr/testify/require"
)

var collectionStartingDoc = bson.NewDocument(
	bson.EC.Int32("y", 1),
)

var doc1 = bson.NewDocument(
	bson.EC.Int32("x", 1),
)

var wcMajority = writeconcern.New(writeconcern.WMajority())

type errorCursor struct {
	errCode int32
}

func (er *errorCursor) ID() int64 {
	return 1
}

func (er *errorCursor) Next(ctx context.Context) bool {
	return false
}

func (er *errorCursor) Decode(interface{}) error {
	return nil
}

func (er *errorCursor) DecodeBytes() (bson.Reader, error) {
	return nil, nil
}

func (er *errorCursor) Err() error {
	return command.Error{
		Code: er.errCode,
	}
}

func (er *errorCursor) Close(ctx context.Context) error {
	return nil
}

func skipIfBelow36(t *testing.T) {
	serverVersion, err := getServerVersion(createTestDatabase(t, nil))
	require.NoError(t, err)

	if compareVersions(t, serverVersion, "3.6") < 0 {
		t.Skip()
	}
}

func createStream(t *testing.T, client *Client, dbName string, collName string, pipeline interface{}) (*Collection, Cursor) {
	client.writeConcern = wcMajority
	db := client.Database(dbName)
	err := db.Drop(ctx)
	testhelpers.RequireNil(t, err, "error dropping db: %s", err)

	coll := db.Collection(collName)
	coll.writeConcern = wcMajority
	_, err = coll.InsertOne(ctx, collectionStartingDoc) // create collection on server for 3.6

	drainChannels()
	stream, err := coll.Watch(ctx, pipeline)
	testhelpers.RequireNil(t, err, "error creating stream: %s", err)

	return coll, stream
}

func createCollectionStream(t *testing.T, dbName string, collName string, pipeline interface{}) (*Collection, Cursor) {
	client := createTestClient(t)
	return createStream(t, client, dbName, collName, pipeline)
}

func createMonitoredStream(t *testing.T, dbName string, collName string, pipeline interface{}) (*Collection, Cursor) {
	client := createMonitoredClient(t)
	return createStream(t, client, dbName, collName, pipeline)
}

func compareOptions(t *testing.T, expected *bson.Document, actual *bson.Document) {
	eIter := expected.Iterator()
	for eIter.Next() {
		elem := eIter.Element()
		key := elem.Key()

		if key == "resumeAfter" {
			continue
		}

		val := elem.Value()
		var aVal *bson.Value
		var err error

		if aVal, err = actual.LookupErr(key); err != nil {
			t.Fatalf("key %s not found in options document", key)
		}

		if !compareValues(val, aVal) {
			t.Fatalf("values for key %s do not match", key)
		}
	}
}

func comparePipelines(t *testing.T, expected *bson.Array, actual *bson.Array) {
	eIter, err := expected.Iterator()
	testhelpers.RequireNil(t, err, "error creating expected iterator: %s", err)

	aIter, err := actual.Iterator()
	testhelpers.RequireNil(t, err, "error creating actual iterator: %s", err)

	firstIteration := true
	for eIter.Next() {
		if !aIter.Next() {
			t.Fatal("actual has fewer values than expected")
		}

		eVal := eIter.Value()
		aVal := aIter.Value()

		if firstIteration {
			// found $changeStream document with options --> must compare options, ignoring extra resume token
			compareOptions(t, eVal.MutableDocument(), aVal.MutableDocument())

			firstIteration = false
			continue
		}

		if !compareValues(eVal, aVal) {
			t.Fatalf("pipelines do not mach")
		}
	}
}

func TestChangeStream(t *testing.T) {
	if os.Getenv("TOPOLOGY") == "server" {
		t.Skip("skipping invalid topology")
	}
	skipIfBelow36(t)

	t.Run("TestTrackResumeToken", func(t *testing.T) {
		// Stream must continuously track last seen resumeToken

		coll, stream := createCollectionStream(t, "TrackTokenDB", "TrackTokenColl", bson.NewDocument())
		defer closeCursor(stream)

		cs := stream.(*changeStream)
		if cs.resumeToken != nil {
			t.Fatalf("non-nil error on stream")
		}

		coll.writeConcern = wcMajority
		_, err := coll.InsertOne(ctx, doc1)
		testhelpers.RequireNil(t, err, "error running insertOne: %s", err)
		if !stream.Next(ctx) {
			t.Fatalf("no change found")
		}

		_, err = stream.DecodeBytes()
		testhelpers.RequireNil(t, err, "error decoding bytes: %s", err)

		testhelpers.RequireNotNil(t, cs.resumeToken, "no resume token found after first change")
	})

	t.Run("TestMissingResumeToken", func(t *testing.T) {
		// Stream will throw an error if the server response is missing the resume token

		coll, stream := createCollectionStream(t, "MissingTokenDB", "MissingTokenColl", []*bson.Document{
			bson.NewDocument(
				bson.EC.SubDocumentFromElements("$project", bson.EC.Int32("_id", 0)),
			),
		})
		defer closeCursor(stream)

		coll.writeConcern = wcMajority
		_, err := coll.InsertOne(ctx, doc1)
		testhelpers.RequireNil(t, err, "error running insertOne: %s", err)
		if !stream.Next(ctx) {
			t.Fatal("no change found")
		}

		_, err = stream.DecodeBytes()
		if err == nil || err != ErrMissingResumeToken {
			t.Fatalf("expected ErrMissingResumeToken, got %s", err)
		}
	})

	t.Run("ResumeOnce", func(t *testing.T) {
		// ChangeStream will automatically resume one time on a resumable error (including not master) with the initial
		// pipeline and options, except for the addition/update of a resumeToken.

		coll, stream := createMonitoredStream(t, "ResumeOnceDB", "ResumeOnceColl", nil)
		defer closeCursor(stream)
		startCmd := (<-startedChan).Command
		startPipeline := startCmd.Lookup("pipeline").MutableArray()

		cs := stream.(*changeStream)

		kc := command.KillCursors{
			NS:  cs.ns,
			IDs: []int64{cs.ID()},
		}

		_, err := dispatch.KillCursors(ctx, kc, cs.client.topology, cs.db.writeSelector)
		testhelpers.RequireNil(t, err, "error running killCursors cmd: %s", err)

		_, err = coll.InsertOne(ctx, doc1)
		testhelpers.RequireNil(t, err, "error inserting doc: %s", err)

		drainChannels()
		stream.Next(ctx)

		//Next() should cause getMore, killCursors and aggregate to run
		if len(startedChan) != 3 {
			t.Fatalf("expected 3 events waiting, got %d", len(startedChan))
		}

		<-startedChan            // getMore
		<-startedChan            // killCursors
		started := <-startedChan // aggregate

		if started.CommandName != "aggregate" {
			t.Fatalf("command name mismatch. expected aggregate got %s", started.CommandName)
		}

		pipeline := started.Command.Lookup("pipeline").MutableArray()

		if startPipeline.Len() != pipeline.Len() {
			t.Fatalf("pipeline len mismatch")
		}

		comparePipelines(t, startPipeline, pipeline)
	})

	t.Run("NoResumeForAggregateErrors", func(t *testing.T) {
		// ChangeStream will not attempt to resume on any error encountered while executing an aggregate command.

		dbName := "NoResumeDB"
		collName := "NoResumeColl"
		coll := createTestCollection(t, &dbName, &collName)

		stream, err := coll.Watch(ctx, []*bson.Document{
			bson.NewDocument(
				bson.EC.SubDocumentFromElements("$unsupportedStage", bson.EC.Int32("_id", 0)),
			),
		})
		testhelpers.RequireNil(t, stream, "stream was not nil")
		testhelpers.RequireNotNil(t, err, "error was nil")
	})

	t.Run("NoResumeErrors", func(t *testing.T) {
		// ChangeStream will not attempt to resume after encountering error code 11601 (Interrupted),
		// 136 (CappedPositionLost), or 237 (CursorKilled) while executing a getMore command.

		var tests = []struct {
			name    string
			errCode int32
		}{
			{"ErrorInterrupted", errorInterrupted},
			{"ErrorCappedPostionLost", errorCappedPositionLost},
			{"ErrorCursorKilled", errorCursorKilled},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				_, stream := createMonitoredStream(t, "ResumeOnceDB", "ResumeOnceColl", nil)
				defer closeCursor(stream)
				cs := stream.(*changeStream)
				cs.cursor = &errorCursor{
					errCode: tc.errCode,
				}

				drainChannels()
				if stream.Next(ctx) {
					t.Fatal("stream Next() returned true, expected false")
				}

				// no commands should be started because fake cursor's Next() does not call getMore
				if len(startedChan) != 0 {
					t.Fatalf("expected 1 command started, got %d", len(startedChan))
				}
			})
		}
	})

	t.Run("ServerSelection", func(t *testing.T) {
		// ChangeStream will perform server selection before attempting to resume, using initial readPreference
		t.Skip("Skipping for lack of SDAM monitoring")
	})

	t.Run("CursorNotClosed", func(t *testing.T) {
		// Ensure that a cursor returned from an aggregate command with a cursor id and an initial empty batch is not

		_, stream := createCollectionStream(t, "CursorNotClosedDB", "CursorNotClosedColl", nil)
		defer closeCursor(stream)
		cs := stream.(*changeStream)

		if cs.sess.Terminated {
			t.Fatalf("session was prematurely terminated")
		}
	})

	t.Run("NoExceptionFromKillCursors", func(t *testing.T) {
		// The killCursors command sent during the "Resume Process" must not be allowed to throw an exception

		// fail points don't work for mongos or <4.0
		if os.Getenv("TOPOLOGY") == "sharded_cluster" {
			t.Skip("skipping for sharded clusters")
		}

		version, err := getServerVersion(createTestDatabase(t, nil))
		testhelpers.RequireNil(t, err, "error getting server version: %s", err)

		if compareVersions(t, version, "4.0") < 0 {
			t.Skip("skipping for version < 4.0")
		}

		coll, stream := createMonitoredStream(t, "NoExceptionsDB", "NoExceptionsColl", nil)
		defer closeCursor(stream)
		cs := stream.(*changeStream)

		// kill cursor to force a resumable error
		kc := command.KillCursors{
			NS:  cs.ns,
			IDs: []int64{cs.ID()},
		}

		_, err = dispatch.KillCursors(ctx, kc, cs.client.topology, cs.db.writeSelector)
		testhelpers.RequireNil(t, err, "error running killCursors cmd: %s", err)

		adminDb := coll.client.Database("admin")
		_, err = adminDb.RunCommand(ctx, bson.NewDocument(
			bson.EC.String("configureFailPoint", "failCommand"),
			bson.EC.SubDocument("mode", bson.NewDocument(
				bson.EC.Int32("times", 1),
			)),
			bson.EC.SubDocument("data", bson.NewDocument(
				bson.EC.ArrayFromElements("failCommands", bson.VC.String("killCursors")),
				bson.EC.Int32("errorCode", 184),
			)),
		))
		testhelpers.RequireNil(t, err, "error creating fail point: %s", err)

		if !stream.Next(ctx) {
			t.Fatal("stream Next() returned false, expected true")
		}
	})

	t.Run("OperationTimeIncluded", func(t *testing.T) {
		// $changeStream stage for ChangeStream against a server >=4.0 that has not received any results yet MUST
		// include a startAtOperationTime option when resuming a changestream.

		version, err := getServerVersion(createTestDatabase(t, nil))
		testhelpers.RequireNil(t, err, "error getting server version: %s", err)

		if compareVersions(t, version, "4.0") < 0 {
			t.Skip("skipping for version < 4.0")
		}

		_, stream := createMonitoredStream(t, "IncludeTimeDB", "IncludeTimeColl", nil)
		defer closeCursor(stream)
		cs := stream.(*changeStream)

		// kill cursor to force a resumable error
		kc := command.KillCursors{
			NS:  cs.ns,
			IDs: []int64{cs.ID()},
		}

		_, err = dispatch.KillCursors(ctx, kc, cs.client.topology, cs.db.writeSelector)
		testhelpers.RequireNil(t, err, "error running killCursors cmd: %s", err)

		drainChannels()
		stream.Next(ctx)

		// channel should have getMore, killCursors, and aggregate
		if len(startedChan) != 3 {
			t.Fatalf("expected 3 commands started, got %d", len(startedChan))
		}

		<-startedChan
		<-startedChan

		aggCmd := <-startedChan
		if aggCmd.CommandName != "aggregate" {
			t.Fatalf("command name mismatch. expected aggregate, got %s", aggCmd.CommandName)
		}

		pipeline := aggCmd.Command.Lookup("pipeline").MutableArray()
		csVal, err := pipeline.Lookup(0) // doc with nested options document (key $changeStream)
		testhelpers.RequireNil(t, err, "pipeline is empty")

		optsVal, err := csVal.MutableDocument().LookupErr("$changeStream")
		testhelpers.RequireNil(t, err, "key $changeStream not found")

		if _, err := optsVal.MutableDocument().LookupErr("startAtOperationTime"); err != nil {
			t.Fatal("key startAtOperationTime not found in command")
		}
	})

	// There's another test: ChangeStream will resume after a killCursors command is issued for its child cursor.
	// But, killCursors was already used to cause an error for the ResumeOnce test, so this does not need to be tested
	// again.
}
