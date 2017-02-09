package ops_test

import (
	"flag"
	"fmt"
	. "github.com/10gen/mongo-go-driver/core"
	"github.com/10gen/mongo-go-driver/core/msg"
	. "github.com/10gen/mongo-go-driver/ops"
	"github.com/stretchr/testify/require"
	"gopkg.in/mgo.v2/bson"
	"strings"
	"testing"
)

var host = flag.String("host", "127.0.0.1:27017", "specify the location of a running mongodb server.")

const databaseName = "mongo-go-driver"

var conn Connection

func getConnection() Connection {
	if conn == nil {
		var err error
		conn, err = DialConnection(ConnectionOptions{
			AppName:        "mongo-go-driver-test",
			Codec:          msg.NewWireProtocolCodec(),
			Endpoint:       Endpoint(*host),
			EndpointDialer: DialEndpoint,
		})
		if err != nil {
			panic(fmt.Errorf("failed dialing mongodb server - ensure that one is running at %s: %v", *host, err))
		}
	}
	return conn
}

func insertDocuments(conn Connection, collectionName string, documents []bson.D, t *testing.T) {
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

	err := ExecuteCommand(conn, request, result)
	require.Nil(t, err)
}

func find(conn Connection, collectionName string, batchSize int32, t *testing.T) CursorResult {
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

	err := ExecuteCommand(conn, request, &result)
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

func (cursorResult *firstBatchCursorResult) Namespace() *Namespace {
	return NewNamespace(cursorResult.NS)
}

func (cursorResult *firstBatchCursorResult) InitialBatch() []bson.Raw {
	return cursorResult.FirstBatch
}

func (cursorResult *firstBatchCursorResult) CursorId() int64 {
	return cursorResult.ID
}

func dropCollection(conn Connection, collectionName string, t *testing.T) {
	err := ExecuteCommand(conn, msg.NewCommand(msg.NextRequestID(), databaseName, false, bson.D{{"drop", collectionName}}),
		&bson.D{})
	if err != nil && !strings.HasSuffix(err.Error(), "ns not found") {
		t.Fatal(err)
	}
}

func enableMaxTimeFailPoint(conn Connection, t *testing.T) {
	err := ExecuteCommand(conn, msg.NewCommand(msg.NextRequestID(), "admin", false,
		bson.D{{"configureFailPoint", "maxTimeAlwaysTimeOut"},
			{"mode", "alwaysOn"}}),
		&bson.D{})
	require.Nil(t, err)
}

func disableMaxTimeFailPoint(conn Connection, t *testing.T) {
	err := ExecuteCommand(conn, msg.NewCommand(msg.NextRequestID(), "admin", false,
		bson.D{{"configureFailPoint", "maxTimeAlwaysTimeOut"},
			{"mode", "off"}}),
		&bson.D{})
	require.Nil(t, err)
}
