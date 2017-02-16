package ops_test

import (
	"context"
	"testing"
	"time"

	. "github.com/10gen/mongo-go-driver/ops"
	"github.com/stretchr/testify/require"
	"gopkg.in/mgo.v2/bson"
)

func TestAggregateWithInvalidNamespace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	_, err := Aggregate(
		context.Background(),
		getServer(),
		Namespace{},
		[]bson.D{},
		AggregationOptions{})
	require.NotNil(t, err)
}

func TestAggregateWithMultipleBatches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	server := getServer()

	collectionName := "TestAggregateWithMultipleBatches"
	documents := []bson.D{{{"_id", 1}}, {{"_id", 2}}, {{"_id", 3}}, {{"_id", 4}}, {{"_id", 5}}}
	insertDocuments(server, collectionName, documents, t)

	namespace := Namespace{databaseName, collectionName}
	cursor, err := Aggregate(context.Background(), server, namespace,
		[]bson.D{
			{{"$match", bson.D{{"_id", bson.D{{"$gt", 2}}}}}},
			{{"$sort", bson.D{{"_id", -1}}}}},
		AggregationOptions{
			BatchSize: 2})
	require.Nil(t, err)

	var next bson.D

	cursor.Next(context.Background(), &next)
	require.Equal(t, documents[4], next)

	cursor.Next(context.Background(), &next)
	require.Equal(t, documents[3], next)

	cursor.Next(context.Background(), &next)
	require.Equal(t, documents[2], next)

	hasNext := cursor.Next(context.Background(), &next)
	require.False(t, hasNext)
}

// This is not a great test since there are no visible side effects of allowDiskUse, and there server does not currently
// check the validity of field names for the aggregate command
func TestAggregateWithAllowDiskUse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	server := getServer()

	collectionName := "TestAggregateWithAllowDiskUse"
	documents := []bson.D{{{"_id", 1}}, {{"_id", 2}}}
	insertDocuments(server, collectionName, documents, t)

	namespace := Namespace{databaseName, collectionName}
	_, err := Aggregate(context.Background(), server, namespace,
		[]bson.D{},
		AggregationOptions{
			AllowDiskUse: true})
	require.Nil(t, err)
}

func TestAggregateWithMaxTimeMS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	server := getServer()

	if enableMaxTimeFailPoint(server, t) != nil {
		t.Skip("skipping maxTimeMS test when max time failpoint is disabled")
	}
	defer disableMaxTimeFailPoint(server, t)

	collectionName := "TestAggregateWithAllowDiskUse"
	namespace := Namespace{databaseName, collectionName}
	_, err := Aggregate(context.Background(), server, namespace,
		[]bson.D{},
		AggregationOptions{
			MaxTime: time.Millisecond})
	require.NotNil(t, err)

	// Hacky check for the error message.  Should we be returning a more structured error?
	require.Contains(t, err.Error(), "operation exceeded time limit")
}
