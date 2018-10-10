// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dispatch

import (
	"context"
	"errors"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/options"

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

	aggOpts := options.ToAggregateOptions(opts...)

	if aggOpts.AllowDiskUse != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Boolean("allowDiskUse", *aggOpts.AllowDiskUse))
	}
	if aggOpts.BatchSize != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Int32("batchSize", *aggOpts.BatchSize))
	}
	if aggOpts.BypassDocumentValidation != nil && desc.WireVersion.Includes(4) {
		cmd.Opts = append(cmd.Opts, bson.EC.Boolean("bypassDocumentValidation", *aggOpts.BypassDocumentValidation))
	}
	if aggOpts.Collation != nil {
		if desc.WireVersion.Max < 5 {
			return nil, errors.New("collation specified for invalid server version")
		}
		cmd.Opts = append(cmd.Opts, bson.EC.SubDocument("collation", aggOpts.Collation.ToDocument()))
	}
	if aggOpts.MaxTimeMS != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Int64("maxTimeMS", *aggOpts.MaxTimeMS))
	}
	if aggOpts.MaxTimeMS != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.Int64("maxAwaitTimeMS", *aggOpts.MaxAwaitTimeMS))
	}
	if aggOpts.Comment != nil {
		cmd.Opts = append(cmd.Opts, bson.EC.String("comment", *aggOpts.Comment))
	}
	if aggOpts.Hint != nil {
		switch t := (aggOpts.Hint).(type) {
		case string:
			cmd.Opts = append(cmd.Opts, bson.EC.String("hint", t))
		case *bson.Document:
			cmd.Opts = append(cmd.Opts, bson.EC.SubDocument("hint", t))
		default:
			h, err := bsoncodec.Marshal(t)
			if err != nil {
				return nil, err
			}
			cmd.Opts = append(cmd.Opts, bson.EC.SubDocumentFromReader("hint", h))
		}
	}

	return cmd.RoundTrip(ctx, desc, ss, conn)
}
