package mongo

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/stretchr/testify/require"
)

func isServerError(err error) bool {
	_, ok := err.(command.Error)
	return ok
}

// TODO(GODRIVER-251): Replace manual check with functionality of improved testing framework.
func skipIfBelow36(t *testing.T) {
	serverVersion, err := getServerVersion(createTestDatabase(t, nil))
	require.NoError(t, err)

	if compareVersions(t, serverVersion, "3.6") < 0 {
		t.Skip()
	}
}

func getNextChange(changes Cursor) {
	for !changes.Next(context.Background()) {
	}
}

func TestChangeStream_firstStage(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}
	skipIfBelow36(t)

	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	coll := createTestCollection(t, nil, nil)

	// Ensure the database is created.
	_, err := coll.InsertOne(context.Background(), bson.NewDocument(bson.EC.Int32("x", 1)))
	require.NoError(t, err)

	changes, err := coll.Watch(context.Background(), nil)
	require.NoError(t, err)

	elem, err := changes.(*changeStream).pipeline.Lookup(0)
	require.NoError(t, err)

	doc := elem.MutableDocument()
	require.Equal(t, 1, doc.Len())

	_, err = doc.LookupErr("$changeStream")
	require.NoError(t, err)
}

func TestChangeStream_noCustomStandaloneError(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}
	skipIfBelow36(t)

	topology := os.Getenv("TOPOLOGY")
	if topology == "replica_set" || topology == "sharded_cluster" {
		t.Skip()
	}

	coll := createTestCollection(t, nil, nil)

	// Ensure the database is created.
	_, err := coll.InsertOne(context.Background(), bson.NewDocument(bson.EC.Int32("x", 1)))
	require.NoError(t, err)

	_, err = coll.Watch(context.Background(), nil)
	require.Error(t, err)
	if _, ok := err.(command.Error); !ok {
		t.Errorf("Should have returned command error, but got %T", err)
	}
}

func TestChangeStream_trackResumeToken(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}
	skipIfBelow36(t)

	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	coll := createTestCollection(t, nil, nil)

	// Ensure the database is created.
	_, err := coll.InsertOne(context.Background(), bson.NewDocument(bson.EC.Int32("y", 1)))
	require.NoError(t, err)

	changes, err := coll.Watch(context.Background(), nil)
	require.NoError(t, err)

	for i := 1; i <= 4; i++ {
		_, err = coll.InsertOne(context.Background(), bson.NewDocument(bson.EC.Interface("x", i)))
		require.NoError(t, err)
	}

	for i := 1; i <= 4; i++ {
		getNextChange(changes)
		doc := bson.NewDocument()
		err := changes.Decode(doc)
		require.NoError(t, err)

		id, err := doc.LookupErr("_id")
		require.NoError(t, err)

		require.Equal(t, id.MutableDocument(), changes.(*changeStream).resumeToken)
	}
}

func TestChangeStream_errorMissingResponseToken(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}
	skipIfBelow36(t)

	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	coll := createTestCollection(t, nil, nil)

	// Ensure the database is created.
	_, err := coll.InsertOne(context.Background(), bson.NewDocument(bson.EC.Int32("y", 1)))
	require.NoError(t, err)

	// Project out the response token
	changes, err := coll.Watch(context.Background(), []*bson.Document{
		bson.NewDocument(
			bson.EC.SubDocumentFromElements("$project",
				bson.EC.Int32("_id", 0))),
	})
	require.NoError(t, err)

	_, err = coll.InsertOne(context.Background(), bson.NewDocument(bson.EC.Int32("x", 1)))
	require.NoError(t, err)

	getNextChange(changes)
	require.Error(t, changes.Decode(bson.NewDocument()))
}

func TestChangeStream_resumableError(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}
	skipIfBelow36(t)

	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	coll := createTestCollection(t, nil, nil)

	// Ensure the database is created.
	_, err := coll.InsertOne(context.Background(), bson.NewDocument(bson.EC.Int32("y", 1)))
	require.NoError(t, err)

	changes, err := coll.Watch(context.Background(), nil)
	require.NoError(t, err)

	// Create a context that will expire before the operation can finish.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Nanosecond)

	// "Use" the cancel function, which go vet complains if we throw away.
	func(context.CancelFunc) {}(cancel)

	require.False(t, changes.Next(ctx))

	err = changes.Err()
	require.Error(t, err)
	require.False(t, isServerError(err))

	// If the ResumeAfter option is present, the the operation attempted to resume.
	hasResume := false

	for _, opt := range changes.(*changeStream).options {
		if _, ok := opt.(option.OptResumeAfter); ok {
			hasResume = true
			break
		}
	}

	require.True(t, hasResume)
}

// TODO: GODRIVER-247 Test that a change stream does not attempt to resume after a server error.

func TestChangeStream_resumeAfterKillCursors(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip()
	}
	skipIfBelow36(t)

	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	coll := createTestCollection(t, nil, nil)

	// Ensure the database is created.
	_, err := coll.InsertOne(context.Background(), bson.NewDocument(bson.EC.Int32("y", 1)))
	require.NoError(t, err)

	changes, err := coll.Watch(context.Background(), nil)
	require.NoError(t, err)

	oldns := coll.namespace()
	killCursors := command.KillCursors{
		NS:  command.Namespace{DB: oldns.DB, Collection: oldns.Collection},
		IDs: []int64{changes.ID()},
	}

	ss, err := coll.client.topology.SelectServer(context.Background(), coll.readSelector)
	require.NoError(t, err)

	conn, err := ss.Connection(context.Background())
	require.NoError(t, err)
	defer conn.Close()

	_, err = killCursors.RoundTrip(context.Background(), ss.Description(), conn)
	require.NoError(t, err)

	require.False(t, changes.Next(context.Background()))
	require.NoError(t, changes.Err())

	_, err = coll.InsertOne(context.Background(), bson.NewDocument(bson.EC.Int32("x", 1)))
	require.NoError(t, err)

	getNextChange(changes)
	require.NoError(t, changes.Decode(bson.NewDocument()))
}
