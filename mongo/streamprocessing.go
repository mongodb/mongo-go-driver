// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"crypto/tls"
	"strings"

	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// streamProcessingAdminDB is the database name used for routing all Atlas
// Stream Processing commands. Workspace endpoint routing is performed at the
// network layer; the database name is not used as a routing key, but the
// driver sends commands against "admin" for consistency with mongosh and the
// authentication database.
const streamProcessingAdminDB = "admin"

// StreamProcessingClient is a handle for connecting to an Atlas Stream
// Processing workspace. It exposes the commands defined by the Atlas Stream
// Processing wire protocol.
//
// A StreamProcessingClient is distinct from a Client: it enforces TLS,
// defaults authSource to "admin", and exposes only commands meaningful for
// stream processing workspaces.
//
// Most callers should obtain a StreamProcessingClient via the
// streamprocessing.Connect factory, which re-exports this type.
type StreamProcessingClient struct {
	client *Client
}

// ConnectStreamProcessing creates a new StreamProcessingClient configured for
// an Atlas Stream Processing workspace endpoint.
//
// ConnectStreamProcessing enforces the following workspace requirements:
//
//   - TLS is always enabled. If the caller did not configure a TLS config,
//     a default *tls.Config is set.
//   - authSource defaults to "admin" when credentials are provided without
//     an explicit auth source.
//
// Like [Connect], ConnectStreamProcessing returns without verifying that the
// workspace is reachable; call Ping to verify connectivity.
func ConnectStreamProcessing(opts ...*options.ClientOptions) (*StreamProcessingClient, error) {
	merged := options.MergeClientOptions(opts...)
	applyStreamProcessingDefaults(merged)

	c, err := Connect(merged)
	if err != nil {
		return nil, err
	}
	return &StreamProcessingClient{client: c}, nil
}

// applyStreamProcessingDefaults enforces TLS and authSource=admin on the
// merged ClientOptions.
func applyStreamProcessingDefaults(opts *options.ClientOptions) {
	if opts.TLSConfig == nil {
		opts.TLSConfig = &tls.Config{}
	}
	if opts.Auth != nil && opts.Auth.AuthSource == "" {
		opts.Auth.AuthSource = streamProcessingAdminDB
	}
}

// IsStreamProcessingHost reports whether the provided hostname looks like an
// Atlas Stream Processing workspace endpoint. Detection is advisory only; the
// driver does not require it.
func IsStreamProcessingHost(host string) bool {
	return strings.HasPrefix(host, "atlas-stream-") &&
		strings.HasSuffix(host, ".a.query.mongodb.net")
}

// Client returns the underlying *Client used by this StreamProcessingClient.
// This is exposed for advanced cases (e.g. running unsupported commands via
// RunCommand); ordinary code should use StreamProcessors instead.
func (spc *StreamProcessingClient) Client() *Client { return spc.client }

// StreamProcessors returns a handle for managing stream processors in this
// workspace.
func (spc *StreamProcessingClient) StreamProcessors() *StreamProcessors {
	return &StreamProcessors{client: spc.client}
}

// Disconnect closes the underlying connections, waiting for in-flight
// operations to complete. After Disconnect, the StreamProcessingClient must
// not be used.
func (spc *StreamProcessingClient) Disconnect(ctx context.Context) error {
	return spc.client.Disconnect(ctx)
}

// Ping verifies that the workspace endpoint is reachable.
func (spc *StreamProcessingClient) Ping(ctx context.Context, rp *readpref.ReadPref) error {
	return spc.client.Ping(ctx, rp)
}
