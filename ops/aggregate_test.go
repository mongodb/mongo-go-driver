package ops_test

import (
	"testing"
	"time"

	"github.com/10gen/mongo-go-driver/core"
	. "github.com/10gen/mongo-go-driver/ops"
	"github.com/stretchr/testify/require"
	"gopkg.in/mgo.v2/bson"
)

func TestAggregateWithMultipleBatches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn := getConnection()

	collectionName := "TestAggregateWithMultipleBatches"
	documents := []bson.D{{{"_id", 1}}, {{"_id", 2}}, {{"_id", 3}}, {{"_id", 4}}, {{"_id", 5}}}
	insertDocuments(conn, collectionName, documents, t)

	namespace, _ := core.NewNamespace(databaseName, collectionName)
	cursor, err := Aggregate(conn, *namespace,
		[]bson.D{
			{{"$match", bson.D{{"_id", bson.D{{"$gt", 2}}}}}},
			{{"$sort", bson.D{{"_id", -1}}}}},
		AggregationOptions{
			BatchSize: 2})
	require.Nil(t, err)

	var next bson.D

	cursor.Next(&next)
	require.Equal(t, documents[4], next)

	cursor.Next(&next)
	require.Equal(t, documents[3], next)

	cursor.Next(&next)
	require.Equal(t, documents[2], next)

	hasNext := cursor.Next(&next)
	require.False(t, hasNext)
}

// This is not a great test since there are no visible side effects of allowDiskUse, and there server does not currently
// check the validity of field names for the aggregate command
func TestAggregateWithAllowDiskUse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn := getConnection()

	collectionName := "TestAggregateWithAllowDiskUse"
	documents := []bson.D{{{"_id", 1}}, {{"_id", 2}}}
	insertDocuments(conn, collectionName, documents, t)

	namespace, _ := core.NewNamespace(databaseName, collectionName)
	_, err := Aggregate(conn, *namespace,
		[]bson.D{},
		AggregationOptions{
			AllowDiskUse: true})
	require.Nil(t, err)
}

func TestAggregateWithMaxTimeMS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn := getConnection()

	enableMaxTimeFailPoint(conn, t)
	defer disableMaxTimeFailPoint(conn, t)

	collectionName := "TestAggregateWithAllowDiskUse"
	namespace, _ := core.NewNamespace(databaseName, collectionName)
	_, err := Aggregate(conn, *namespace,
		[]bson.D{},
		AggregationOptions{
			MaxTime: time.Millisecond})
	require.NotNil(t, err)

	// Hacky check for the error message.  Should we be returning a more structured error?
	require.Contains(t, err.Error(), "operation exceeded time limit")
}
