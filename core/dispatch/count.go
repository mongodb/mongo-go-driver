// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dispatch

import (
	"context"
	"time"

	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/x/bsonx"

	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/core/uuid"
	"github.com/mongodb/mongo-go-driver/options"
)

// Count handles the full cycle dispatch and execution of a count command against the provided
// topology.
func Count(
	ctx context.Context,
	cmd command.Count,
	topo *topology.Topology,
	selector description.ServerSelector,
	clientID uuid.UUID,
	pool *session.Pool,
	registry *bsoncodec.Registry,
	opts ...*options.CountOptions,
) (int64, error) {

	ss, err := topo.SelectServer(ctx, selector)
	if err != nil {
		return 0, err
	}

	desc := ss.Description()
	conn, err := ss.Connection(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	rp, err := getReadPrefBasedOnTransaction(cmd.ReadPref, cmd.Session)
	if err != nil {
		return 0, err
	}
	cmd.ReadPref = rp

	// If no explicit session and deployment supports sessions, start implicit session.
	if cmd.Session == nil && topo.SupportsSessions() {
		cmd.Session, err = session.NewClientSession(pool, clientID, session.Implicit)
		if err != nil {
			return 0, err
		}
		defer cmd.Session.EndSession()
	}

	countOpts := options.MergeCountOptions(opts...)

	if countOpts.Limit != nil {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"limit", bsonx.Int64(*countOpts.Limit)})
	}
	if countOpts.MaxTime != nil {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{
			"maxTimeMS", bsonx.Int64(int64(time.Duration(*countOpts.MaxTime) / time.Millisecond)),
		})
	}
	if countOpts.Skip != nil {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"skip", bsonx.Int64(*countOpts.Skip)})
	}
	if countOpts.Collation != nil {
		if desc.WireVersion.Max < 5 {
			return 0, ErrCollation
		}
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"collation", bsonx.Document(countOpts.Collation.ToDocument())})
	}
	if countOpts.Hint != nil {
		hintElem, err := interfaceToElement("hint", countOpts.Hint, registry)
		if err != nil {
			return 0, err
		}

		cmd.Opts = append(cmd.Opts, hintElem)
	}

	return cmd.RoundTrip(ctx, desc, conn)
}
