package ops_test

import (
	. "github.com/10gen/mongo-go-driver/ops"
	"gopkg.in/mgo.v2/bson"
	"testing"
	"strings"
)

func TestListDatabases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn, err := createIntegrationTestConnection()
	if err != nil {
		t.Fatal(err)
	}

	collectionName := "TestListDatabases"
	dropCollection(conn, collectionName, t)
	insertDocuments(conn, collectionName, []bson.D{{{"_id", 1}}}, t)

	cursor, err := ListDatabases(conn, &ListDatabasesOptions{})
	if err != nil {
		t.Fatalf("Error %v", err)
	}


	var next bson.M
	var found bool
	for cursor.Next(&next) {
		if next["name"] == databaseName {
			found = true
		}
	}
	if !found {
		t.Fatalf("Expected to have listed at least database named %v", databaseName)
	}

	if cursor.Err() != nil {
		t.Fatal("Unexpected error")
	}

	if cursor.Close() != nil {
		t.Fatal("Unexpected error on close")

	}
}

func TestListDatabasesWithMaxTimeMS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn, err := createIntegrationTestConnection()
	if err != nil {
		t.Fatal(err)
	}

	enableMaxTimeFailPoint(conn, t)
	defer disableMaxTimeFailPoint(conn, t)

	_, err = ListDatabases(conn, &ListDatabasesOptions{
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
