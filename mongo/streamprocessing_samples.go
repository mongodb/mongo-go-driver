// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/mongoutil"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/operation"
)

// GetStreamProcessorSamplesResult is the result of a
// GetStreamProcessorSamples call.
//
// CursorID is the cursor identifier to pass to the next call. A value of 0
// means the cursor has been exhausted; callers MUST NOT issue another
// GetStreamProcessorSamples for that cursor.
type GetStreamProcessorSamplesResult struct {
	CursorID  int64
	Documents []bson.Raw
}

// GetStreamProcessorSamples retrieves a batch of sampled documents from a
// running stream processor. It abstracts over the two-phase wire protocol:
//
//   - If CursorID is absent or zero, GetStreamProcessorSamples sends
//     startSampleStreamProcessor (using Limit if provided) and then
//     immediately issues a getMoreSampleStreamProcessor on the returned
//     cursorId so that the caller receives documents on the first call.
//   - Otherwise it sends a single getMoreSampleStreamProcessor (using
//     BatchSize if provided) with the supplied CursorID.
//
// Callers MUST stop iterating when the returned CursorID is 0.
func (sp *StreamProcessor) GetStreamProcessorSamples(
	ctx context.Context,
	opts ...options.Lister[options.GetStreamProcessorSamplesOptions],
) (*GetStreamProcessorSamplesResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	args, err := mongoutil.NewOptions[options.GetStreamProcessorSamplesOptions](opts...)
	if err != nil {
		return nil, err
	}

	sess, release, err := sp.parent.acquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	cursorID := int64(0)
	if args.CursorID != nil {
		cursorID = *args.CursorID
	}

	if cursorID == 0 {
		startOp := operation.NewStartSampleStreamProcessor(sp.name)
		if args.Limit != nil {
			startOp = startOp.Limit(*args.Limit)
		}
		startOp = startOp.
			Session(sess).
			ClusterClock(sp.parent.client.clock).
			CommandMonitor(sp.parent.client.monitor).
			Crypt(sp.parent.client.cryptFLE).
			Database(streamProcessingAdminDB).
			Deployment(sp.parent.client.deployment).
			ServerSelector(sp.parent.readSelector()).
			ServerAPI(sp.parent.client.serverAPI).
			Authenticator(sp.parent.client.authenticator)
		if err := startOp.Execute(ctx); err != nil {
			return nil, err
		}
		cursorID = startOp.CursorID()
		if cursorID == 0 {
			// Server returned an exhausted cursor immediately.
			return &GetStreamProcessorSamplesResult{}, nil
		}
	}

	getMoreOp := operation.NewGetMoreSampleStreamProcessor(sp.name, cursorID)
	if args.BatchSize != nil {
		getMoreOp = getMoreOp.BatchSize(*args.BatchSize)
	}
	getMoreOp = getMoreOp.
		Session(sess).
		ClusterClock(sp.parent.client.clock).
		CommandMonitor(sp.parent.client.monitor).
		Crypt(sp.parent.client.cryptFLE).
		Database(streamProcessingAdminDB).
		Deployment(sp.parent.client.deployment).
		ServerSelector(sp.parent.readSelector()).
		ServerAPI(sp.parent.client.serverAPI).
		Authenticator(sp.parent.client.authenticator)

	if err := getMoreOp.Execute(ctx); err != nil {
		return nil, err
	}

	batch := getMoreOp.ResultBatch()
	docs := make([]bson.Raw, len(batch))
	for i, d := range batch {
		// Copy so the caller can retain documents past the next Execute.
		docs[i] = append(bson.Raw(nil), bson.Raw(d)...)
	}
	return &GetStreamProcessorSamplesResult{
		CursorID:  getMoreOp.ResultCursorID(),
		Documents: docs,
	}, nil
}
