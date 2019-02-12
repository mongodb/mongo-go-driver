// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"context"
	"drivers.mongodb.org/go/bson/bsoncodec"
	"drivers.mongodb.org/go/x/bsonx"

	"drivers.mongodb.org/go/mongo/options"
	"drivers.mongodb.org/go/x/mongo/driver/session"
	"drivers.mongodb.org/go/x/mongo/driver/topology"
	"drivers.mongodb.org/go/x/mongo/driver/uuid"
	"drivers.mongodb.org/go/x/network/command"
	"drivers.mongodb.org/go/x/network/description"
	"time"
)

// CountDocuments handles the full cycle dispatch and execution of a countDocuments command against the provided
// topology.
func CountDocuments(
	ctx context.Context,
	cmd command.CountDocuments,
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

	// ignore Skip and Limit because we already have these options in the pipeline
	if countOpts.MaxTime != nil {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{
			"maxTimeMS", bsonx.Int64(int64(*countOpts.MaxTime / time.Millisecond)),
		})
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
