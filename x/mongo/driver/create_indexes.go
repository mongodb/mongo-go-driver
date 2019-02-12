// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"context"

	"time"

	"drivers.mongodb.org/go/mongo/options"
	"drivers.mongodb.org/go/x/bsonx"
	"drivers.mongodb.org/go/x/mongo/driver/session"
	"drivers.mongodb.org/go/x/mongo/driver/topology"
	"drivers.mongodb.org/go/x/mongo/driver/uuid"
	"drivers.mongodb.org/go/x/network/command"
	"drivers.mongodb.org/go/x/network/description"
	"drivers.mongodb.org/go/x/network/result"
)

// CreateIndexes handles the full cycle dispatch and execution of a createIndexes
// command against the provided topology.
func CreateIndexes(
	ctx context.Context,
	cmd command.CreateIndexes,
	topo *topology.Topology,
	selector description.ServerSelector,
	clientID uuid.UUID,
	pool *session.Pool,
	opts ...*options.CreateIndexesOptions,
) (result.CreateIndexes, error) {

	ss, err := topo.SelectServer(ctx, selector)
	if err != nil {
		return result.CreateIndexes{}, err
	}

	desc := ss.Description()
	if desc.WireVersion.Max < 5 && hasCollation(cmd) {
		return result.CreateIndexes{}, ErrCollation
	}

	conn, err := ss.Connection(ctx)
	if err != nil {
		return result.CreateIndexes{}, err
	}
	defer conn.Close()

	cio := options.MergeCreateIndexesOptions(opts...)
	if cio.MaxTime != nil {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"maxTimeMS", bsonx.Int64(int64(*cio.MaxTime / time.Millisecond))})
	}

	// If no explicit session and deployment supports sessions, start implicit session.
	if cmd.Session == nil && topo.SupportsSessions() {
		cmd.Session, err = session.NewClientSession(pool, clientID, session.Implicit)
		if err != nil {
			return result.CreateIndexes{}, err
		}
		defer cmd.Session.EndSession()
	}

	return cmd.RoundTrip(ctx, ss.Description(), conn)
}

func hasCollation(cmd command.CreateIndexes) bool {
	for _, ind := range cmd.Indexes {
		if _, err := ind.Document().LookupErr("collation"); err == nil {
			return true
		}
	}

	return false
}
