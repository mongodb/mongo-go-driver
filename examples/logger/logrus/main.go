package main

import (
	"context"
	"log"

	"github.com/bombsimon/logrusr/v4"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Create a new logrus logger instance.
	logger := logrus.StandardLogger()
	logger.SetLevel(logrus.DebugLevel)

	// Create a new sink for logrus using "logrusr".
	sink := logrusr.New(logger).GetSink()

	// Create a client with our logger options.
	loggerOptions := options.
		Logger().
		SetSink(sink).
		SetMaxDocumentLength(25).
		SetComponentLevel(options.CommandLogComponent, options.DebugLogLevel)

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
