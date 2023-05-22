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
	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/version"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func TestHandshakeProse(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()

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

	const (
		envVarAWSExecutionEnv             = "AWS_EXECUTION_ENV"
		envVarAWSRegion                   = "AWS_REGION"
		envVarAWSLambdaFunctionMemorySize = "AWS_LAMBDA_FUNCTION_MEMORY_SIZE"
		envVarFunctionsWorkerRuntime      = "FUNCTIONS_WORKER_RUNTIME"
		envVarKService                    = "K_SERVICE"
		envVarFunctionMemoryMB            = "FUNCTION_MEMORY_MB"
		envVarFunctionTimeoutSec          = "FUNCTION_TIMEOUT_SEC"
		envVarFunctionRegion              = "FUNCTION_REGION"
		envVarVercel                      = "VERCEL"
		envVarVercelRegion                = "VERCEL_REGION"
	)

	// Reset the environment variables to avoid environment namespace
	// collision.
	t.Setenv(envVarAWSExecutionEnv, "")
	t.Setenv(envVarFunctionsWorkerRuntime, "")
	t.Setenv(envVarKService, "")
	t.Setenv(envVarVercel, "")
	t.Setenv(envVarAWSRegion, "")
	t.Setenv(envVarAWSLambdaFunctionMemorySize, "")
	t.Setenv(envVarFunctionMemoryMB, "")
	t.Setenv(envVarFunctionTimeoutSec, "")
	t.Setenv(envVarFunctionRegion, "")
	t.Setenv(envVarVercelRegion, "")

	for _, test := range []struct {
		name string
		env  map[string]string
		want bson.D
	}{
		{
			name: "1. valid AWS",
			env: map[string]string{
				envVarAWSExecutionEnv:             "AWS_Lambda_java8",
				envVarAWSRegion:                   "us-east-2",
				envVarAWSLambdaFunctionMemorySize: "1024",
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
				envVarFunctionsWorkerRuntime: "node",
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "azure.func"},
			}),
		},
		{
			name: "3. valid GCP",
			env: map[string]string{
				envVarKService:           "servicename",
				envVarFunctionMemoryMB:   "1024",
				envVarFunctionTimeoutSec: "60",
				envVarFunctionRegion:     "us-central1",
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
				envVarVercel:       "1",
				envVarVercelRegion: "cdg1",
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "vercel"},
				{Key: "region", Value: "cdg1"},
			}),
		},
		{
			name: "5. invalid multiple providers",
			env: map[string]string{
				envVarAWSExecutionEnv:        "AWS_Lambda_java8",
				envVarFunctionsWorkerRuntime: "node",
			},
			want: clientMetadata(nil),
		},
		{
			name: "6. invalid long string",
			env: map[string]string{
				envVarAWSExecutionEnv: "AWS_Lambda_java8",
				envVarAWSRegion: func() string {
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
				envVarAWSExecutionEnv:             "AWS_Lambda_java8",
				envVarAWSLambdaFunctionMemorySize: "big",
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "aws.lambda"},
			}),
		},
		{
			name: "8. Invalid - AWS_EXECUTION_ENV does not start with \"AWS_Lambda_\"",
			env: map[string]string{
				envVarAWSExecutionEnv: "EC2",
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

			// First two messages are handshake messages
			for idx, pair := range messages[:2] {
				hello := internal.LegacyHello
				//  Expect "hello" command name with API version.
				if os.Getenv("REQUIRE_API_VERSION") == "true" {
					hello = "hello"
				}

				assert.Equal(mt, pair.CommandName, hello, "expected and actual command name at index %d are different", idx)

				sent := pair.Sent

				// Lookup the "client" field in the command document.
				clientVal, err := sent.Command.LookupErr("client")
				require.NoError(mt, err, "expected command %s at index %d to contain client field", sent.Command, idx)

				got, ok := clientVal.DocumentOK()
				require.True(mt, ok, "expected client field to be a document, got %s", clientVal.Type)

				wantBytes, err := bson.Marshal(test.want)
				require.NoError(mt, err, "error marshaling want document: %v", err)

				want := bsoncore.Document(wantBytes)
				assert.Equal(mt, want, got, "want: %v, got: %v", want, got)
			}
		})
	}
}
