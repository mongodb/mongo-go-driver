package ops_test

import (
	"context"
	"testing"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/internal/testconfig"
	"github.com/10gen/mongo-go-driver/yamgo/private/cluster"
	"github.com/10gen/mongo-go-driver/yamgo/private/conn"
	"github.com/10gen/mongo-go-driver/yamgo/private/msg"
	. "github.com/10gen/mongo-go-driver/yamgo/private/ops"
	"github.com/10gen/mongo-go-driver/yamgo/readpref"
	"github.com/stretchr/testify/require"
)

func getServer(t *testing.T) *SelectedServer {

	c := testconfig.Cluster(t)

	server, err := c.SelectServer(context.Background(), cluster.WriteSelector(), readpref.Primary())
	require.NoError(t, err)

	return &SelectedServer{
		Server:   server,
		ReadPref: readpref.Primary(),
	}
}

func find(t *testing.T, s Server, batchSize int32) CursorResult {
	findCommand := bson.D{
		{"find", testconfig.ColName(t)},
	}
	if batchSize != 0 {
		findCommand = append(findCommand, bson.DocElem{"batchSize", batchSize})
	}
	request := msg.NewCommand(
		msg.NextRequestID(),
		testconfig.DBName(t),
		false,
		findCommand,
	)

	c, err := s.Connection(context.Background())
	require.NoError(t, err)
	defer c.Close()

	var result cursorReturningResult

	err = conn.ExecuteCommand(context.Background(), c, request, &result)
	require.NoError(t, err)

	return &result.Cursor
}

type cursorReturningResult struct {
	Cursor firstBatchCursorResult `bson:"cursor"`
}

type firstBatchCursorResult struct {
	FirstBatch []bson.Raw `bson:"firstBatch"`
	NS         string     `bson:"ns"`
	ID         int64      `bson:"id"`
}

func (cursorResult *firstBatchCursorResult) Namespace() Namespace {
	namespace := ParseNamespace(cursorResult.NS)
	return namespace
}

func (cursorResult *firstBatchCursorResult) InitialBatch() []bson.Raw {
	return cursorResult.FirstBatch
}

func (cursorResult *firstBatchCursorResult) CursorID() int64 {
	return cursorResult.ID
}
