package main

import (
	"context"
	"errors"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestAWS(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err, "Connect error")

	defer func() {
		err = client.Disconnect(context.Background())
		require.NoError(t, err)
	}()

	coll := client.Database("aws").Collection("test")

	err = coll.FindOne(context.Background(), bson.D{{Key: "x", Value: 1}}).Err()
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		t.Logf("FindOne error: %v", err)
	}
}
