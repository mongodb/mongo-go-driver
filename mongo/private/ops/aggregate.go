// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"context"
	"time"

	"github.com/10gen/mongo-go-driver/mongo/internal"
	"github.com/10gen/mongo-go-driver/mongo/options"
	"github.com/10gen/mongo-go-driver/mongo/readconcern"
	"github.com/skriptble/wilson/bson"
)

// Aggregate performs an aggregation.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
// TODO GODRIVER-95: Deal with $out and corresponding behavior (read preference primary, write
// concern, etc.)
func Aggregate(ctx context.Context, s *SelectedServer, ns Namespace, readConcern *readconcern.ReadConcern,
	pipeline *bson.Array, opts ...options.AggregateOption) (Cursor, error) {

	if err := ns.validate(); err != nil {
		return nil, err
	}

	command := bson.NewDocument()
	command.Append(bson.C.String("aggregate", ns.Collection), bson.C.Array("pipeline", pipeline))

	var batchSize int32
	cursor := bson.NewDocument()
	command.Append(bson.C.SubDocument("cursor", cursor))

	for _, option := range opts {
		switch t := option.(type) {
		case nil:
			continue
		case options.OptBatchSize:
			batchSize = int32(t)
			option.Option(cursor)
		default:
			option.Option(command)
		}
	}

	if readConcern != nil {
		elem, err := readConcern.MarshalBSONElement()
		if err != nil {
			return nil, err
		}
		command.Append(elem)
	}

	rdr, err := runMayUseSecondary(ctx, s, ns.DB, command)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute aggregate")
	}

	return NewCursor(rdr, batchSize, s)
}

// AggregationOptions are the options for the aggregate command.
type AggregationOptions struct {
	// Whether the server can use stable storage for sorting results.
	AllowDiskUse bool
	// The batch size for fetching results. A zero value indicates the server's default batch size.
	BatchSize int32
	// The maximum execution time.  A zero value indicates no maximum.
	MaxTime time.Duration
}

// LegacyAggregate executes the aggregate command with the given pipeline and options.
//
// The pipeline must encode as a BSON array of pipeline stages.
func LegacyAggregate(ctx context.Context, s *SelectedServer, ns Namespace, pipeline *bson.Array, options AggregationOptions) (Cursor, error) {
	if err := ns.validate(); err != nil {
		return nil, err
	}

	aggregateCommand := bson.NewDocument(
		bson.C.String("aggregate", ns.Collection),
		bson.C.Array("pipeline", pipeline),
		bson.C.SubDocumentFromElements("cursor", bson.C.Int32("batchSize", options.BatchSize)))

	if options.AllowDiskUse {
		aggregateCommand.Append(bson.C.Boolean("allowDiskUse", true))
	}
	if options.MaxTime != 0 {
		aggregateCommand.Append(bson.C.Int64("maxTimeMS", int64(options.MaxTime/time.Millisecond)))
	}

	rdr, err := runMayUseSecondary(ctx, s, ns.DB, aggregateCommand)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute aggregate")
	}

	return NewCursor(rdr, options.BatchSize, s)
}
