// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/handshake"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func TestAtlas(t *testing.T) {
	uris := flag.Args()
	ctx := context.Background()

	t.Logf("Running atlas tests for %d uris\n", len(uris))

	for idx, uri := range uris {
		t.Logf("Running test %d\n", idx)

		// Set a low server selection timeout so we fail fast if there are errors.
		clientOpts := options.Client().
			ApplyURI(uri).
			SetServerSelectionTimeout(1 * time.Second)

		// Run basic connectivity test.
		if err := runTest(ctx, clientOpts); err != nil {
			t.Fatalf("error running test with TLS at index %d: %v", idx, err)
		}

		tlsConfigSkipVerify := clientOpts.TLSConfig
		tlsConfigSkipVerify.InsecureSkipVerify = true

		// Run the connectivity test with InsecureSkipVerify to ensure SNI is done correctly even if verification is
		// disabled.
		clientOpts.SetTLSConfig(tlsConfigSkipVerify)

		if err := runTest(ctx, clientOpts); err != nil {
			t.Fatalf("error running test with tlsInsecure at index %d: %v", idx, err)
		}
	}

	t.Logf("Finished!")
}

func runTest(ctx context.Context, clientOpts *options.ClientOptions) error {
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return fmt.Errorf("Connect error: %w", err)
	}

	defer func() {
		_ = client.Disconnect(ctx)
	}()

	db := client.Database("test")
	cmd := bson.D{{handshake.LegacyHello, 1}}
	err = db.RunCommand(ctx, cmd).Err()
	if err != nil {
		return fmt.Errorf("legacy hello error: %w", err)
	}

	coll := db.Collection("test")
	if err = coll.FindOne(ctx, bson.D{{"x", 1}}).Err(); err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("FindOne error: %w", err)
	}
	return nil
}
