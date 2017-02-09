package ops_test

import (
	. "github.com/10gen/mongo-go-driver/core"
	. "github.com/10gen/mongo-go-driver/ops"
	"gopkg.in/mgo.v2/bson"
	"reflect"
	"strings"
	"testing"
)

func TestAggregateWithMultipleBatches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn, err := createIntegrationTestConnection()
	if err != nil {
		t.Fatal(err)
	}

	collectionName := "TestAggregateWithMultipleBatches"
	documents := []bson.D{{{"_id", 1}}, {{"_id", 2}}, {{"_id", 3}}, {{"_id", 4}}, {{"_id", 5}}}
	insertDocuments(conn, collectionName, documents, t)

	cursor, err := Aggregate(conn, NewNamespaceFromDatabaseAndCollection(databaseName, collectionName),
		[]bson.D{
			{{"$match", bson.D{{"_id", bson.D{{"$gt", 2}}}}}},
			{{"$sort", bson.D{{"_id", -1}}}}},
		&AggregationOptions{
			BatchSize: 2})
	if err != nil {
		t.Fatalf("Error: %v\n", err)
	}

	var next bson.D

	cursor.Next(&next)
	if !reflect.DeepEqual(documents[4], next) {
		t.Fatal("Expected documents to be equals")
	}
	cursor.Next(&next)
	if !reflect.DeepEqual(documents[3], next) {
		t.Fatal("Expected documents to be equals")
	}

	cursor.Next(&next)
	if !reflect.DeepEqual(documents[2], next) {
		t.Fatal("Expected documents to be equals")
	}

	hasNext := cursor.Next(&next)
	if hasNext {
		t.Fatal("Expected end of cursor")
	}
}

// This is not a great test since there are no visible side effects of allowDiskUse, and there server does not currently
// check the validity of field names for the aggregate command
func TestAggregateWithAllowDiskUse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn, err := createIntegrationTestConnection()
	if err != nil {
		t.Fatal(err)
	}

	collectionName := "TestAggregateWithAllowDiskUse"
	documents := []bson.D{{{"_id", 1}}, {{"_id", 2}}}
	insertDocuments(conn, collectionName, documents, t)

	_, err = Aggregate(conn, NewNamespaceFromDatabaseAndCollection(databaseName, collectionName),
		[]bson.D{},
		&AggregationOptions{
			AllowDiskUse: true})
	if err != nil {
		t.Fatalf("Error: %v\n", err)
	}
}

func TestAggregateWithMaxTimeMS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn, err := createIntegrationTestConnection()
	if err != nil {
		t.Fatal(err)
	}

	enableMaxTimeFailPoint(conn, t)
	defer disableMaxTimeFailPoint(conn, t)

	collectionName := "TestAggregateWithAllowDiskUse"
	_, err = Aggregate(conn, NewNamespaceFromDatabaseAndCollection(databaseName, collectionName),
		[]bson.D{},
		&AggregationOptions{
			MaxTimeMS: 1})
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
	// Hacky check for the error message.  Should we be returning a more structured error?
	if !strings.Contains(err.Error(), "operation exceeded time limit") {
		t.Fatalf("Expected execution timeout error: %v\n", err)
	}
}
