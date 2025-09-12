// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/assert/assertbsoncore"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/ptrutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/internal/test"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/wiremessage"
)

func TestHandshakeProse(t *testing.T) {
	mt := mtest.New(t)

	if len(os.Getenv("DOCKER_RUNNING")) > 0 {
		t.Skip("These tests gives different results when run in Docker due to extra environment data.")
	}

	opts := mtest.NewOptions().
		CreateCollection(false).
		ClientType(mtest.Proxy)

	driverInfo := &options.DriverInfo{
		Name:     "outer-library-name",
		Version:  "outer-library-version",
		Platform: "outer-library-platform",
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

	testCases := []struct {
		name string
		env  map[string]string
		opts *options.ClientOptions
		want []byte
	}{
		{
			name: "1. valid AWS",
			env: map[string]string{
				"AWS_EXECUTION_ENV":               "AWS_Lambda_java8",
				"AWS_REGION":                      "us-east-2",
				"AWS_LAMBDA_FUNCTION_MEMORY_SIZE": "1024",
			},
			opts: nil,
			want: test.EncodeClientMetadata(mt,
				test.WithClientMetadataEnvName("aws.lambda"),
				test.WithClientMetadataEnvMemoryMB(ptrutil.Ptr(1024)),
				test.WithClientMetadataEnvRegion("us-east-2"),
			),
		},
		{
			name: "2. valid Azure",
			env: map[string]string{
				"FUNCTIONS_WORKER_RUNTIME": "node",
			},
			opts: nil,
			want: test.EncodeClientMetadata(mt,
				test.WithClientMetadataEnvName("azure.func"),
			),
		},
		{
			name: "3. valid GCP",
			env: map[string]string{
				"K_SERVICE":            "servicename",
				"FUNCTION_MEMORY_MB":   "1024",
				"FUNCTION_TIMEOUT_SEC": "60",
				"FUNCTION_REGION":      "us-central1",
			},
			opts: nil,
			want: test.EncodeClientMetadata(mt,
				test.WithClientMetadataEnvName("gcp.func"),
				test.WithClientMetadataEnvMemoryMB(ptrutil.Ptr(1024)),
				test.WithClientMetadataEnvRegion("us-central1"),
				test.WithClientMetadataEnvTimeoutSec(ptrutil.Ptr(60)),
			),
		},
		{
			name: "4. valid Vercel",
			env: map[string]string{
				"VERCEL":        "1",
				"VERCEL_REGION": "cdg1",
			},
			opts: nil,
			want: test.EncodeClientMetadata(mt,
				test.WithClientMetadataEnvName("vercel"),
				test.WithClientMetadataEnvRegion("cdg1"),
			),
		},
		{
			name: "5. invalid multiple providers",
			env: map[string]string{
				"AWS_EXECUTION_ENV":        "AWS_Lambda_java8",
				"FUNCTIONS_WORKER_RUNTIME": "node",
			},
			opts: nil,
			want: test.EncodeClientMetadata(mt),
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
			opts: nil,
			want: test.EncodeClientMetadata(mt,
				test.WithClientMetadataEnvName("aws.lambda"),
			),
		},
		{
			name: "7. invalid wrong types",
			env: map[string]string{
				"AWS_EXECUTION_ENV":               "AWS_Lambda_java8",
				"AWS_LAMBDA_FUNCTION_MEMORY_SIZE": "big",
			},
			opts: nil,
			want: test.EncodeClientMetadata(mt,
				test.WithClientMetadataEnvName("aws.lambda"),
			),
		},
		{
			name: "8. Invalid - AWS_EXECUTION_ENV does not start with \"AWS_Lambda_\"",
			env: map[string]string{
				"AWS_EXECUTION_ENV": "EC2",
			},
			opts: nil,
			want: test.EncodeClientMetadata(mt),
		},
		{
			name: "driver info included",
			opts: options.Client().SetDriverInfo(driverInfo),
			want: test.EncodeClientMetadata(mt,
				test.WithClientMetadataDriverName("outer-library-name"),
				test.WithClientMetadataDriverVersion("outer-library-version"),
				test.WithClientMetadataDriverPlatform("outer-library-platform"),
			),
		},
	}

	for _, tc := range testCases {
		mt.RunOpts(tc.name, opts, func(mt *mtest.T) {
			for k, v := range tc.env {
				mt.Setenv(k, v)
			}

			if tc.opts != nil {
				mt.ResetClient(tc.opts)
			}

			// Ping the server to ensure the handshake has completed.
			err := mt.Client.Ping(context.Background(), nil)
			require.NoError(mt, err, "Ping error: %v", err)

			firstMessage := mt.GetProxyCapture().TryNext()
			require.NotNil(mt, firstMessage, "expected to capture a proxied message")

			assert.True(mt, firstMessage.IsHandshake(), "expected first message to be a handshake")
			assertbsoncore.HandshakeClientMetadata(mt, tc.want, firstMessage.Sent.Command)
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

		firstMessage := mt.GetProxyCapture().TryNext()
		require.NotNil(mt, firstMessage, "expected to capture a proxied message")

		// Per the specifications, if loadBalanced=true, drivers MUST use the hello
		// command for the initial handshake and use the OP_MSG protocol.
		assert.True(mt, firstMessage.IsHandshake(), "expected first message to be a handshake")
		assert.Equal(mt, wiremessage.OpMsg, firstMessage.Sent.OpCode)
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

		firstMessage := mt.GetProxyCapture().TryNext()
		require.NotNil(mt, firstMessage, "expected to capture a proxied message")

		want := wiremessage.OpQuery
		if os.Getenv("REQUIRE_API_VERSION") == "true" {
			// If the server API version is requested, then we should use OP_MSG
			// regardless of the topology
			want = wiremessage.OpMsg
		}

		assert.True(mt, firstMessage.IsHandshake(), "expected first message to be a handshake")
		assert.Equal(mt, want, firstMessage.Sent.OpCode)
	})
}

// Test 1: Test that the driver updates metadata
// Test 2: Multiple Successive Metadata Updates
// Test 3: Multiple Successive Metadata Updates with Duplicate Data
func TestHandshakeProse_AppendMetadata_Test1_Test2_Test3(t *testing.T) {
	mt := mtest.New(t)

	initialDriverInfo := options.DriverInfo{
		Name:     "library",
		Version:  "1.2",
		Platform: "Library Platform",
	}

	testCases := []struct {
		name       string
		driverInfo options.DriverInfo
		want       options.DriverInfo

		// append initialDriverInfo using client.AppendDriverInfo instead of as a
		// client-level constructor.
		append bool
	}{
		{
			name: "test1.1: append new driver info",
			driverInfo: options.DriverInfo{
				Name:     "framework",
				Version:  "2.0",
				Platform: "Framework Platform",
			},
			want: options.DriverInfo{
				Name:     "library|framework",
				Version:  "1.2|2.0",
				Platform: "Library Platform|Framework Platform",
			},
			append: false,
		},
		{
			name: "test1.2: append with no platform",
			driverInfo: options.DriverInfo{
				Name:     "framework",
				Version:  "2.0",
				Platform: "",
			},
			want: options.DriverInfo{
				Name:     "library|framework",
				Version:  "1.2|2.0",
				Platform: "Library Platform",
			},
			append: false,
		},
		{
			name: "test1.3: append with no version",
			driverInfo: options.DriverInfo{
				Name:     "framework",
				Version:  "",
				Platform: "Framework Platform",
			},
			want: options.DriverInfo{
				Name:     "library|framework",
				Version:  "1.2",
				Platform: "Library Platform|Framework Platform",
			},
			append: false,
		},
		{
			name: "test1.4: append with name only",
			driverInfo: options.DriverInfo{
				Name:     "framework",
				Version:  "",
				Platform: "",
			},
			want: options.DriverInfo{
				Name:     "library|framework",
				Version:  "1.2",
				Platform: "Library Platform",
			},
			append: false,
		},
		{
			name: "test2.1: append new driver info after appending",
			driverInfo: options.DriverInfo{
				Name:     "framework",
				Version:  "2.0",
				Platform: "Framework Platform",
			},
			want: options.DriverInfo{
				Name:     "library|framework",
				Version:  "1.2|2.0",
				Platform: "Library Platform|Framework Platform",
			},
			append: true,
		},
		{
			name: "test2.2: append with no platform after appending",
			driverInfo: options.DriverInfo{
				Name:     "framework",
				Version:  "2.0",
				Platform: "",
			},
			want: options.DriverInfo{
				Name:     "library|framework",
				Version:  "1.2|2.0",
				Platform: "Library Platform",
			},
			append: true,
		},
		{
			name: "test2.3: append with no version after appending",
			driverInfo: options.DriverInfo{
				Name:     "framework",
				Version:  "",
				Platform: "Framework Platform",
			},
			want: options.DriverInfo{
				Name:     "library|framework",
				Version:  "1.2",
				Platform: "Library Platform|Framework Platform",
			},
			append: true,
		},
		{
			name: "test2.4: append with name only after appending",
			driverInfo: options.DriverInfo{
				Name:     "framework",
				Version:  "",
				Platform: "",
			},
			want: options.DriverInfo{
				Name:     "library|framework",
				Version:  "1.2",
				Platform: "Library Platform",
			},
			append: true,
		},
		{
			name: "test3.1: same driver info after appending",
			driverInfo: options.DriverInfo{
				Name:     "library",
				Version:  "1.2",
				Platform: "Library Platform",
			},
			want: options.DriverInfo{
				Name:     "library",
				Version:  "1.2",
				Platform: "Library Platform",
			},
			append: true,
		},
		{
			name: "test3.2: same version and platform after appending",
			driverInfo: options.DriverInfo{
				Name:     "framework",
				Version:  "1.2",
				Platform: "Library Platform",
			},
			want: options.DriverInfo{
				Name:     "library|framework",
				Version:  "1.2",
				Platform: "Library Platform",
			},
			append: true,
		},
		{
			name: "test3.3: same name and platform after appending",
			driverInfo: options.DriverInfo{
				Name:     "library",
				Version:  "2.0",
				Platform: "Library Platform",
			},
			want: options.DriverInfo{
				Name:     "library",
				Version:  "1.2|2.0",
				Platform: "Library Platform",
			},
			append: true,
		},
		{
			name: "test3.4: same name and version after appending",
			driverInfo: options.DriverInfo{
				Name:     "library",
				Version:  "1.2",
				Platform: "Framework Platform",
			},
			want: options.DriverInfo{
				Name:     "library",
				Version:  "1.2",
				Platform: "Library Platform|Framework Platform",
			},
			append: true,
		},
		{
			name: "test3.5: same platform after appending",
			driverInfo: options.DriverInfo{
				Name:     "framework",
				Version:  "2.0",
				Platform: "Library Platform",
			},
			want: options.DriverInfo{
				Name:     "library|framework",
				Version:  "1.2|2.0",
				Platform: "Library Platform",
			},
			append: true,
		},
		{
			name: "test3.6: same version after appending",
			driverInfo: options.DriverInfo{
				Name:     "framework",
				Version:  "1.2",
				Platform: "Framework Platform",
			},
			want: options.DriverInfo{
				Name:     "library|framework",
				Version:  "1.2",
				Platform: "Library Platform|Framework Platform",
			},
			append: true,
		},
		{
			name: "test3.7: same name after appending",
			driverInfo: options.DriverInfo{
				Name:     "library",
				Version:  "2.0",
				Platform: "Framework Platform",
			},
			want: options.DriverInfo{
				Name:     "library",
				Version:  "1.2|2.0",
				Platform: "Library Platform|Framework Platform",
			},
			append: true,
		},
	}

	for _, tc := range testCases {
		// Create a top-level client that can be shared among sub-tests. This is
		// necessary to test appending driver info to an existing client.
		opts := mtest.NewOptions().CreateClient(false).ClientType(mtest.Proxy)

		mt.RunOpts(tc.name, opts, func(mt *mtest.T) {
			clientOpts := options.Client().
				// Set idle timeout to 1ms to force new connections to be created
				// throughout the lifetime of the test.
				SetMaxConnIdleTime(1 * time.Millisecond)

			if !tc.append {
				clientOpts = clientOpts.SetDriverInfo(&initialDriverInfo)
			}

			mt.ResetClient(clientOpts)

			if tc.append {
				mt.Client.AppendDriverInfo(initialDriverInfo)
			}

			// Send a ping command to the server and verify that the command succeeded.
			err := mt.Client.Ping(context.Background(), nil)
			require.NoError(mt, err, "Ping error: %v", err)

			// Save intercepted `client` document as `initialClientMetadata`.
			initialClientMetadata := mt.GetProxyCapture().TryNext()

			require.NotNil(mt, initialClientMetadata, "expected to capture a proxied message")
			assert.True(mt, initialClientMetadata.IsHandshake(), "expected first message to be a handshake")

			// Wait 5ms for the connection to become idle.
			time.Sleep(20 * time.Millisecond)

			mt.Client.AppendDriverInfo(tc.driverInfo)

			// Drain the proxy
			mt.GetProxyCapture().Drain()

			// Send a ping command to the server and verify that the command succeeded.
			err = mt.Client.Ping(context.Background(), nil)
			require.NoError(mt, err, "Ping error: %v", err)

			// Capture the first message sent after appending driver info.
			gotMessage := mt.GetProxyCapture().TryNext()
			require.NotNil(mt, gotMessage, "expected to capture a proxied message")
			assert.True(mt, gotMessage.IsHandshake(), "expected first message to be a handshake")

			want := test.EncodeClientMetadata(mt,
				test.WithClientMetadataDriverName(tc.want.Name),
				test.WithClientMetadataDriverVersion(tc.want.Version),
				test.WithClientMetadataDriverPlatform(tc.want.Platform),
			)

			assertbsoncore.HandshakeClientMetadata(mt, want, gotMessage.Sent.Command)
		})
	}
}

// Test 4: Multiple Metadata Updates with Duplicate Data.
func TestHandshakeProse_AppendMetadata_MultipleUpdatesWithDuplicateFields(t *testing.T) {
	opts := mtest.NewOptions().ClientType(mtest.Proxy)
	mt := mtest.New(t, opts)

	clientOpts := options.Client().
		// Set idle timeout to 1ms to force new connections to be created
		// throughout the lifetime of the test.
		SetMaxConnIdleTime(1 * time.Millisecond)

	// 1. Create a top-level client that can be shared among sub-tests. This is
	// necessary to test appending driver info to an existing client.
	mt.ResetClient(clientOpts)

	originalDriverInfo := options.DriverInfo{
		Name:     "library",
		Version:  "1.2",
		Platform: "Library Platform",
	}

	// 2. Append initial driver info using client.AppendDriverInfo.
	mt.Client.AppendDriverInfo(originalDriverInfo)

	// 3. Send a ping command to the server and verify that the command succeeded.
	err := mt.Client.Ping(context.Background(), nil)
	require.NoError(mt, err, "Ping error: %v", err)

	// 4. Wait 5ms for the connection to become idle.
	time.Sleep(5 * time.Millisecond)

	// 5. Append new driver info.
	mt.Client.AppendDriverInfo(options.DriverInfo{
		Name:     "framework",
		Version:  "2.0",
		Platform: "Framework Platform",
	})

	// Drain the proxy to ensure we only capture messages after appending.
	mt.GetProxyCapture().Drain()

	// 6. Send a ping command to the server and verify that the command succeeded.
	err = mt.Client.Ping(context.Background(), nil)
	require.NoError(mt, err, "Ping error: %v", err)

	// 7. Save intercepted `client` document as `clientMetadata`.
	clientMetadata := mt.GetProxyCapture().TryNext()

	require.NotNil(mt, clientMetadata, "expected to capture a proxied message")
	assert.True(mt, clientMetadata.IsHandshake(), "expected first message to be a handshake")

	// 8. Wait 5ms for the connection to become idle.
	time.Sleep(5 * time.Millisecond)

	// Drain the proxy to ensure we only capture messages after appending.
	mt.GetProxyCapture().Drain()

	// 9. Append the original driver info again.
	mt.Client.AppendDriverInfo(originalDriverInfo)

	// 10. Send a ping command to the server and verify that the command succeeded.
	err = mt.Client.Ping(context.Background(), nil)
	require.NoError(mt, err, "Ping error: %v", err)

	// 11. Save intercepted `client` document as `clientMetadata`.
	updatedClientMetadata := mt.GetProxyCapture().TryNext()

	require.NotNil(mt, updatedClientMetadata, "expected to capture a proxied message")
	assert.True(mt, updatedClientMetadata.IsHandshake(), "expected first message to be a handshake")

	assertbsoncore.HandshakeClientMetadata(mt, clientMetadata.Sent.Command, updatedClientMetadata.Sent.Command)
}

// Test 5: Metadata is not appended if identical to initial metadata
func TestHandshakeProse_AppendMetadata_NotAppendedIfIdentical(t *testing.T) {
	opts := mtest.NewOptions().ClientType(mtest.Proxy)
	mt := mtest.New(t, opts)

	originalDriverInfo := options.DriverInfo{
		Name:     "library",
		Version:  "1.2",
		Platform: "Library Platform",
	}

	clientOpts := options.Client().
		// Set idle timeout to 1ms to force new connections to be created
		// throughout the lifetime of the test.
		SetMaxConnIdleTime(1 * time.Millisecond).
		SetDriverInfo(&originalDriverInfo)

	// 1. Create a top-level client that can be shared among sub-tests. This is
	// necessary to test appending driver info to an existing client.
	mt.ResetClient(clientOpts)

	// Drain the proxy to ensure we only capture messages after appending.
	mt.GetProxyCapture().Drain()

	// 2. Send a ping command to the server and verify that the command succeeded.
	err := mt.Client.Ping(context.Background(), nil)
	require.NoError(mt, err, "Ping error: %v", err)

	clientMetadata := mt.GetProxyCapture().TryNext()
	require.NotNil(mt, clientMetadata, "expected to capture a proxied message")
	assert.True(mt, clientMetadata.IsHandshake(), "expected first message to be a handshake")

	// 3. Wait 5ms for the connection to become idle.
	time.Sleep(5 * time.Millisecond)

	// 5. Append new driver info.
	mt.Client.AppendDriverInfo(options.DriverInfo{
		Name:     "library",
		Version:  "1.2",
		Platform: "Library Platform",
	})

	// Drain the proxy to ensure we only capture messages after appending.
	mt.GetProxyCapture().Drain()

	// 6. Send a ping command to the server and verify that the command succeeded.
	err = mt.Client.Ping(context.Background(), nil)
	require.NoError(mt, err, "Ping error: %v", err)

	// 7.  Save intercepted `client` document as `updatedClientMetadata`.
	updatedClientMetadata := mt.GetProxyCapture().TryNext()
	require.NotNil(mt, updatedClientMetadata, "expected to capture a proxied message")
	assert.True(mt, updatedClientMetadata.IsHandshake(), "expected first message to be a handshake")

	assertbsoncore.HandshakeClientMetadata(mt, clientMetadata.Sent.Command, updatedClientMetadata.Sent.Command)
}

// Test 6: Metadata is not appended if identical to initial metadata (separated
// by non-identical metadata)
func TestHandshakeProse_AppendMetadata_NotAppendedIfIdentical_NonSequential(t *testing.T) {
	opts := mtest.NewOptions().ClientType(mtest.Proxy)
	mt := mtest.New(t, opts)

	originalDriverInfo := options.DriverInfo{
		Name:     "library",
		Version:  "1.2",
		Platform: "Library Platform",
	}

	clientOpts := options.Client().
		// Set idle timeout to 1ms to force new connections to be created
		// throughout the lifetime of the test.
		SetMaxConnIdleTime(1 * time.Millisecond).
		SetDriverInfo(&originalDriverInfo)

	// 1. Create a top-level client that can be shared among sub-tests. This is
	// necessary to test appending driver info to an existing client.
	mt.ResetClient(clientOpts)

	// Drain the proxy to ensure we only capture messages after appending.
	mt.GetProxyCapture().Drain()

	// 2. Send a ping command to the server and verify that the command succeeded.
	err := mt.Client.Ping(context.Background(), nil)
	require.NoError(mt, err, "Ping error: %v", err)

	// 3. Wait 5ms for the connection to become idle.
	time.Sleep(5 * time.Millisecond)

	// 4. Append new driver info.
	mt.Client.AppendDriverInfo(options.DriverInfo{
		Name:     "framework",
		Version:  "1.2",
		Platform: "Framework Platform",
	})

	// Drain the proxy to ensure we only capture messages after appending.
	mt.GetProxyCapture().Drain()

	// 5. Send a ping command to the server and verify that the command succeeded.
	err = mt.Client.Ping(context.Background(), nil)
	require.NoError(mt, err, "Ping error: %v", err)

	// 6. Save intercepted `client` document as `clientMetadata`.
	clientMetadata := mt.GetProxyCapture().TryNext()
	require.NotNil(mt, clientMetadata, "expected to capture a proxied message")
	assert.True(mt, clientMetadata.IsHandshake(), "expected first message to be a handshake")

	// 7. Wait 5ms for the connection to become idle.
	time.Sleep(5 * time.Millisecond)

	// 8. Append new driver info.
	mt.Client.AppendDriverInfo(options.DriverInfo{
		Name:     "library",
		Version:  "1.2",
		Platform: "Library Platform",
	})

	// Drain the proxy to ensure we only capture messages after appending.
	mt.GetProxyCapture().Drain()

	// 9. Send a `ping` command to the server and verify that the command
	// succeeds.
	err = mt.Client.Ping(context.Background(), nil)
	require.NoError(mt, err, "Ping error: %v", err)

	// 10. Save intercepted `client` document as `updatedClientMetadata`.
	updatedClientMetadata := mt.GetProxyCapture().TryNext()
	require.NotNil(mt, updatedClientMetadata, "expected to capture a proxied message")
	assert.True(mt, updatedClientMetadata.IsHandshake(), "expected first message to be a handshake")

	assertbsoncore.HandshakeClientMetadata(mt, clientMetadata.Sent.Command, updatedClientMetadata.Sent.Command)
}

// Test 7: Empty strings are considered unset when appending duplicate metadata.
func TestHandshakeProse_AppendMetadata_EmptyStrings(t *testing.T) {
	mt := mtest.New(t)

	testCases := []struct {
		name               string
		initialDriverInfo  options.DriverInfo
		toAppendDriverInfo options.DriverInfo
	}{
		{
			name: "name empty",
			initialDriverInfo: options.DriverInfo{
				Name:     "",
				Version:  "1.2",
				Platform: "Library Platform",
			},
			toAppendDriverInfo: options.DriverInfo{
				Name:     "",
				Version:  "1.2",
				Platform: "Library Platform",
			},
		},
		{
			name: "version empty",
			initialDriverInfo: options.DriverInfo{
				Name:     "library",
				Version:  "",
				Platform: "Library Platform",
			},
			toAppendDriverInfo: options.DriverInfo{
				Name:     "library",
				Version:  "",
				Platform: "Library Platform",
			},
		},
		{
			name: "platform empty",
			initialDriverInfo: options.DriverInfo{
				Name:     "library",
				Version:  "1.2",
				Platform: "",
			},
			toAppendDriverInfo: options.DriverInfo{
				Name:     "library",
				Version:  "1.2",
				Platform: "",
			},
		},
	}

	for _, tc := range testCases {
		// Create a top-level client that can be shared among sub-tests. This is
		// necessary to test appending driver info to an existing client.
		opts := mtest.NewOptions().CreateClient(false).ClientType(mtest.Proxy)
		mt.RunOpts(tc.name, opts, func(mt *mtest.T) {
			// 1. Create a `MongoClient` instance.
			clientOpts := options.Client().
				// Set idle timeout to 1ms to force new connections to be created
				// throughout the lifetime of the test.
				SetMaxConnIdleTime(1 * time.Millisecond)

			mt.ResetClient(clientOpts)

			// 2. Append the `DriverInfoOptions` from the selected test case from
			// the initial metadata section.
			mt.Client.AppendDriverInfo(tc.initialDriverInfo)

			mt.GetProxyCapture().Drain()

			// 3. Send a `ping` command to the server and verify that the command
			// succeeds.
			err := mt.Client.Ping(context.Background(), nil)
			require.NoError(mt, err, "Ping error: %v", err)

			// 4. Save intercepted `client` document as `initialClientMetadata`.
			initialClientMetadata := mt.GetProxyCapture().TryNext()

			require.NotNil(mt, initialClientMetadata, "expected to capture a proxied message")
			assert.True(mt, initialClientMetadata.IsHandshake(), "expected first message to be a handshake")

			// 5. Wait 5ms for the connection to become idle.
			time.Sleep(20 * time.Millisecond)

			// 6. Append the `DriverInfoOptions` from the selected test case from
			// the appended metadata section.
			mt.Client.AppendDriverInfo(tc.toAppendDriverInfo)

			// Drain the proxy
			mt.GetProxyCapture().Drain()

			// 7. Send a `ping` command to the server and verify the command
			// succeeds.
			err = mt.Client.Ping(context.Background(), nil)
			require.NoError(mt, err, "Ping error: %v", err)

			// Capture the first message sent after appending driver info.
			updatedClientMetadata := mt.GetProxyCapture().TryNext()
			require.NotNil(mt, updatedClientMetadata, "expected to capture a proxied message")
			assert.True(mt, updatedClientMetadata.IsHandshake(), "expected first message to be a handshake")

			assertbsoncore.HandshakeClientMetadata(mt, initialClientMetadata.Sent.Command,
				updatedClientMetadata.Sent.Command)
		})
	}
}

// Test 8: Empty strings are considered unset when appending metadata identical
// to initial metadata
func TestHandshakeProse_AppendMetadata_EmptyStrings_InitializedClient(t *testing.T) {
	mt := mtest.New(t)

	testCases := []struct {
		name               string
		initialDriverInfo  options.DriverInfo
		toAppendDriverInfo options.DriverInfo
	}{
		{
			name: "name empty",
			initialDriverInfo: options.DriverInfo{
				Name:     "",
				Version:  "1.2",
				Platform: "Library Platform",
			},
			toAppendDriverInfo: options.DriverInfo{
				Name:     "",
				Version:  "1.2",
				Platform: "Library Platform",
			},
		},
		{
			name: "version empty",
			initialDriverInfo: options.DriverInfo{
				Name:     "library",
				Version:  "",
				Platform: "Library Platform",
			},
			toAppendDriverInfo: options.DriverInfo{
				Name:     "library",
				Version:  "",
				Platform: "Library Platform",
			},
		},
		{
			name: "platform empty",
			initialDriverInfo: options.DriverInfo{
				Name:     "library",
				Version:  "1.2",
				Platform: "",
			},
			toAppendDriverInfo: options.DriverInfo{
				Name:     "library",
				Version:  "1.2",
				Platform: "",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // Avoid implicit memory aliasing in for loop.

		// Create a top-level client that can be shared among sub-tests. This is
		// necessary to test appending driver info to an existing client.
		opts := mtest.NewOptions().CreateClient(false).ClientType(mtest.Proxy)
		mt.RunOpts(tc.name, opts, func(mt *mtest.T) {
			// 1. Create a `MongoClient` instance.
			clientOpts := options.Client().
				// Set idle timeout to 1ms to force new connections to be created
				// throughout the lifetime of the test.
				SetMaxConnIdleTime(1 * time.Millisecond).
				SetDriverInfo(&tc.initialDriverInfo)

			mt.ResetClient(clientOpts)

			// 2. Send a `ping` command to the server and verify that the command
			// succeeds.
			err := mt.Client.Ping(context.Background(), nil)
			require.NoError(mt, err, "Ping error: %v", err)

			// 3. Save intercepted `client` document as `initialClientMetadata`.
			initialClientMetadata := mt.GetProxyCapture().TryNext()

			require.NotNil(mt, initialClientMetadata, "expected to capture a proxied message")
			assert.True(mt, initialClientMetadata.IsHandshake(), "expected first message to be a handshake")

			// 4. Wait 5ms for the connection to become idle.
			time.Sleep(20 * time.Millisecond)

			// 5. Append the `DriverInfoOptions` from the selected test case from
			// the appended metadata section.
			mt.Client.AppendDriverInfo(tc.toAppendDriverInfo)

			// Drain the proxy
			mt.GetProxyCapture().Drain()

			// 6. Send a `ping` command to the server and verify the command
			// succeeds.
			err = mt.Client.Ping(context.Background(), nil)
			require.NoError(mt, err, "Ping error: %v", err)

			// 7. Store the response as `updatedClientMetadata`.
			updatedClientMetadata := mt.GetProxyCapture().TryNext()
			require.NotNil(mt, updatedClientMetadata, "expected to capture a proxied message")
			assert.True(mt, updatedClientMetadata.IsHandshake(), "expected first message to be a handshake")

			// 8. Assert that `initialClientMetadata` is identical to `updatedClientMetadata`.
			assertbsoncore.HandshakeClientMetadata(mt, initialClientMetadata.Sent.Command,
				updatedClientMetadata.Sent.Command)
		})
	}
}
