// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package container

import (
	"context"
	"regexp"
	"strconv"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"

	mongooptions "go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoOptions struct {
	mongoClientOpts *mongooptions.ClientOptions
	image           string
}

// MongoOption is a function that configures NewMongo.
type MongoOption func(*mongoOptions)

// WithMongoClientOptions configures the mongo.Client options used to connect.
func WithMongoClientOptions(opts *mongooptions.ClientOptions) MongoOption {
	return func(o *mongoOptions) {
		o.mongoClientOpts = opts
	}
}

// WithMongoImage configures the Docker image used for the MongoDB container.
func WithMongoImage(image string) MongoOption {
	return func(o *mongoOptions) {
		o.image = image
	}
}

// NewMongo creates a new MongoDB test container and returns a connected
// mongo.Client and a TeardownFunc to clean up resources.
func NewMongo(t *testing.T, ctx context.Context, optionFuncs ...MongoOption) (*mongo.Client, TeardownFunc) {
	t.Helper()

	opts := &mongoOptions{}
	for _, apply := range optionFuncs {
		apply(opts)
	}

	image := "mongo:latest"
	if opts.image != "" {
		image = opts.image
	}

	var containerOpts []testcontainers.ContainerCustomizer
	if needsCustomWaitStrategy(image) {
		waitStrategy := wait.ForAll(
			wait.ForLog("(?i)waiting for connections").AsRegexp().WithOccurrence(1),
			wait.ForListeningPort("27017/tcp"),
		)
		containerOpts = append(containerOpts, testcontainers.WithWaitStrategy(waitStrategy))
	}

	mongolocalContainer, err := mongodb.Run(ctx, image, containerOpts...)
	require.NoError(t, err, "failed to start MongoDB container")

	tdFunc := func(t *testing.T) {
		t.Helper()

		require.NoError(t, testcontainers.TerminateContainer(mongolocalContainer),
			"failed to terminate MongoDB container")
	}

	connString, err := mongolocalContainer.ConnectionString(ctx)
	if err != nil {
		tdFunc(t)
		t.Fatalf("failed to get connection string: %s", err)
	}

	mopts := opts.mongoClientOpts
	if mopts == nil {
		mopts = mongooptions.Client()
	}

	// Users can't override the connection string.
	mopts = mopts.ApplyURI(connString)

	mongoClient, err := mongo.Connect(mopts)
	if err != nil {
		tdFunc(t)
		t.Fatalf("failed to connect to mongo: %s", err)
	}

	return mongoClient, func(t *testing.T) {
		t.Helper()

		require.NoError(t, mongoClient.Disconnect(ctx), "failed to disconnect mongo client")
		tdFunc(t)
	}
}

// needsCustomWaitStrategy checks if the MongoDB image version requires a custom
// wait strategy. MongoDB 4.x and earlier use lowercase "waiting for
// connections".
func needsCustomWaitStrategy(image string) bool {
	// Extract version from image string.
	re := regexp.MustCompile(`mongo:?(\d+)\.`)
	matches := re.FindStringSubmatch(image)
	if len(matches) < 2 {
		// Can't determine version, use default wait strategy
		return false
	}

	majorVersion, err := strconv.Atoi(matches[1])
	if err != nil {
		return false
	}

	// MongoDB 4.x and earlier need the custom strategy
	return majorVersion <= 4
}
