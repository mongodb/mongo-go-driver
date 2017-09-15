package ops_test

import (
	"context"
	"testing"
	"time"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/internal/testconfig"
	. "github.com/10gen/mongo-go-driver/yamgo/private/ops"
	"github.com/stretchr/testify/require"
)

func TestAggregateWithInvalidNamespace(t *testing.T) {
	t.Parallel()
	testconfig.Integration(t)

	_, err := Aggregate(
		context.Background(),
		getServer(t),
		Namespace{},
		[]bson.D{},
		AggregationOptions{})
	require.Error(t, err)
}

func TestAggregateWithMultipleBatches(t *testing.T) {
	t.Parallel()
	testconfig.Integration(t)
	documents := []bson.D{{{"_id", 1}}, {{"_id", 2}}, {{"_id", 3}}, {{"_id", 4}}, {{"_id", 5}}}
	testconfig.AutoInsertDocs(t, documents...)

	server := getServer(t)
	namespace := Namespace{testconfig.DBName(t), testconfig.ColName(t)}
	cursor, err := Aggregate(context.Background(), server, namespace,
		[]bson.D{
			{{"$match", bson.D{{"_id", bson.D{{"$gt", 2}}}}}},
			{{"$sort", bson.D{{"_id", -1}}}}},
		AggregationOptions{
			BatchSize: 2})
	require.NoError(t, err)

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
	t.Parallel()
	testconfig.Integration(t)
	testconfig.AutoInsertDocs(t,
		bson.D{{"_id", 1}},
		bson.D{{"_id", 2}},
	)

	server := getServer(t)
	namespace := Namespace{testconfig.DBName(t), testconfig.ColName(t)}
	_, err := Aggregate(context.Background(), server, namespace,
		[]bson.D{},
		AggregationOptions{
			AllowDiskUse: true})
	require.NoError(t, err)
}

func TestAggregateWithMaxTimeMS(t *testing.T) {
	t.Skip("max time is flaky on the server")
	t.Parallel()
	testconfig.Integration(t)

	s := getServer(t)

	if testconfig.EnableMaxTimeFailPoint(t, s) != nil {
		t.Skip("skipping maxTimeMS test when max time failpoint is disabled")
	}
	defer testconfig.DisableMaxTimeFailPoint(t, s)

	namespace := Namespace{testconfig.DBName(t), testconfig.ColName(t)}
	_, err := Aggregate(context.Background(), s, namespace,
		[]bson.D{},
		AggregationOptions{
			MaxTime: time.Millisecond})
	require.Error(t, err)

	// Hacky check for the error message.  Should we be returning a more structured error?
	require.Contains(t, err.Error(), "operation exceeded time limit")
}
