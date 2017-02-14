package ops_test

import (
	"flag"
	"fmt"
	"strings"
	"testing"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/msg"
	. "github.com/10gen/mongo-go-driver/ops"
	"github.com/stretchr/testify/require"
	"gopkg.in/mgo.v2/bson"
)

var host = flag.String("host", "127.0.0.1:27017", "specify the location of a running mongodb server.")

const databaseName = "mongo-go-driver"

var testConn conn.Connection

func getConnection() conn.Connection {
	if testConn == nil {
		var err error
		testConn, err = conn.Dial(
			conn.Endpoint(*host),
			conn.WithAppName("mongo-go-driver-test"),
		)
		if err != nil {
			panic(fmt.Errorf("failed dialing mongodb server - ensure that one is running at %s: %v", *host, err))
		}
	}
	return testConn
}

func insertDocuments(c conn.Connection, collectionName string, documents []bson.D, t *testing.T) {
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

	err := conn.ExecuteCommand(c, request, result)
	require.Nil(t, err)
}

func find(c conn.Connection, collectionName string, batchSize int32, t *testing.T) CursorResult {
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

	err := conn.ExecuteCommand(c, request, &result)
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

func (cursorResult *firstBatchCursorResult) CursorID() int64 {
	return cursorResult.ID
}

func dropCollection(c conn.Connection, collectionName string, t *testing.T) {
	err := conn.ExecuteCommand(c, msg.NewCommand(msg.NextRequestID(), databaseName, false, bson.D{{"drop", collectionName}}),
		&bson.D{})
	if err != nil && !strings.HasSuffix(err.Error(), "ns not found") {
		t.Fatal(err)
	}
}

func enableMaxTimeFailPoint(c conn.Connection) error {
	return conn.ExecuteCommand(c, msg.NewCommand(msg.NextRequestID(), "admin", false,
		bson.D{{"configureFailPoint", "maxTimeAlwaysTimeOut"},
			{"mode", "alwaysOn"}}),
		&bson.D{})
}

func disableMaxTimeFailPoint(c conn.Connection, t *testing.T) {
	err := conn.ExecuteCommand(c, msg.NewCommand(msg.NextRequestID(), "admin", false,
		bson.D{{"configureFailPoint", "maxTimeAlwaysTimeOut"},
			{"mode", "off"}}),
		&bson.D{})
	require.Nil(t, err)
}
