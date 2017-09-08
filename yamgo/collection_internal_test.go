package yamgo

import (
	"fmt"
	"testing"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/internal/testconfig"
	"github.com/stretchr/testify/require"
)

func createTestCollection(t *testing.T, dbName *string, collName *string) *Collection {
	if collName == nil {
		coll := testconfig.ColName(t)
		collName = &coll
	}

	db := createTestDatabase(t, dbName)

	return db.Collection(*collName)
}

func initCollection(t *testing.T, coll *Collection) {
	doc1 := bson.D{bson.NewDocElem("x", 1)}
	doc2 := bson.D{bson.NewDocElem("x", 2)}
	doc3 := bson.D{bson.NewDocElem("x", 3)}
	doc4 := bson.D{bson.NewDocElem("x", 4)}
	doc5 := bson.D{bson.NewDocElem("x", 5)}

	var err error

	_, err = coll.InsertOne(doc1)
	require.Nil(t, err)

	_, err = coll.InsertOne(doc2)
	require.Nil(t, err)

	_, err = coll.InsertOne(doc3)
	require.Nil(t, err)

	_, err = coll.InsertOne(doc4)
	require.Nil(t, err)

	_, err = coll.InsertOne(doc5)
	require.Nil(t, err)
}

func TestCollection_initialize(t *testing.T) {
	t.Parallel()

	dbName := "foo"
	collName := "bar"

	coll := createTestCollection(t, &dbName, &collName)
	require.Equal(t, coll.name, collName)
	require.NotNil(t, coll.db)
}

func TestCollection_namespace(t *testing.T) {
	t.Parallel()

	dbName := "foo"
	collName := "bar"

	coll := createTestCollection(t, &dbName, &collName)
	namespace := coll.namespace()
	require.Equal(t, namespace.FullName(), fmt.Sprintf("%s.%s", dbName, collName))
}

func TestCollection_InsertOne(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	id := bson.NewObjectId()
	doc := bson.D{
		bson.NewDocElem("_id", id),
		bson.NewDocElem("x", 1),
	}
	coll := createTestCollection(t, nil, nil)

	result, err := coll.InsertOne(doc)
	require.Nil(t, err)
	require.Equal(t, result.InsertedID, id)
}

func TestCollection_DeleteOne_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.D{{Name: "x", Value: 1}}
	result, err := coll.DeleteOne(filter)
	require.Nil(t, err)
	require.Equal(t, result.DeletedCount, int64(1))
}

func TestCollection_DeleteOne_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.D{{Name: "x", Value: 0}}
	result, err := coll.DeleteOne(filter)
	require.Nil(t, err)
	require.Equal(t, result.DeletedCount, int64(0))
}

func TestCollection_UpdateOne_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.D{{Name: "x", Value: 1}}
	update := bson.M{
		"$inc": bson.M{
			"x": 1,
		},
	}

	result, err := coll.UpdateOne(filter, update)
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(1))
	require.Equal(t, result.ModifiedCount, int64(1))
	require.Nil(t, result.UpsertedID)
}

func TestCollection_UpdateOne_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.D{{Name: "x", Value: 0}}
	update := bson.M{
		"$inc": bson.M{
			"x": 1,
		},
	}

	result, err := coll.UpdateOne(filter, update)
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(0))
	require.Equal(t, result.ModifiedCount, int64(0))
	require.Nil(t, result.UpsertedID)
}

func TestCollection_UpdateOne_upsert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.D{{Name: "x", Value: 0}}
	update := bson.M{
		"$inc": bson.M{
			"x": 1,
		},
	}

	result, err := coll.UpdateOne(filter, update, Upsert(true))
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(1))
	require.Equal(t, result.ModifiedCount, int64(0))
	require.NotNil(t, result.UpsertedID)
}

func TestCollection_ReplaceOne_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.D{{Name: "x", Value: 1}}
	replacement := bson.D{{Name: "y", Value: 1}}

	result, err := coll.ReplaceOne(filter, replacement)
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(1))
	require.Equal(t, result.ModifiedCount, int64(1))
	require.Nil(t, result.UpsertedID)
}

func TestCollection_ReplaceOne_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.D{{Name: "x", Value: 0}}
	replacement := bson.D{{Name: "y", Value: 1}}

	result, err := coll.ReplaceOne(filter, replacement)
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(0))
	require.Equal(t, result.ModifiedCount, int64(0))
	require.Nil(t, result.UpsertedID)
}

func TestCollection_ReplaceOne_upsert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.D{{Name: "x", Value: 0}}
	replacement := bson.D{{Name: "y", Value: 1}}

	result, err := coll.ReplaceOne(filter, replacement, Upsert(true))
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(1))
	require.Equal(t, result.ModifiedCount, int64(0))
	require.NotNil(t, result.UpsertedID)
}
