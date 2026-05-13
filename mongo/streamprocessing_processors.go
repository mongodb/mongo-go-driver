// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/mongoutil"
	"go.mongodb.org/mongo-driver/v2/internal/serverselector"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/operation"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// StreamProcessors is a handle for managing stream processors in a workspace.
type StreamProcessors struct {
	client *Client
}

// Create issues a createStreamProcessor command for a new processor with the
// given name and aggregation pipeline.
//
// The pipeline must be an order-preserving Go value (e.g. mongo.Pipeline,
// []bson.D, []bson.Raw). Map types are rejected.
func (sps *StreamProcessors) Create(
	ctx context.Context,
	name string,
	pipeline any,
	opts ...options.Lister[options.CreateStreamProcessorOptions],
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if name == "" {
		return errors.New("stream processor name must not be empty")
	}

	pipelineArr, _, err := marshalAggregatePipeline(pipeline, sps.client.bsonOpts, sps.client.registry)
	if err != nil {
		return err
	}

	args, err := mongoutil.NewOptions[options.CreateStreamProcessorOptions](opts...)
	if err != nil {
		return err
	}

	sess, releaseSession, err := sps.acquireSession(ctx)
	if err != nil {
		return err
	}
	defer releaseSession()

	op := operation.NewCreateStreamProcessor(name, bsoncore.Document(pipelineArr))
	if args.DLQ != nil {
		op = op.DLQ(bsoncore.Document(args.DLQ))
	}
	if args.StreamMetaFieldName != nil {
		op = op.StreamMetaFieldName(*args.StreamMetaFieldName)
	}
	if args.Tier != nil {
		op = op.Tier(*args.Tier)
	}
	if args.FailoverEnabled != nil {
		op = op.FailoverEnabled(*args.FailoverEnabled)
	}

	op = op.
		Session(sess).
		ClusterClock(sps.client.clock).
		CommandMonitor(sps.client.monitor).
		Crypt(sps.client.cryptFLE).
		Database(streamProcessingAdminDB).
		Deployment(sps.client.deployment).
		ServerSelector(sps.writeSelector()).
		ServerAPI(sps.client.serverAPI).
		Authenticator(sps.client.authenticator)

	return op.Execute(ctx)
}

// Get returns a handle for an existing stream processor by name. Get does not
// verify that a processor with this name exists on the server.
func (sps *StreamProcessors) Get(name string) *StreamProcessor {
	return &StreamProcessor{parent: sps, name: name}
}

// GetInfo issues a getStreamProcessor command for the processor with the
// given name.
func (sps *StreamProcessors) GetInfo(ctx context.Context, name string) (*StreamProcessorInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if name == "" {
		return nil, errors.New("stream processor name must not be empty")
	}

	sess, releaseSession, err := sps.acquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer releaseSession()

	op := operation.NewGetStreamProcessor(name).
		Session(sess).
		ClusterClock(sps.client.clock).
		CommandMonitor(sps.client.monitor).
		Crypt(sps.client.cryptFLE).
		Database(streamProcessingAdminDB).
		Deployment(sps.client.deployment).
		ServerSelector(sps.readSelector()).
		ReadPreference(sps.client.readPreference).
		ServerAPI(sps.client.serverAPI).
		Authenticator(sps.client.authenticator).
		Retry(driver.RetryOncePerCommand)

	if err := op.Execute(ctx); err != nil {
		return nil, err
	}
	return parseStreamProcessorInfo(bson.Raw(op.Result()), sps.client.bsonOpts, sps.client.registry)
}

// acquireSession returns either the explicit session from the context or an
// implicit session from the client's session pool. The returned release
// callback ends the implicit session (if any).
func (sps *StreamProcessors) acquireSession(ctx context.Context) (*session.Client, func(), error) {
	sess := sessionFromContext(ctx)
	implicit := false
	if sess == nil && sps.client.sessionPool != nil {
		sess = session.NewImplicitClientSession(sps.client.sessionPool, sps.client.id)
		implicit = true
	}
	if err := sps.client.validSession(sess); err != nil {
		if implicit {
			closeImplicitSession(sess)
		}
		return nil, func() {}, err
	}
	if implicit {
		return sess, func() { closeImplicitSession(sess) }, nil
	}
	return sess, func() {}, nil
}

func (sps *StreamProcessors) readSelector() description.ServerSelector {
	return &serverselector.Composite{
		Selectors: []description.ServerSelector{
			&serverselector.ReadPref{ReadPref: sps.client.readPreference},
			&serverselector.Latency{Latency: sps.client.localThreshold},
		},
	}
}

func (sps *StreamProcessors) writeSelector() description.ServerSelector {
	return &serverselector.Composite{
		Selectors: []description.ServerSelector{
			&serverselector.Write{},
			&serverselector.Latency{Latency: sps.client.localThreshold},
		},
	}
}
