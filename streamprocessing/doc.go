// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package streamprocessing exposes a client for Atlas Stream Processing
// (ASP) workspaces.
//
// A workspace is a dedicated Atlas endpoint that hosts one or more stream
// processors. Workspace endpoints use the standard "mongodb://" URI scheme;
// the hostname follows the pattern
//
//	mongodb://atlas-stream-<workspaceId>-<suffix>.<region>.a.query.mongodb.net/
//
// TLS is required and authentication uses the admin database by default.
//
// The package re-exports a small set of types from the parent mongo package
// so callers can write
//
//	import (
//	    "go.mongodb.org/mongo-driver/v2/streamprocessing"
//	    "go.mongodb.org/mongo-driver/v2/mongo/options"
//	)
//
//	client, err := streamprocessing.Connect(options.Client().ApplyURI(uri))
//	if err != nil { ... }
//	defer client.Disconnect(ctx)
//
//	sps := client.StreamProcessors()
//	if err := sps.Create(ctx, "agg-1", pipeline); err != nil { ... }
//	sp := sps.Get("agg-1")
//	if err := sp.Start(ctx, nil); err != nil { ... }
//
// The wire protocol and behavioral semantics follow the Atlas Stream
// Processing driver specification.
package streamprocessing
