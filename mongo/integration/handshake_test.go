// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"os"
	"reflect"
	"runtime"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/handshake"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/version"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

func TestHandshakeProse(t *testing.T) {
	mt := mtest.New(t)

	if len(os.Getenv("DOCKER_RUNNING")) > 0 {
		t.Skip("These tests gives different results when run in Docker due to extra environment data.")
	}

	opts := mtest.NewOptions().
		CreateCollection(false).
		ClientType(mtest.Proxy)

	clientMetadata := func(env bson.D) bson.D {
		elems := bson.D{
			{Key: "driver", Value: bson.D{
				{Key: "name", Value: "mongo-go-driver"},
				{Key: "version", Value: version.Driver},
			}},
			{Key: "os", Value: bson.D{
				{Key: "type", Value: runtime.GOOS},
				{Key: "architecture", Value: runtime.GOARCH},
			}},
		}

		elems = append(elems, bson.E{Key: "platform", Value: runtime.Version()})

		// If env is empty, don't include it in the metadata.
		if env != nil && !reflect.DeepEqual(env, bson.D{}) {
			elems = append(elems, bson.E{Key: "env", Value: env})
		}

		return elems
	}

	// Reset the environment variables to avoid environment namespace
	// collision.
	t.Setenv("AWS_EXECUTION_ENV", "")
	t.Setenv("FUNCTIONS_WORKER_RUNTIME", "")
	t.Setenv("K_SERVICE", "")
	t.Setenv("VERCEL", "")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_LAMBDA_FUNCTION_MEMORY_SIZE", "")
	t.Setenv("FUNCTION_MEMORY_MB", "")
	t.Setenv("FUNCTION_TIMEOUT_SEC", "")
	t.Setenv("FUNCTION_REGION", "")
	t.Setenv("VERCEL_REGION", "")

	for _, test := range []struct {
		name string
		env  map[string]string
		want bson.D
	}{
		{
			name: "1. valid AWS",
			env: map[string]string{
				"AWS_EXECUTION_ENV":               "AWS_Lambda_java8",
				"AWS_REGION":                      "us-east-2",
				"AWS_LAMBDA_FUNCTION_MEMORY_SIZE": "1024",
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "aws.lambda"},
				{Key: "memory_mb", Value: 1024},
				{Key: "region", Value: "us-east-2"},
			}),
		},
		{
			name: "2. valid Azure",
			env: map[string]string{
				"FUNCTIONS_WORKER_RUNTIME": "node",
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "azure.func"},
			}),
		},
		{
			name: "3. valid GCP",
			env: map[string]string{
				"K_SERVICE":            "servicename",
				"FUNCTION_MEMORY_MB":   "1024",
				"FUNCTION_TIMEOUT_SEC": "60",
				"FUNCTION_REGION":      "us-central1",
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "gcp.func"},
				{Key: "memory_mb", Value: 1024},
				{Key: "region", Value: "us-central1"},
				{Key: "timeout_sec", Value: 60},
			}),
		},
		{
			name: "4. valid Vercel",
			env: map[string]string{
				"VERCEL":        "1",
				"VERCEL_REGION": "cdg1",
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "vercel"},
				{Key: "region", Value: "cdg1"},
			}),
		},
		{
			name: "5. invalid multiple providers",
			env: map[string]string{
				"AWS_EXECUTION_ENV":        "AWS_Lambda_java8",
				"FUNCTIONS_WORKER_RUNTIME": "node",
			},
			want: clientMetadata(nil),
		},
		{
			name: "6. invalid long string",
			env: map[string]string{
				"AWS_EXECUTION_ENV": "AWS_Lambda_java8",
				"AWS_REGION": func() string {
					var s string
					for i := 0; i < 512; i++ {
						s += "a"
					}
					return s
				}(),
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "aws.lambda"},
			}),
		},
		{
			name: "7. invalid wrong types",
			env: map[string]string{
				"AWS_EXECUTION_ENV":               "AWS_Lambda_java8",
				"AWS_LAMBDA_FUNCTION_MEMORY_SIZE": "big",
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "aws.lambda"},
			}),
		},
		{
			name: "8. Invalid - AWS_EXECUTION_ENV does not start with \"AWS_Lambda_\"",
			env: map[string]string{
				"AWS_EXECUTION_ENV": "EC2",
			},
			want: clientMetadata(nil),
		},
	} {
		test := test

		mt.RunOpts(test.name, opts, func(mt *mtest.T) {
			for k, v := range test.env {
				mt.Setenv(k, v)
			}

			// Ping the server to ensure the handshake has completed.
			err := mt.Client.Ping(context.Background(), nil)
			require.NoError(mt, err, "Ping error: %v", err)

			messages := mt.GetProxiedMessages()
			handshakeMessage := messages[:1][0]

			hello := handshake.LegacyHello
			if os.Getenv("REQUIRE_API_VERSION") == "true" {
				hello = "hello"
			}

			assert.Equal(mt, hello, handshakeMessage.CommandName)

			// Lookup the "client" field in the command document.
			clientVal, err := handshakeMessage.Sent.Command.LookupErr("client")
			require.NoError(mt, err, "expected command %s to contain client field", handshakeMessage.Sent.Command)

			got, ok := clientVal.DocumentOK()
			require.True(mt, ok, "expected client field to be a document, got %s", clientVal.Type)

			wantBytes, err := bson.Marshal(test.want)
			require.NoError(mt, err, "error marshaling want document: %v", err)

			want := bsoncore.Document(wantBytes)
			assert.Equal(mt, want, got, "want: %v, got: %v", want, got)
		})
	}
}

func TestLoadBalancedConnectionHandshake(t *testing.T) {
	mt := mtest.New(t)

	lbopts := mtest.NewOptions().ClientType(mtest.Proxy).Topologies(
		mtest.LoadBalanced)

	mt.RunOpts("LB connection handshake uses OP_MSG", lbopts, func(mt *mtest.T) {
		// Ping the server to ensure the handshake has completed.
		err := mt.Client.Ping(context.Background(), nil)
		require.NoError(mt, err, "Ping error: %v", err)

		messages := mt.GetProxiedMessages()
		handshakeMessage := messages[:1][0]

		// Per the specifications, if loadBalanced=true, drivers MUST use the hello
		// command for the initial handshake and use the OP_MSG protocol.
		assert.Equal(mt, "hello", handshakeMessage.CommandName)
		assert.Equal(mt, wiremessage.OpMsg, handshakeMessage.Sent.OpCode)
	})

	opts := mtest.NewOptions().ClientType(mtest.Proxy).Topologies(
		mtest.ReplicaSet,
		mtest.Sharded,
		mtest.Single,
		mtest.ShardedReplicaSet)

	mt.RunOpts("non-LB connection handshake uses OP_QUERY", opts, func(mt *mtest.T) {
		// Ping the server to ensure the handshake has completed.
		err := mt.Client.Ping(context.Background(), nil)
		require.NoError(mt, err, "Ping error: %v", err)

		messages := mt.GetProxiedMessages()
		handshakeMessage := messages[:1][0]

		want := wiremessage.OpQuery

		hello := handshake.LegacyHello
		if os.Getenv("REQUIRE_API_VERSION") == "true" {
			hello = "hello"

			// If the server API version is requested, then we should use OP_MSG
			// regardless of the topology
			want = wiremessage.OpMsg
		}

		assert.Equal(mt, hello, handshakeMessage.CommandName)
		assert.Equal(mt, want, handshakeMessage.Sent.OpCode)
	})
}
