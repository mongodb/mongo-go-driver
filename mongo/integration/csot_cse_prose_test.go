// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build cse
// +build cse

package integration

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CSOT prose tests that require the 'cse' Go build tag.
func TestCSOTClientSideEncryptionProse(t *testing.T) {
	verifyClientSideEncryptionVarsSet(t)
	mt := mtest.New(t, mtest.NewOptions().MinServerVersion("4.2").CreateClient(false))
	defer mt.Close()

	mt.RunOpts("2. maxTimeMS is not set for commands sent to mongocryptd",
		noClientOpts, func(mt *mtest.T) {
			kmsProviders := map[string]map[string]interface{}{
				"local": {
					"key": localMasterKey,
				},
			}
			mongocryptdSpawnArgs := map[string]interface{}{
				// Pass a custom pidfilepath to ensure a new mongocryptd process is spawned.
				"mongocryptdSpawnArgs": []string{"--port=23000", "--pidfilepath=TestCSOTClientSideEncryptionProse_1.pid"},
				"mongocryptdURI":       "mongodb://localhost:23000",
				// Do not use the shared library to ensure mongocryptd is spawned.
				"__cryptSharedLibDisabledForTestOnly": true,
			}

			// Setup encrypted client to cause spawning of mongocryptd on port 23000.
			aeo := options.AutoEncryption().SetKmsProviders(kmsProviders).
				SetExtraOptions(mongocryptdSpawnArgs)
			cliOpts := options.Client().ApplyURI(mtest.ClusterURI()).SetAutoEncryptionOptions(aeo)
			testutil.AddTestServerAPIVersion(cliOpts)
			encClient, err := mongo.Connect(context.Background(), cliOpts)
			assert.Nil(mt, err, "Connect error: %v", err)
			defer func() {
				err = encClient.Disconnect(context.Background())
				assert.Nil(mt, err, "encrypted client Disconnect error: %v", err)
			}()

			// Run a Find through the encrypted client to make sure mongocryptd is started ('find' uses the
			// mongocryptd and will wait for it to be active).
			_, err = encClient.Database("test").Collection("test").Find(context.Background(), bson.D{})
			assert.Nil(mt, err, "Find error: %v", err)

			// Use a new Client to connect to 23000 where mongocryptd should be running. Use a custom
			// command monitor to examine the eventual 'ping'.
			var started *event.CommandStartedEvent
			mcryptMonitor := &event.CommandMonitor{
				Started: func(_ context.Context, evt *event.CommandStartedEvent) {
					started = evt
				},
			}
			mcryptOpts := options.Client().SetMonitor(mcryptMonitor).
				ApplyURI("mongodb://localhost:23000/?timeoutMS=1000")
			testutil.AddTestServerAPIVersion(mcryptOpts)
			mcryptClient, err := mongo.Connect(context.Background(), mcryptOpts)
			assert.Nil(mt, err, "mongocryptd Connect error: %v", err)
			defer func() {
				err = mcryptClient.Disconnect(context.Background())
				assert.Nil(mt, err, "mongocryptd Disconnect error: %v", err)
			}()

			// Run Ping and assert that sent command does not contain 'maxTimeMS'. The 'ping' command
			// does not exist on mongocryptd, so ignore the CommandNotFound error.
			_ = mcryptClient.Ping(context.Background(), nil)
			assert.NotNil(mt, started, "expected a CommandStartedEvent, got nil")
			assert.Equal(mt, started.CommandName, "ping", "expected 'ping', got %q", started.CommandName)
			commandElems, err := started.Command.Elements()
			assert.Nil(mt, err, "Elements error: %v", err)
			for _, elem := range commandElems {
				assert.NotEqual(mt, elem.Key(), "maxTimeMS",
					"expected no 'maxTimeMS' field in ping to mongocryptd")
			}
		})
}
