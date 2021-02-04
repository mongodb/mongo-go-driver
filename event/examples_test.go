// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package event_test

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Event examples

// CommandMonitor represents a monitor that is triggered for different events.
func ExampleCommandMonitor() {
	// If the application makes multiple concurrent requests, it would have to
	// use a concurrent map like sync.Map
	startedCommands := make(map[int64]bson.Raw)
	cmdMonitor := &event.CommandMonitor{
		Started: func(_ context.Context, evt *event.CommandStartedEvent) {
			startedCommands[evt.RequestID] = evt.Command
		},
		Succeeded: func(_ context.Context, evt *event.CommandSucceededEvent) {
			log.Printf("Command: %v Reply: %v\n",
				startedCommands[evt.RequestID],
				evt.Reply,
			)
		},
		Failed: func(_ context.Context, evt *event.CommandFailedEvent) {
			log.Printf("Command: %v Failure: %v\n",
				startedCommands[evt.RequestID],
				evt.Failure,
			)
		},
	}
	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017").SetMonitor(cmdMonitor)
	client, err := mongo.Connect(context.Background(), clientOpts)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			log.Fatal(err)
		}
	}()
}
