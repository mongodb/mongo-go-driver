// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/topology"
)

// AbortTransaction handles the full cycle dispatch and execution of abortting a transaction
// against the provided topology.
func AbortTransaction(
	ctx context.Context,
	cmd command.AbortTransaction,
	topo *topology.Topology,
	selector description.ServerSelector,
) (result.TransactionResult, error) {
	res, err := abortTransaction(ctx, cmd, topo, selector, nil)
	if cerr, ok := err.(command.Error); ok && err != nil {
		// Retry if appropriate
		if cerr.Retryable() {
			res, err = abortTransaction(ctx, cmd, topo, selector, cerr)
		}
	}
	return res, err
}

func abortTransaction(
	ctx context.Context,
	cmd command.AbortTransaction,
	topo *topology.Topology,
	selector description.ServerSelector,
	oldErr error,
) (result.TransactionResult, error) {
	ss, err := topo.SelectServer(ctx, selector)
	if err != nil {
		// If retrying server selection, return the original error if it fails
		if oldErr != nil {
			return result.TransactionResult{}, oldErr
		}
		return result.TransactionResult{}, err
	}

	desc := ss.Description()

	if oldErr != nil && (!topo.SupportsSessions() || !description.SessionsSupported(desc.WireVersion)) {
		// Assuming we are retrying (oldErr != nil),
		// if server doesn't support retryable writes, return the original error
		// Conditions for retry write support are the same as that of sessions
		return result.TransactionResult{}, oldErr
	}

	conn, err := ss.Connection(ctx)
	if err != nil {
		if oldErr != nil {
			return result.TransactionResult{}, oldErr
		}
		return result.TransactionResult{}, err
	}
	defer conn.Close()

	return cmd.RoundTrip(ctx, desc, conn)
}
