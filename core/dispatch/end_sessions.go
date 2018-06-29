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

// EndSessions handles the full cycle dispatch and execution of an endSessions command against the provided
// topology.
func EndSessions(
	ctx context.Context,
	cmd command.EndSessions,
	topo *topology.Topology,
	selector description.ServerSelector,
) ([]result.EndSessions, []error) {

	var allowed []description.Server
	for _, s := range topo.Description().Servers {
		if s.Kind != description.Unknown {
			allowed = append(allowed, s)
		}
	}

	// Manually select server as topo.SelectServer is blocking and could cause the application to hang until timeout
	// May cause small race condition in which server selected may become unavailable during selection
	suitable, err := selector.SelectServer(topo.Description(), allowed)
	if err != nil {
		return nil, []error{err}
	}
	var ss *topology.SelectedServer
	for _, server := range suitable {
		ss, err := topo.FindServer(server)
		if err != nil {
			return nil, []error{err}
		}

		if ss != nil {
			break
		}
	}

	if ss == nil {
		return nil, []error{topology.ErrServerSelectionTimeout} // failed to select server
	}

	desc := ss.Description()
	conn, err := ss.Connection(ctx)
	if err != nil {
		return nil, []error{err}
	}
	defer conn.Close()

	return cmd.RoundTrip(ctx, desc, conn)
}
