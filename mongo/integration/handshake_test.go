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
	"go.mongodb.org/mongo-driver/internal/driverutil"
	"go.mongodb.org/mongo-driver/internal/handshake"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/version"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func TestHandshakeProse(t *testing.T) {
	mt := mtest.New(t)

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
	t.Setenv(driverutil.EnvVarAWSExecutionEnv, "")
	t.Setenv(driverutil.EnvVarFunctionsWorkerRuntime, "")
	t.Setenv(driverutil.EnvVarKService, "")
	t.Setenv(driverutil.EnvVarVercel, "")
	t.Setenv(driverutil.EnvVarAWSRegion, "")
	t.Setenv(driverutil.EnvVarAWSLambdaFunctionMemorySize, "")
	t.Setenv(driverutil.EnvVarFunctionMemoryMB, "")
	t.Setenv(driverutil.EnvVarFunctionTimeoutSec, "")
	t.Setenv(driverutil.EnvVarFunctionRegion, "")
	t.Setenv(driverutil.EnvVarVercelRegion, "")

	for _, test := range []struct {
		name string
		env  map[string]string
		want bson.D
	}{
		{
			name: "1. valid AWS",
			env: map[string]string{
				driverutil.EnvVarAWSExecutionEnv:             "AWS_Lambda_java8",
				driverutil.EnvVarAWSRegion:                   "us-east-2",
				driverutil.EnvVarAWSLambdaFunctionMemorySize: "1024",
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
				driverutil.EnvVarFunctionsWorkerRuntime: "node",
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "azure.func"},
			}),
		},
		{
			name: "3. valid GCP",
			env: map[string]string{
				driverutil.EnvVarKService:           "servicename",
				driverutil.EnvVarFunctionMemoryMB:   "1024",
				driverutil.EnvVarFunctionTimeoutSec: "60",
				driverutil.EnvVarFunctionRegion:     "us-central1",
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
				driverutil.EnvVarVercel:       "1",
				driverutil.EnvVarVercelRegion: "cdg1",
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "vercel"},
				{Key: "region", Value: "cdg1"},
			}),
		},
		{
			name: "5. invalid multiple providers",
			env: map[string]string{
				driverutil.EnvVarAWSExecutionEnv:        "AWS_Lambda_java8",
				driverutil.EnvVarFunctionsWorkerRuntime: "node",
			},
			want: clientMetadata(nil),
		},
		{
			name: "6. invalid long string",
			env: map[string]string{
				driverutil.EnvVarAWSExecutionEnv: "AWS_Lambda_java8",
				driverutil.EnvVarAWSRegion: func() string {
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
				driverutil.EnvVarAWSExecutionEnv:             "AWS_Lambda_java8",
				driverutil.EnvVarAWSLambdaFunctionMemorySize: "big",
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "aws.lambda"},
			}),
		},
		{
			name: "8. Invalid - AWS_EXECUTION_ENV does not start with \"AWS_Lambda_\"",
			env: map[string]string{
				driverutil.EnvVarAWSExecutionEnv: "EC2",
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
