// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dispatch

import (
	"context"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
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
	opts *options.CountOptions,
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

	if opts.MaxTimeMS != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Int64("maxTimeMS", int64(time.Duration(*opts.MaxTimeMS)/time.Millisecond)))
	}
	if opts.Skip != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Int64("skip", *opts.Skip))
	}
	if opts.Limit != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Int64("limit", *opts.Limit))
	}
	if opts.Collation != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.SubDocument("collation", opts.Collation.ToDocument()))
	}
	if opts.Hint != nil {
		switch t := (*opts.Hint).(type) {
		case *options.HintString:
			cmd.Opts = append(cmd.Opts, bson.EC.String("hint", t.S))
		case *options.HintDocument:
			cmd.Opts = append(cmd.Opts, bson.EC.SubDocument("hint", t.D))
		}
	}

	return cmd.RoundTrip(ctx, desc, conn)
}
