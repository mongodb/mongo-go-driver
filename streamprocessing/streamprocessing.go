// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package streamprocessing

import (
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Client is a handle for a connection to an Atlas Stream Processing
// workspace. It is a type alias for [mongo.StreamProcessingClient].
type Client = mongo.StreamProcessingClient

// StreamProcessors is a handle for managing stream processors in a
// workspace. It is a type alias for [mongo.StreamProcessors].
type StreamProcessors = mongo.StreamProcessors

// StreamProcessor is a handle for a single named stream processor. It is a
// type alias for [mongo.StreamProcessor].
type StreamProcessor = mongo.StreamProcessor

// Info describes a single stream processor as returned by getStreamProcessor.
// It is a type alias for [mongo.StreamProcessorInfo].
type Info = mongo.StreamProcessorInfo

// SamplesResult is the result of a GetStreamProcessorSamples call. It is a
// type alias for [mongo.GetStreamProcessorSamplesResult].
type SamplesResult = mongo.GetStreamProcessorSamplesResult

// Connect creates a new [Client] for an Atlas Stream Processing workspace.
//
// Connect enforces TLS and defaults authSource to "admin"; otherwise it
// behaves identically to [mongo.Connect].
func Connect(opts ...*options.ClientOptions) (*Client, error) {
	return mongo.ConnectStreamProcessing(opts...)
}

// IsWorkspaceHost reports whether the given hostname looks like an Atlas
// Stream Processing workspace endpoint (matches the pattern
// "atlas-stream-*.a.query.mongodb.net"). Detection is advisory only.
func IsWorkspaceHost(host string) bool {
	return mongo.IsStreamProcessingHost(host)
}
