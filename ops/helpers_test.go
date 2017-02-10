package ops_test

import (
	"flag"
	"fmt"
	"strings"
	"testing"

	"github.com/10gen/mongo-go-driver/core"
	"github.com/10gen/mongo-go-driver/core/desc"
	"github.com/10gen/mongo-go-driver/core/msg"
	. "github.com/10gen/mongo-go-driver/ops"
	"github.com/stretchr/testify/require"
	"gopkg.in/mgo.v2/bson"
)

var host = flag.String("host", "127.0.0.1:27017", "specify the location of a running mongodb server.")

const databaseName = "mongo-go-driver"

var conn core.Connection

func getConnection() core.Connection {
	if conn == nil {
		var err error
		conn, err = core.DialConnection(core.ConnectionOptions{
			AppName:        "mongo-go-driver-test",
			Codec:          msg.NewWireProtocolCodec(),
			Endpoint:       desc.Endpoint(*host),
			EndpointDialer: core.DialEndpoint,
		})
		if err != nil {
			panic(fmt.Errorf("failed dialing mongodb server - ensure that one is running at %s: %v", *host, err))
		}
	}
	return conn
}

func insertDocuments(conn core.Connection, collectionName string, documents []bson.D, t *testing.T) {
	insertCommand := bson.D{
		{"insert", collectionName},
		{"documents", documents},
	}
	request := msg.NewCommand(
		msg.NextRequestID(),
		databaseName,
		false,
		insertCommand,
	)

	result := &bson.D{}

	err := core.ExecuteCommand(conn, request, result)
	require.Nil(t, err)
}

func find(conn core.Connection, collectionName string, batchSize int32, t *testing.T) CursorResult {
	findCommand := bson.D{
		{"find", collectionName},
	}
	if batchSize != 0 {
		findCommand = append(findCommand, bson.DocElem{"batchSize", batchSize})
	}
	request := msg.NewCommand(
		msg.NextRequestID(),
		databaseName,
		false,
		findCommand,
	)

	var result cursorReturningResult

	err := core.ExecuteCommand(conn, request, &result)
	require.Nil(t, err)

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

func (cursorResult *firstBatchCursorResult) CursorId() int64 {
	return cursorResult.ID
}

func dropCollection(conn core.Connection, collectionName string, t *testing.T) {
	err := core.ExecuteCommand(conn, msg.NewCommand(msg.NextRequestID(), databaseName, false, bson.D{{"drop", collectionName}}),
		&bson.D{})
	if err != nil && !strings.HasSuffix(err.Error(), "ns not found") {
		t.Fatal(err)
	}
}

func enableMaxTimeFailPoint(conn core.Connection) error {
	return core.ExecuteCommand(conn, msg.NewCommand(msg.NextRequestID(), "admin", false,
		bson.D{{"configureFailPoint", "maxTimeAlwaysTimeOut"},
			{"mode", "alwaysOn"}}),
		&bson.D{})
}

func disableMaxTimeFailPoint(conn core.Connection, t *testing.T) {
	err := core.ExecuteCommand(conn, msg.NewCommand(msg.NextRequestID(), "admin", false,
		bson.D{{"configureFailPoint", "maxTimeAlwaysTimeOut"},
			{"mode", "off"}}),
		&bson.D{})
	require.Nil(t, err)
}
