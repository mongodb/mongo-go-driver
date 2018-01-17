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

	command := bson.NewDocument(2 + uint(len(opts)))
	command.Append(bson.C.String("aggregate", ns.Collection), bson.C.Array("pipeline", pipeline))
	// command := oldbson.D{
	// 	{Name: "aggregate", Value: ns.Collection},
	// 	{Name: "pipeline", Value: pipeline},
	// }

	// cursorArg := oldbson.D{}
	// batchSize := int32(0)
	var batchSize int32
	cursor := bson.NewDocument(1)
	command.Append(bson.C.SubDocument("cursor", cursor))

	for _, option := range opts {
		switch t := option.(type) {
		case options.OptBatchSize:
			batchSize = int32(t)
			option.Option(cursor)
		default:
			option.Option(command)
		}
		// switch name := option.AggregateName(); name {
		// case "batchSize":
		// 	batchSize = int32(option.AggregateValue().(options.OptBatchSize))
		// 	cursorArg.AppendElem("batchSize", batchSize)
		// case "maxTimeMS":
		// 	command.AppendElem(
		// 		name,
		// 		int64(option.AggregateValue().(time.Duration)/time.Millisecond),
		// 	)
		// default:
		// 	command.AppendElem(name, option.AggregateValue())
		// }
	}

	// command.AppendElem("cursor", cursorArg)

	if readConcern != nil {
		elem, err := readConcern.MarshalBSONElement()
		if err != nil {
			return nil, err
		}
		command.Append(elem)
		// command.AppendElem("readConcern", readConcern)
	}

	var result cursorReturningResult

	err := runMayUseSecondary(ctx, s, ns.DB, command, &result)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute aggregate")
	}

	return NewCursor(&result.Cursor, batchSize, s)
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
func LegacyAggregate(ctx context.Context, s *SelectedServer, ns Namespace, pipeline interface{}, options AggregationOptions) (Cursor, error) {
	if err := ns.validate(); err != nil {
		return nil, err
	}

	aggregateCommand := struct {
		Collection   string         `bson:"aggregate"`
		AllowDiskUse bool           `bson:"allowDiskUse,omitempty"`
		MaxTimeMS    int64          `bson:"maxTimeMS,omitempty"`
		Pipeline     interface{}    `bson:"pipeline"`
		Cursor       *cursorRequest `bson:"cursor"`
	}{
		Collection:   ns.Collection,
		AllowDiskUse: options.AllowDiskUse,
		MaxTimeMS:    int64(options.MaxTime / time.Millisecond),
		Pipeline:     pipeline,
		Cursor: &cursorRequest{
			BatchSize: options.BatchSize,
		},
	}

	var result cursorReturningResult

	err := runMayUseSecondary(ctx, s, ns.DB, aggregateCommand, &result)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute aggregate")
	}

	return NewCursor(&result.Cursor, options.BatchSize, s)
}
