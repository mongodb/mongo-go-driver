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
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// FindOneAndUpdate handles the full cycle dispatch and execution of a FindOneAndUpdate command against the provided
// topology.
func FindOneAndUpdate(
	ctx context.Context,
	cmd command.FindOneAndUpdate,
	topo *topology.Topology,
	selector description.ServerSelector,
	wc *writeconcern.WriteConcern,
) (result.FindAndModify, error) {

	ss, err := topo.SelectServer(ctx, selector)
	if err != nil {
		return result.FindAndModify{}, err
	}

	if wc != nil {
		opt, err := writeConcernOption(wc)
		if err != nil {
			return result.FindAndModify{}, err
		}
		cmd.Opts = append(cmd.Opts, opt)
	}

	// NOTE: We iterate through the options because the user may have provided
	// an option explicitly and that needs to override the provided write concern.
	// We put this here because it would complicate the methods that call this to
	// parse out the option.
	acknowledged := true
	for _, opt := range cmd.Opts {
		wc, ok := opt.(option.OptWriteConcern)
		if !ok {
			continue
		}
		acknowledged = wc.Acknowledged
		break
	}

	desc := ss.Description()
	conn, err := ss.Connection(ctx)
	if err != nil {
		return result.FindAndModify{}, err
	}

	if !acknowledged {
		go func() {
			defer func() {
				_ = recover()
			}()
			defer conn.Close()
			_, _ = cmd.RoundTrip(ctx, desc, conn)
		}()
		return result.FindAndModify{}, ErrUnacknowledgedWrite
	}
	defer conn.Close()

	return cmd.RoundTrip(ctx, desc, conn)
}
