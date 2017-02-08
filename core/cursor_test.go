package core_test

import (
	"github.com/10gen/mongo-go-driver/core"
	"gopkg.in/mgo.v2/bson"
	"reflect"
	"testing"
)

func TestEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn, err := createIntegrationTestConnection()
	if err != nil {
		t.Fatal(err)
	}

	collectionName := "CursorTestEmpty"
	dropCollection(conn, collectionName, t)

	cursorResult := find(conn, collectionName, 0, t)

	subject := core.NewCursor(cursorResult, 0, conn)
	hasNext := subject.Next(&bson.D{})
	if hasNext {
		t.Fatal("Empty cursor should not have next")
	}
}

func TestSingleBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn, err := createIntegrationTestConnection()
	if err != nil {
		t.Fatal(err)
	}

	collectionName := "CursorTestSingleBatch"
	dropCollection(conn, collectionName, t)
	documents := []bson.D{{{"_id", 1}}, {{"_id", 2}}}
	insertDocuments(conn, collectionName, documents, t)

	cursorResult := find(conn, collectionName, 0, t)

	subject := core.NewCursor(cursorResult, 0, conn)
	var next bson.D
	var hasNext bool

	hasNext = subject.Next(&next)
	if !hasNext {
		t.Fatal("Should have result")
	}
	if !(reflect.DeepEqual(next, documents[0])) {
		t.Fatal("Documents not equal")
	}

	hasNext = subject.Next(&next)
	if !hasNext {
		t.Fatal("Should have result")
	}
	if !(reflect.DeepEqual(next, documents[1])) {
		t.Fatal("Documents not equal")
	}

	hasNext = subject.Next(&next)
	if hasNext {
		t.Fatal("Should not have result")
	}
}

func TestMultipleBatches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn, err := createIntegrationTestConnection()
	if err != nil {
		t.Fatal(err)
	}

	collectionName := "CursorTestMultipleBatches"
	dropCollection(conn, collectionName, t)
	documents := []bson.D{{{"_id", 1}}, {{"_id", 2}}, {{"_id", 3}}, {{"_id", 4}}, {{"_id", 5}}}
	insertDocuments(conn, collectionName, documents, t)

	cursorResult := find(conn, collectionName, 2, t)

	subject := core.NewCursor(cursorResult, 2, conn)
	var next bson.D
	var hasNext bool

	hasNext = subject.Next(&next)
	if !hasNext {
		t.Fatal("Should have result")
	}
	if !(reflect.DeepEqual(next, documents[0])) {
		t.Fatal("Documents not equal")
	}

	hasNext = subject.Next(&next)
	if !hasNext {
		t.Fatal("Should have result")
	}
	if !(reflect.DeepEqual(next, documents[1])) {
		t.Fatal("Documents not equal")
	}

	hasNext = subject.Next(&next)
	if !hasNext {
		t.Fatal("Should have result")
	}
	if !(reflect.DeepEqual(next, documents[2])) {
		t.Fatal("Documents not equal")
	}

	hasNext = subject.Next(&next)
	if !hasNext {
		t.Fatal("Should have result")
	}
	if !(reflect.DeepEqual(next, documents[3])) {
		t.Fatal("Documents not equal")
	}

	hasNext = subject.Next(&next)
	if !hasNext {
		t.Fatal("Should have result")
	}
	if !(reflect.DeepEqual(next, documents[4])) {
		t.Fatal("Documents not equal")
	}

	hasNext = subject.Next(&next)
	if hasNext {
		t.Fatal("Should not have result")
	}
}

func TestClose(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn, err := createIntegrationTestConnection()
	if err != nil {
		t.Fatal(err)
	}

	collectionName := "CursorTestClose"
	dropCollection(conn, collectionName, t)
	documents := []bson.D{{{"_id", 1}}, {{"_id", 2}}, {{"_id", 3}}, {{"_id", 4}}, {{"_id", 5}}}
	insertDocuments(conn, collectionName, documents, t)

	cursorResult := find(conn, collectionName, 2, t)

	subject := core.NewCursor(cursorResult, 2, conn)
	err = subject.Close()
	if err != nil {
		t.Fatal("Did not expect error")
	}
	// call it again
	err = subject.Close()
	if err != nil {
		t.Fatal("Did not expect error")
	}
}

func TestError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn, err := createIntegrationTestConnection()
	if err != nil {
		t.Fatal(err)
	}

	collectionName := "CursorTestError"
	dropCollection(conn, collectionName, t)
	documents := []bson.D{{{"_id", 1}}, {{"_id", 2}}, {{"_id", 3}}, {{"_id", 4}}, {{"_id", 5}}}
	insertDocuments(conn, collectionName, documents, t)

	cursorResult := find(conn, collectionName, 2, t)

	subject := core.NewCursor(cursorResult, 2, conn)
	var next string
	var hasNext bool

	hasNext = subject.Next(&next)
	if subject.Err() == nil {
		t.Fatal("Expected error")
	}
	if hasNext {
		t.Fatal("Should not have result")
	}
}
