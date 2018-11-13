// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dispatch

import (
	"context"
	"time"

	"github.com/mongodb/mongo-go-driver/options"
	"github.com/mongodb/mongo-go-driver/x/bsonx"

	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/core/uuid"
)

// Distinct handles the full cycle dispatch and execution of a distinct command against the provided
// topology.
func Distinct(
	ctx context.Context,
	cmd command.Distinct,
	topo *topology.Topology,
	selector description.ServerSelector,
	clientID uuid.UUID,
	pool *session.Pool,
	opts ...*options.DistinctOptions,
) (result.Distinct, error) {

	ss, err := topo.SelectServer(ctx, selector)
	if err != nil {
		return result.Distinct{}, err
	}

	desc := ss.Description()
	conn, err := ss.Connection(ctx)
	if err != nil {
		return result.Distinct{}, err
	}
	defer conn.Close()

	rp, err := getReadPrefBasedOnTransaction(cmd.ReadPref, cmd.Session)
	if err != nil {
		return result.Distinct{}, err
	}
	cmd.ReadPref = rp

	// If no explicit session and deployment supports sessions, start implicit session.
	if cmd.Session == nil && topo.SupportsSessions() {
		cmd.Session, err = session.NewClientSession(pool, clientID, session.Implicit)
		if err != nil {
			return result.Distinct{}, err
		}
		defer cmd.Session.EndSession()
	}

	distinctOpts := options.MergeDistinctOptions(opts...)

	if distinctOpts.MaxTime != nil {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{
			"maxTimeMS", bsonx.Int64(int64(time.Duration(*distinctOpts.MaxTime) / time.Millisecond)),
		})
	}
	if distinctOpts.Collation != nil {
		if desc.WireVersion.Max < 5 {
			return result.Distinct{}, ErrCollation
		}
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"collation", bsonx.Document(distinctOpts.Collation.ToDocument())})
	}

	return cmd.RoundTrip(ctx, desc, conn)
}
