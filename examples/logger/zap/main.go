package main

import (
	"context"
	"log"

	"github.com/go-logr/zapr"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("error creating zap logger: %w", err)
	}

	sink := zapr.NewLogger(logger).GetSink()

	// Create a client with our logger options.
	loggerOptions := options.
		Logger().
		SetSink(sink).
		SetMaxDocumentLength(25).
		SetComponentLevel(options.LogComponentCommand, options.LogLevelDebug)

	clientOptions := options.
		Client().
		ApplyURI("mongodb://localhost:27017").
		SetLoggerOptions(loggerOptions)

	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatalf("error connecting to MongoDB: %v", err)
	}

	defer client.Disconnect(context.TODO())

	// Make a databse request to test our logging solution
	coll := client.Database("test").Collection("test")

	_, err = coll.InsertOne(context.TODO(), bson.D{{"Alice", "123"}})
	if err != nil {
		log.Fatalf("InsertOne failed: %v", err)
	}
}
