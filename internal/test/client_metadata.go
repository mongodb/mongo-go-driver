// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package test

import (
	"runtime"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/version"
)

type clientMetadataOptions struct {
	appName        string
	driverName     string
	driverVersion  string
	driverPlatform string
	envName        string
	envTimeoutSec  *int
	envMemoryMB    *int
	envRegion      string
}

// ClientMetadataOption represents a configuration option for building client
// metadata.
type ClientMetadataOption func(*clientMetadataOptions)

// WithClientMetadataAppName sets the application name included in client metadata.
func WithClientMetadataAppName(name string) ClientMetadataOption {
	return func(o *clientMetadataOptions) {
		o.appName = name
	}
}

// WithClientMetadataDriverName sets the driver name (e.g., "mongo-go-driver").
func WithClientMetadataDriverName(name string) ClientMetadataOption {
	return func(o *clientMetadataOptions) {
		o.driverName = name
	}
}

// WithClientMetadataDriverVersion sets the driver version (e.g., "1.16.0").
func WithClientMetadataDriverVersion(version string) ClientMetadataOption {
	return func(o *clientMetadataOptions) {
		o.driverVersion = version
	}
}

// WithClientMetadataDriverPlatform sets the driver platform string
// (e.g., "go1.22.5 gc linux/amd64").
func WithClientMetadataDriverPlatform(platform string) ClientMetadataOption {
	return func(o *clientMetadataOptions) {
		o.driverPlatform = platform
	}
}

// WithClientMetadataEnvName sets the execution environment name
// (e.g., "AWS Lambda", "GCP Cloud Functions", "Kubernetes").
func WithClientMetadataEnvName(name string) ClientMetadataOption {
	return func(o *clientMetadataOptions) {
		o.envName = name
	}
}

// WithClientMetadataEnvTimeoutSec sets the execution timeout in seconds.
// Pass nil to indicate "unspecified" or "not applicable".
func WithClientMetadataEnvTimeoutSec(timeoutSec *int) ClientMetadataOption {
	return func(o *clientMetadataOptions) {
		o.envTimeoutSec = timeoutSec
	}
}

// WithClientMetadataEnvMemoryMB sets the memory limit in megabytes.
// Pass nil to indicate "unspecified" or "not applicable".
func WithClientMetadataEnvMemoryMB(memoryMB *int) ClientMetadataOption {
	return func(o *clientMetadataOptions) {
		o.envMemoryMB = memoryMB
	}
}

// WithClientMetadataEnvRegion sets the deployment/region identifier
// (e.g., "us-east-1", "europe-west1").
func WithClientMetadataEnvRegion(region string) ClientMetadataOption {
	return func(o *clientMetadataOptions) {
		o.envRegion = region
	}
}

// EncodeClientMetadata constructs the WM byte slice that represents the client
// metadata document for the given options with the intent of comparing to an
// actual handshake wire message:
//
//	{
//		application: {
//			name: "<string>"
//		},
//		driver: {
//			name: "<string>",
//			version: "<string>"
//		},
//		platform: "<string>",
//		os: {
//			type: "<string>",
//			name: "<string>",
//			architecture: "<string>",
//			version: "<string>"
//		},
//		env: {
//			name: "<string>",
//			timeout_sec: 42,
//			memory_mb: 1024,
//			region: "<string>",
//			container: {
//				runtime: "<string>",
//				orchestrator: "<string>"
//			}
//		}
//	}
//
//	This function was not put in mtest since it could be used in non-integration
//	test conditions.
func EncodeClientMetadata(t testing.TB, opts ...ClientMetadataOption) []byte {
	t.Helper()

	cfg := clientMetadataOptions{}
	for _, apply := range opts {
		apply(&cfg)
	}

	var (
		driverName    = "mongo-go-driver" // Default
		driverVersion = version.Driver
		platform      = runtime.Version() // Default
	)

	if cfg.driverName != "" {
		driverName = driverName + "|" + cfg.driverName
	}

	if cfg.driverVersion != "" {
		driverVersion = driverVersion + "|" + cfg.driverVersion
	}

	if cfg.driverPlatform != "" {
		platform = platform + "|" + cfg.driverPlatform
	}

	elems := bson.D{}

	if cfg.appName != "" {
		elems = append(elems, bson.E{Key: "application", Value: bson.D{
			{Key: "name", Value: cfg.appName},
		}})
	}

	elems = append(elems, bson.D{
		{Key: "driver", Value: bson.D{
			{Key: "name", Value: driverName},
			{Key: "version", Value: driverVersion},
		}},
		{Key: "os", Value: bson.D{
			{Key: "type", Value: runtime.GOOS},
			{Key: "architecture", Value: runtime.GOARCH},
		}},
	}...)

	elems = append(elems, bson.E{Key: "platform", Value: platform})

	envElems := bson.D{}
	if cfg.envName != "" {
		envElems = append(envElems, bson.E{Key: "name", Value: cfg.envName})
	}

	if cfg.envMemoryMB != nil {
		envElems = append(envElems, bson.E{Key: "memory_mb", Value: *cfg.envMemoryMB})
	}

	if cfg.envRegion != "" {
		envElems = append(envElems, bson.E{Key: "region", Value: cfg.envRegion})
	}

	if cfg.envTimeoutSec != nil {
		envElems = append(envElems, bson.E{Key: "timeout_sec", Value: *cfg.envTimeoutSec})
	}

	if len(envElems) > 0 {
		elems = append(elems, bson.E{Key: "env", Value: envElems})
	}

	bytes, err := bson.Marshal(elems)
	require.NoError(t, err)

	return bytes
}
