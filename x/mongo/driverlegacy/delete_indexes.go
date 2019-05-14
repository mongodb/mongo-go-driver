// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driverlegacy

import (
	"context"

	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
	"go.mongodb.org/mongo-driver/x/mongo/driver/uuid"
	"go.mongodb.org/mongo-driver/x/network/command"
)

// DropIndexes handles the full cycle dispatch and execution of a dropIndexes
// command against the provided topology.
func DropIndexes(
	ctx context.Context,
	cmd command.DropIndexes,
	topo *topology.Topology,
	selector description.ServerSelector,
	clientID uuid.UUID,
	pool *session.Pool,
	opts ...*options.DropIndexesOptions,
) (bson.Raw, error) {

	ss, err := topo.SelectServerLegacy(ctx, selector)
	if err != nil {
		return nil, err
	}

	conn, err := ss.ConnectionLegacy(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	dio := options.MergeDropIndexesOptions(opts...)
	if dio.MaxTime != nil {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"maxTimeMS", bsonx.Int64(int64(*dio.MaxTime / time.Millisecond))})
	}

	// If no explicit session and deployment supports sessions, start implicit session.
	if cmd.Session == nil && topo.SupportsSessions() {
		cmd.Session, err = session.NewClientSession(pool, clientID, session.Implicit)
		if err != nil {
			return nil, err
		}
		defer cmd.Session.EndSession()
	}

	return cmd.RoundTrip(ctx, ss.Description(), conn)
}
