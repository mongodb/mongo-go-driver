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
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/core/uuid"
	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
	"github.com/mongodb/mongo-go-driver/options"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
)

// FindOneAndUpdate handles the full cycle dispatch and execution of a FindOneAndUpdate command against the provided
// topology.
func FindOneAndUpdate(
	ctx context.Context,
	cmd command.FindOneAndUpdate,
	topo *topology.Topology,
	selector description.ServerSelector,
	clientID uuid.UUID,
	pool *session.Pool,
	retryWrite bool,
	registry *bsoncodec.Registry,
	opts ...*options.FindOneAndUpdateOptions,
) (result.FindAndModify, error) {

	ss, err := topo.SelectServer(ctx, selector)
	if err != nil {
		return result.FindAndModify{}, err
	}

	// If no explicit session and deployment supports sessions, start implicit session.
	if cmd.Session == nil && topo.SupportsSessions() {
		cmd.Session, err = session.NewClientSession(pool, clientID, session.Implicit)
		if err != nil {
			return result.FindAndModify{}, err
		}
		defer cmd.Session.EndSession()
	}

	uo := options.MergeFindOneAndUpdateOptions(opts...)
	if uo.ArrayFilters != nil {
		arr, err := uo.ArrayFilters.ToArray()
		if err != nil {
			return result.FindAndModify{}, err
		}

		cmd.Opts = append(cmd.Opts, bsonx.Elem{"arrayFilters", bsonx.Array(arr)})
	}
	if uo.BypassDocumentValidation != nil {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"bypassDocumentValidation", bsonx.Boolean(*uo.BypassDocumentValidation)})
	}
	if uo.Collation != nil {
		if ss.Description().WireVersion.Max < 5 {
			return result.FindAndModify{}, ErrCollation
		}
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"collation", bsonx.Document(uo.Collation.ToDocument())})
	}
	if uo.MaxTime != nil {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"maxTimeMS", bsonx.Int64(int64(*uo.MaxTime / time.Millisecond))})
	}
	if uo.Projection != nil {
		projElem, err := interfaceToElement("fields", uo.Projection, registry)
		if err != nil {
			return result.FindAndModify{}, err
		}

		cmd.Opts = append(cmd.Opts, projElem)
	}
	if uo.ReturnDocument != nil {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"new", bsonx.Boolean(*uo.ReturnDocument == options.After)})
	}
	if uo.Sort != nil {
		sortElem, err := interfaceToElement("sort", uo.Sort, registry)
		if err != nil {
			return result.FindAndModify{}, err
		}

		cmd.Opts = append(cmd.Opts, sortElem)
	}
	if uo.Upsert != nil {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"upsert", bsonx.Boolean(*uo.Upsert)})
	}

	// Execute in a single trip if retry writes not supported, or retry not enabled
	if !retrySupported(topo, ss.Description(), cmd.Session, cmd.WriteConcern) || !retryWrite {
		if cmd.Session != nil {
			cmd.Session.RetryWrite = false // explicitly set to false to prevent encoding transaction number
		}
		return findOneAndUpdate(ctx, cmd, ss, nil)
	}

	cmd.Session.RetryWrite = retryWrite
	cmd.Session.IncrementTxnNumber()

	res, originalErr := findOneAndUpdate(ctx, cmd, ss, nil)

	// Retry if appropriate
	if cerr, ok := originalErr.(command.Error); ok && cerr.Retryable() {
		ss, err := topo.SelectServer(ctx, selector)

		// Return original error if server selection fails or new server does not support retryable writes
		if err != nil || !retrySupported(topo, ss.Description(), cmd.Session, cmd.WriteConcern) {
			return result.FindAndModify{}, originalErr
		}

		return findOneAndUpdate(ctx, cmd, ss, cerr)
	}

	return res, originalErr
}

func findOneAndUpdate(
	ctx context.Context,
	cmd command.FindOneAndUpdate,
	ss *topology.SelectedServer,
	oldErr error,
) (result.FindAndModify, error) {
	desc := ss.Description()
	conn, err := ss.Connection(ctx)
	if err != nil {
		if oldErr != nil {
			return result.FindAndModify{}, oldErr
		}
		return result.FindAndModify{}, err
	}

	if !writeconcern.AckWrite(cmd.WriteConcern) {
		go func() {
			defer func() { _ = recover() }()
			defer conn.Close()

			_, _ = cmd.RoundTrip(ctx, desc, conn)
		}()

		return result.FindAndModify{}, command.ErrUnacknowledgedWrite
	}
	defer conn.Close()

	return cmd.RoundTrip(ctx, desc, conn)
}
