// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/core/uuid"
	"github.com/mongodb/mongo-go-driver/options"
	"time"
)

// Find handles the full cycle dispatch and execution of a find command against the provided
// topology.
func Find(
	ctx context.Context,
	cmd command.Find,
	topo *topology.Topology,
	selector description.ServerSelector,
	clientID uuid.UUID,
	pool *session.Pool,
	registry *bsoncodec.Registry,
	opts ...*options.FindOptions,
) (command.Cursor, error) {

	ss, err := topo.SelectServer(ctx, selector)
	if err != nil {
		return nil, err
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
		cmd.Session, err = session.NewClientSession(pool, clientID, session.Implicit, nil)
		if err != nil {
			return nil, err
		}
	}

	fo := options.ToFindOptions(opts...)
	if fo.AllowPartialResults != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Boolean("allowPartialResults", *fo.AllowPartialResults))
	}
	if fo.BatchSize != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Int32("batchSize", *fo.BatchSize))
		if fo.Limit != nil && *fo.BatchSize != 0 && *fo.Limit <= int64(*fo.BatchSize) {
			cmd.Opts = append(cmd.Opts, bson.EC.Boolean("singleBatch", true))
		}
	}
	if fo.Collation != nil {
		if desc.WireVersion.Max < 5 {
			return nil, ErrCollation
		}
		cmd.Opts = append(cmd.Opts, bson.EC.SubDocument("collation", fo.Collation.ToDocument()))
	}
	if fo.Comment != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.String("comment", *fo.Comment))
	}
	if fo.CursorType != nil {
		switch *fo.CursorType {
		case options.Tailable:
			cmd.Opts = append(cmd.Opts, bson.EC.Boolean("tailable", true))
		case options.TailableAwait:
			cmd.Opts = append(cmd.Opts, bson.EC.Boolean("tailable", true), bson.EC.Boolean("awaitData", true))
		}
	}
	if fo.Hint != nil {
		hintElem, err := interfaceToElement("hint", fo.Hint, registry)
		if err != nil {
			return nil, err
		}

		cmd.Opts = append(cmd.Opts, hintElem)
	}
	if fo.Limit != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Int64("limit", *fo.Limit))
	}
	if fo.Max != nil {
		maxElem, err := interfaceToElement("max", fo.Max, registry)
		if err != nil {
			return nil, err
		}

		cmd.Opts = append(cmd.Opts, maxElem)
	}
	if fo.MaxAwaitTime != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Int64("maxAwaitTimeMS", int64(*fo.MaxAwaitTime/time.Millisecond)))
	}
	if fo.MaxTime != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Int64("maxTimeMS", int64(*fo.MaxTime/time.Millisecond)))
	}
	if fo.Min != nil {
		minElem, err := interfaceToElement("min", fo.Min, registry)
		if err != nil {
			return nil, err
		}

		cmd.Opts = append(cmd.Opts, minElem)
	}
	if fo.NoCursorTimeout != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Boolean("noCursorTimeout", *fo.NoCursorTimeout))
	}
	if fo.OplogReplay != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Boolean("oplogReplay", *fo.OplogReplay))
	}
	if fo.Projection != nil {
		projElem, err := interfaceToElement("projection", fo.Projection, registry)
		if err != nil {
			return nil, err
		}

		cmd.Opts = append(cmd.Opts, projElem)
	}
	if fo.ReturnKey != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Boolean("returnKey", *fo.ReturnKey))
	}
	if fo.ShowRecordID != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Boolean("showRecordId", *fo.ShowRecordID))
	}
	if fo.Skip != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Int64("skip", *fo.Skip))
	}
	if fo.Snapshot != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Boolean("snapshot", *fo.Snapshot))
	}
	if fo.Sort != nil {
		sortElem, err := interfaceToElement("sort", fo.Sort, registry)
		if err != nil {
			return nil, err
		}

		cmd.Opts = append(cmd.Opts, sortElem)
	}

	return cmd.RoundTrip(ctx, desc, ss, conn)
}
