// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// testping pings the deployment at MONGODB_URI and exits non-zero on failure.
// It is used to check connectivity from inside test containers.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
	log.Println("Ping succeeded")
}

func run() error {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		return fmt.Errorf("MONGODB_URI is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("error connecting client: %w", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	if err := client.Database("admin").RunCommand(ctx, bson.D{{"ping", 1}}).Err(); err != nil {
		return fmt.Errorf("error pinging deployment: %w", err)
	}
	return nil
}
