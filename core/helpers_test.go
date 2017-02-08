package core_test

import (
	"github.com/10gen/mongo-go-driver/core"
	"github.com/10gen/mongo-go-driver/core/msg"
	"gopkg.in/mgo.v2/bson"
	"strings"
	"testing"
)

const databaseName = "mongo-go-driver"

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
	if err != nil {
		t.Fatal(err)
	}
}

func find(conn core.Connection, collectionName string, batchSize int32, t *testing.T) (core.CursorResult) {
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

	var result core.FindResult

	err := core.ExecuteCommand(conn, request, &result)
	if err != nil {
		t.Fatal(err)
	}

	return &result.Cursor
}

func dropCollection(conn core.Connection, collectionName string, t *testing.T) {
	err := core.ExecuteCommand(conn, msg.NewCommand(msg.NextRequestID(), databaseName, false, bson.D{{"drop", collectionName}}),
		&bson.D{})
	if err != nil && !strings.HasSuffix(err.Error(), "ns not found") {
		t.Fatal(err)
	}
}
