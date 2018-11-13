// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/options"
	"github.com/mongodb/mongo-go-driver/x/bsonx"

	"time"

	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/core/uuid"
)

// Aggregate handles the full cycle dispatch and execution of an aggregate command against the provided
// topology.
func Aggregate(
	ctx context.Context,
	cmd command.Aggregate,
	topo *topology.Topology,
	readSelector, writeSelector description.ServerSelector,
	clientID uuid.UUID,
	pool *session.Pool,
	registry *bsoncodec.Registry,
	opts ...*options.AggregateOptions,
) (command.Cursor, error) {

	dollarOut := cmd.HasDollarOut()

	var ss *topology.SelectedServer
	var err error
	switch dollarOut {
	case true:
		ss, err = topo.SelectServer(ctx, writeSelector)
		if err != nil {
			return nil, err
		}
	case false:
		ss, err = topo.SelectServer(ctx, readSelector)
		if err != nil {
			return nil, err
		}
	}

	desc := ss.Description()
	conn, err := ss.Connection(ctx)
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	rp, err := getReadPrefBasedOnTransaction(cmd.ReadPref, cmd.Session)
	if err != nil {
		return nil, err
	}
	cmd.ReadPref = rp

	// If no explicit session and deployment supports sessions, start implicit session.
	if cmd.Session == nil && topo.SupportsSessions() {
		cmd.Session, err = session.NewClientSession(pool, clientID, session.Implicit)
		if err != nil {
			return nil, err
		}
	}

	aggOpts := options.MergeAggregateOptions(opts...)

	if aggOpts.AllowDiskUse != nil {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"allowDiskUse", bsonx.Boolean(*aggOpts.AllowDiskUse)})
	}
	if aggOpts.BatchSize != nil {
		elem := bsonx.Elem{"batchSize", bsonx.Int32(*aggOpts.BatchSize)}
		cmd.Opts = append(cmd.Opts, elem)
		cmd.CursorOpts = append(cmd.CursorOpts, elem)
	}
	if aggOpts.BypassDocumentValidation != nil && desc.WireVersion.Includes(4) {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"bypassDocumentValidation", bsonx.Boolean(*aggOpts.BypassDocumentValidation)})
	}
	if aggOpts.Collation != nil {
		if desc.WireVersion.Max < 5 {
			return nil, ErrCollation
		}
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"collation", bsonx.Document(aggOpts.Collation.ToDocument())})
	}
	if aggOpts.MaxTime != nil {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"maxTimeMS", bsonx.Int64(int64(*aggOpts.MaxTime / time.Millisecond))})
	}
	if aggOpts.MaxAwaitTime != nil {
		// specified as maxTimeMS on getMore commands
		cmd.CursorOpts = append(cmd.CursorOpts, bsonx.Elem{
			"maxTimeMS", bsonx.Int64(int64(*aggOpts.MaxAwaitTime / time.Millisecond)),
		})
	}
	if aggOpts.Comment != nil {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"comment", bsonx.String(*aggOpts.Comment)})
	}
	if aggOpts.Hint != nil {
		hintElem, err := interfaceToElement("hint", aggOpts.Hint, registry)
		if err != nil {
			return nil, err
		}

		cmd.Opts = append(cmd.Opts, hintElem)
	}

	return cmd.RoundTrip(ctx, desc, ss, conn)
}
