// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const timeout = 60 * time.Second

// eventListener supports command, heartbeat, and pool event handlers to record
// event durations, as well as the number of heartbeats, commands, and open
// connections.
type eventListener struct {
	commandCount          int
	commandDuration       int64
	heartbeatAwaitedCount int
	heartbeatCount        int
	heartbeatDuration     int64
	openConnections       int
}

// commandMonitor initializes an event.CommandMonitor that will count the number
// of successful or failed command events and record a running duration of these
// events.
func (listener *eventListener) commandMonitor() *event.CommandMonitor {
	succeeded := func(_ context.Context, e *event.CommandSucceededEvent) {
		listener.commandCount++
		listener.commandDuration += e.DurationNanos
	}

	failed := func(_ context.Context, e *event.CommandFailedEvent) {
		listener.commandCount++
		listener.commandDuration += e.DurationNanos
	}

	return &event.CommandMonitor{
		Succeeded: succeeded,
		Failed:    failed,
	}
}

// severMonitor initializes an event.ServerMonitor that will count the number
// of successful or failed heartbeat events and record a running duration of
// these events.
func (listener *eventListener) serverMonitor() *event.ServerMonitor {
	succeeded := func(e *event.ServerHeartbeatSucceededEvent) {
		listener.heartbeatCount++
		listener.heartbeatDuration += e.DurationNanos

		if e.Awaited {
			listener.heartbeatAwaitedCount++
		}
	}

	failed := func(e *event.ServerHeartbeatFailedEvent) {
		listener.heartbeatCount++
		listener.heartbeatDuration += e.DurationNanos

		if e.Awaited {
			listener.heartbeatAwaitedCount++
		}
	}

	return &event.ServerMonitor{
		ServerHeartbeatSucceeded: succeeded,
		ServerHeartbeatFailed:    failed,
	}
}

// poolMonitor initialize an event.PoolMonitor that will increment each time a
// new connection is created and decrement each time a connection is closed.
func (listener *eventListener) poolMonitor() *event.PoolMonitor {
	poolEvent := func(e *event.PoolEvent) {
		switch e.Type {
		case event.ConnectionCreated:
			listener.openConnections++
		case event.ConnectionClosed:
			listener.openConnections--
		}
	}

	return &event.PoolMonitor{Event: poolEvent}
}

// response is the data we return in the body of the API Gateway response.
type response struct {
	AvgCommandDuration   float64 `json:"averageCommandDuration"`
	AvgHeartbeatDuration float64 `json:"averageHeartbeatDuration"`
	OpenConnections      int     `json:"openConnections"`
	HeartbeatCount       int     `json:"heartbeatCount"`
}

// gateway500 is a convenience function for constructing a gateway response with
// a 500 status code, indicating an internal server error.
func gateway500() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       http.StatusText(http.StatusInternalServerError),
	}

}

// handler is the AWS Lambda handler, executing at runtime.
func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	listener := new(eventListener)

	clientOptions := options.Client().ApplyURI(os.Getenv("MONGODB_URI")).
		SetMonitor(listener.commandMonitor()).
		SetServerMonitor(listener.serverMonitor()).
		SetPoolMonitor(listener.poolMonitor())

	// Create a MongoClient that points to MONGODB_URI and listens to the
	// ComandMonitor, ServerMonitor, and PoolMonitor events.
	client, err := mongo.NewClient(clientOptions)
	if err != nil {
		return gateway500(), fmt.Errorf("failed to create client: %w", err)
	}

	// Attempt to connect to the client with a timeout.
	if err = client.Connect(ctx); err != nil {
		return gateway500(), fmt.Errorf("failed to connect: %w", err)
	}

	defer client.Disconnect(ctx)

	collection := client.Database("faas").Collection("lambda")

	// Create a document to insert for the automated test.
	doc := map[string]string{"hello": "world"}

	// Insert the document.
	_, err = collection.InsertOne(ctx, doc)
	if err != nil {
		return gateway500(), fmt.Errorf("failed to insert: %w", err)
	}

	// Delete the document.
	_, err = collection.DeleteOne(ctx, doc)
	if err != nil {
		return gateway500(), fmt.Errorf("failed to delete: %w", err)
	}

	// Driver must switch to polling monitoring when running within a FaaS
	// environment.
	if listener.heartbeatAwaitedCount > 0 {
		return gateway500(), fmt.Errorf("FaaS environment failed to switch to polling")
	}

	var avgCmdDur float64
	if count := listener.commandCount; count != 0 {
		avgCmdDur = float64(listener.commandDuration) / float64(count)
	}

	var avgHBDur float64
	if count := listener.heartbeatCount; count != 0 {
		avgHBDur = float64(listener.heartbeatDuration) / float64(count)
	}

	rsp := &response{
		AvgCommandDuration:   avgCmdDur,
		AvgHeartbeatDuration: avgHBDur,
		OpenConnections:      listener.openConnections,
		HeartbeatCount:       listener.heartbeatCount,
	}

	body, err := json.Marshal(rsp)
	if err != nil {
		return gateway500(), fmt.Errorf("failed to marshal: %w", err)
	}

	return events.APIGatewayProxyResponse{
		Body:       string(body),
		StatusCode: http.StatusOK,
	}, nil
}

func main() {
	ctx := context.Background()

	lambda.StartWithOptions(handler, lambda.WithContext(ctx))
}
