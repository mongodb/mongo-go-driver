// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/mongoutil"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/operation"
)

// StreamProcessor is a handle for a specific named stream processor. Holding
// a StreamProcessor does not imply that a processor with the given name
// currently exists on the server.
type StreamProcessor struct {
	parent *StreamProcessors
	name   string
}

// Name returns the processor name.
func (sp *StreamProcessor) Name() string { return sp.name }

// Start issues a startStreamProcessor command. The server requires the
// processor to be in the STOPPED or FAILED state.
func (sp *StreamProcessor) Start(
	ctx context.Context,
	opts ...options.Lister[options.StartStreamProcessorOptions],
) error {
	if ctx == nil {
		ctx = context.Background()
	}

	args, err := mongoutil.NewOptions[options.StartStreamProcessorOptions](opts...)
	if err != nil {
		return err
	}

	sess, release, err := sp.parent.acquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	op := operation.NewStartStreamProcessor(sp.name)
	if args.Workers != nil {
		op = op.Workers(*args.Workers)
	}
	if args.ClearCheckpoints != nil {
		op = op.ClearCheckpoints(*args.ClearCheckpoints)
	}
	if args.StartAtOperationTime != nil {
		op = op.StartAtOperationTime(args.StartAtOperationTime.T, args.StartAtOperationTime.I)
	}
	if args.Tier != nil {
		op = op.Tier(*args.Tier)
	}
	if args.EnableAutoScaling != nil {
		op = op.EnableAutoScaling(*args.EnableAutoScaling)
	}
	if args.Failover != nil {
		if args.Failover.Region == "" {
			return errors.New("failover requires a target region")
		}
		op = op.FailoverRegion(args.Failover.Region)
		if args.Failover.Mode != nil {
			op = op.FailoverMode(*args.Failover.Mode)
		}
		if args.Failover.DryRun != nil {
			op = op.FailoverDryRun(*args.Failover.DryRun)
		}
	}

	op = op.
		Session(sess).
		ClusterClock(sp.parent.client.clock).
		CommandMonitor(sp.parent.client.monitor).
		Crypt(sp.parent.client.cryptFLE).
		Database(streamProcessingAdminDB).
		Deployment(sp.parent.client.deployment).
		ServerSelector(sp.parent.writeSelector()).
		ServerAPI(sp.parent.client.serverAPI).
		Authenticator(sp.parent.client.authenticator)

	return op.Execute(ctx)
}

// Stop issues a stopStreamProcessor command. The processor remains in a
// STOPPED state and can be restarted.
func (sp *StreamProcessor) Stop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	sess, release, err := sp.parent.acquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	op := operation.NewStopStreamProcessor(sp.name).
		Session(sess).
		ClusterClock(sp.parent.client.clock).
		CommandMonitor(sp.parent.client.monitor).
		Crypt(sp.parent.client.cryptFLE).
		Database(streamProcessingAdminDB).
		Deployment(sp.parent.client.deployment).
		ServerSelector(sp.parent.writeSelector()).
		ServerAPI(sp.parent.client.serverAPI).
		Authenticator(sp.parent.client.authenticator)

	return op.Execute(ctx)
}

// Drop issues a dropStreamProcessor command. A dropped processor cannot be
// recovered.
func (sp *StreamProcessor) Drop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	sess, release, err := sp.parent.acquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	op := operation.NewDropStreamProcessor(sp.name).
		Session(sess).
		ClusterClock(sp.parent.client.clock).
		CommandMonitor(sp.parent.client.monitor).
		Crypt(sp.parent.client.cryptFLE).
		Database(streamProcessingAdminDB).
		Deployment(sp.parent.client.deployment).
		ServerSelector(sp.parent.writeSelector()).
		ServerAPI(sp.parent.client.serverAPI).
		Authenticator(sp.parent.client.authenticator)

	return op.Execute(ctx)
}

// Stats issues a getStreamProcessorStats command and returns the raw response
// document. The server returns an error if the processor is not running.
func (sp *StreamProcessor) Stats(
	ctx context.Context,
	opts ...options.Lister[options.GetStreamProcessorStatsOptions],
) (bson.Raw, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	args, err := mongoutil.NewOptions[options.GetStreamProcessorStatsOptions](opts...)
	if err != nil {
		return nil, err
	}

	sess, release, err := sp.parent.acquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	op := operation.NewGetStreamProcessorStats(sp.name)
	if args.Verbose != nil {
		op = op.Verbose(*args.Verbose)
	}
	op = op.
		Session(sess).
		ClusterClock(sp.parent.client.clock).
		CommandMonitor(sp.parent.client.monitor).
		Crypt(sp.parent.client.cryptFLE).
		Database(streamProcessingAdminDB).
		Deployment(sp.parent.client.deployment).
		ServerSelector(sp.parent.readSelector()).
		ReadPreference(sp.parent.client.readPreference).
		ServerAPI(sp.parent.client.serverAPI).
		Authenticator(sp.parent.client.authenticator).
		Retry(driver.RetryOncePerCommand)

	if err := op.Execute(ctx); err != nil {
		return nil, err
	}
	return bson.Raw(op.Result()), nil
}

// StreamProcessorInfo describes a single stream processor as returned by
// getStreamProcessor.
//
// Server-internal fields (tenantID, projectId, processorId, …) are not
// surfaced. Unknown fields on the wire are tolerated and ignored.
type StreamProcessorInfo struct {
	Name            string     `bson:"name"`
	State           string     `bson:"state"`
	Pipeline        []bson.Raw `bson:"pipeline"`
	LastStateChange *time.Time `bson:"lastStateChange,omitempty"`
	ErrorMsg        string     `bson:"errorMsg"`
}

func parseStreamProcessorInfo(raw bson.Raw, bsonOpts *options.BSONOptions, reg *bson.Registry) (*StreamProcessorInfo, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty getStreamProcessor response")
	}
	// The current server wraps the processor document inside a "result"
	// sub-document; the spec describes the fields at the top level. Decode
	// from "result" if present, else fall back to the top-level document so
	// the driver works against either shape.
	target := raw
	if sub, err := raw.LookupErr("result"); err == nil {
		if doc, ok := sub.DocumentOK(); ok {
			target = bson.Raw(doc)
		}
	}
	info := new(StreamProcessorInfo)
	dec := getDecoder(target, bsonOpts, reg)
	if err := dec.Decode(info); err != nil {
		return nil, err
	}
	return info, nil
}
