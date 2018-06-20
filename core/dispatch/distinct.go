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

// Distinct handles the full cycle dispatch and execution of a distinct command against the provided
// topology.
func Distinct(
	ctx context.Context,
	cmd command.Distinct,
	topo *topology.Topology,
	selector description.ServerSelector,
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

	return cmd.RoundTrip(ctx, desc, conn)
}
