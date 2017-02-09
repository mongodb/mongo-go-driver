package ops_test

import (
	. "github.com/10gen/mongo-go-driver/ops"
	"gopkg.in/mgo.v2/bson"
	"testing"
	"strings"
)

func TestListCollections(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn, err := createIntegrationTestConnection()
	if err != nil {
		t.Fatal(err)
	}

	collectionNameOne := "TestListCollectionsMultipleBatches1"
	collectionNameTwo := "TestListCollectionsMultipleBatches2"
	collectionNameThree := "TestListCollectionsMultipleBatches3"

	dropCollection(conn, collectionNameOne, t)
	dropCollection(conn, collectionNameTwo, t)
	dropCollection(conn, collectionNameThree, t)

	insertDocuments(conn, collectionNameOne, []bson.D{{{"_id", 1}}}, t)
	insertDocuments(conn, collectionNameTwo, []bson.D{{{"_id", 1}}}, t)
	insertDocuments(conn, collectionNameThree, []bson.D{{{"_id", 1}}}, t)

	cursor, err := ListCollections(conn, databaseName, &ListCollectionsOptions{})

	names := []string{}
	var next bson.M

	for cursor.Next(&next) {
		names = append(names, next["name"].(string))
	}

	if !contains(names, collectionNameOne) {
		t.Fatalf("Expected collection %v", collectionNameOne)
	}
	if !contains(names, collectionNameTwo) {
		t.Fatalf("Expected collection %v", collectionNameTwo)
	}
	if !contains(names, collectionNameThree) {
		t.Fatalf("Expected collection %v", collectionNameThree)
	}
}

func TestListCollectionsMultipleBatches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn, err := createIntegrationTestConnection()
	if err != nil {
		t.Fatal(err)
	}

	collectionNameOne := "TestListCollectionsMultipleBatches1"
	collectionNameTwo := "TestListCollectionsMultipleBatches2"
	collectionNameThree := "TestListCollectionsMultipleBatches3"

	dropCollection(conn, collectionNameOne, t)
	dropCollection(conn, collectionNameTwo, t)
	dropCollection(conn, collectionNameThree, t)

	insertDocuments(conn, collectionNameOne, []bson.D{{{"_id", 1}}}, t)
	insertDocuments(conn, collectionNameTwo, []bson.D{{{"_id", 1}}}, t)
	insertDocuments(conn, collectionNameThree, []bson.D{{{"_id", 1}}}, t)

	cursor, err := ListCollections(conn, databaseName, &ListCollectionsOptions{
		Filter:    bson.D{{"name", bson.RegEx{Pattern: "^TestListCollectionsMultipleBatches.*"}}},
		BatchSize: 2})

	names := []string{}
	var next bson.M

	for cursor.Next(&next) {
		names = append(names, next["name"].(string))
	}

	if len(names) != 3 {
		t.Fatalf("Expected 3 collections but received %d", len(names))
	}
	if !contains(names, collectionNameOne) {
		t.Fatalf("Expected collection %v", collectionNameOne)
	}
	if !contains(names, collectionNameTwo) {
		t.Fatalf("Expected collection %v", collectionNameTwo)
	}
	if !contains(names, collectionNameThree) {
		t.Fatalf("Expected collection %v", collectionNameThree)
	}
}

func TestListCollectionsWithMaxTimeMS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn, err := createIntegrationTestConnection()
	if err != nil {
		t.Fatal(err)
	}

	enableMaxTimeFailPoint(conn, t)
	defer disableMaxTimeFailPoint(conn, t)

	_, err = ListCollections(conn, databaseName, &ListCollectionsOptions{
		MaxTimeMS: 1,
	})
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
	// Hacky check for the error message.  Should we be returning a more structured error?
	if !strings.Contains(err.Error(), "operation exceeded time limit") {
		t.Fatalf("Expected execution timeout error: %v\n", err)
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
