// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"context"

	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/x/bsonx"

	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver/session"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver/topology"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver/uuid"
	"github.com/mongodb/mongo-go-driver/x/network/command"
	"github.com/mongodb/mongo-go-driver/x/network/description"
	"github.com/mongodb/mongo-go-driver/x/network/result"
)

// Insert handles the full cycle dispatch and execution of an insert command against the provided
// topology.
func Insert(
	ctx context.Context,
	cmd command.Insert,
	topo *topology.Topology,
	selector description.ServerSelector,
	clientID uuid.UUID,
	pool *session.Pool,
	retryWrite bool,
	opts ...*options.InsertManyOptions,
) (result.Insert, error) {

	ss, err := topo.SelectServer(ctx, selector)
	if err != nil {
		return result.Insert{}, err
	}

	// If no explicit session and deployment supports sessions, start implicit session.
	if cmd.Session == nil && topo.SupportsSessions() {
		cmd.Session, err = session.NewClientSession(pool, clientID, session.Implicit)
		if err != nil {
			return result.Insert{}, err
		}
		defer cmd.Session.EndSession()
	}

	insertOpts := options.MergeInsertManyOptions(opts...)

	if insertOpts.BypassDocumentValidation != nil && ss.Description().WireVersion.Includes(4) {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"bypassDocumentValidation", bsonx.Boolean(*insertOpts.BypassDocumentValidation)})
	}
	if insertOpts.Ordered != nil {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"ordered", bsonx.Boolean(*insertOpts.Ordered)})
	}

	// Execute in a single trip if retry writes not supported, or retry not enabled
	if !retrySupported(topo, ss.Description(), cmd.Session, cmd.WriteConcern) || !retryWrite {
		if cmd.Session != nil {
			cmd.Session.RetryWrite = false // explicitly set to false to prevent encoding transaction number
		}
		return insert(ctx, cmd, ss, nil)
	}

	// TODO figure out best place to put retry write.  Command shouldn't have to know about this field.
	cmd.Session.RetryWrite = retryWrite
	cmd.Session.IncrementTxnNumber()

	res, originalErr := insert(ctx, cmd, ss, nil)

	// Retry if appropriate
	if cerr, ok := originalErr.(command.Error); ok && cerr.Retryable() ||
		res.WriteConcernError != nil && command.IsWriteConcernErrorRetryable(res.WriteConcernError) {
		ss, err := topo.SelectServer(ctx, selector)

		// Return original error if server selection fails or new server does not support retryable writes
		if err != nil || !retrySupported(topo, ss.Description(), cmd.Session, cmd.WriteConcern) {
			return res, originalErr
		}

		return insert(ctx, cmd, ss, cerr)
	}

	return res, originalErr
}

func insert(
	ctx context.Context,
	cmd command.Insert,
	ss *topology.SelectedServer,
	oldErr error,
) (result.Insert, error) {
	desc := ss.Description()
	conn, err := ss.Connection(ctx)
	if err != nil {
		if oldErr != nil {
			return result.Insert{}, oldErr
		}
		return result.Insert{}, err
	}

	if !writeconcern.AckWrite(cmd.WriteConcern) {
		go func() {
			defer func() { _ = recover() }()
			defer conn.Close()

			_, _ = cmd.RoundTrip(ctx, desc, conn)
		}()

		return result.Insert{}, command.ErrUnacknowledgedWrite
	}
	defer conn.Close()

	return cmd.RoundTrip(ctx, desc, conn)
}
