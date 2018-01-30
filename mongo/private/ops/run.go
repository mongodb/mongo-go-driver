// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"context"

	"github.com/10gen/mongo-go-driver/mongo/model"
	"github.com/10gen/mongo-go-driver/mongo/private/conn"
	"github.com/10gen/mongo-go-driver/mongo/private/msg"
)

// Run executes an arbitrary command against the given database.
func Run(ctx context.Context, s *SelectedServer, db string, command interface{}, result interface{}) error {
	return runMayUseSecondary(ctx, s, db, command, result)
}

func runMustUsePrimary(ctx context.Context, s *SelectedServer, db string, command interface{}, result interface{}) error {

	request := msg.NewCommand(
		msg.NextRequestID(),
		db,
		s.ClusterKind == model.Single && s.Model().Kind != model.Mongos, // slaveOk
		command,
	)

	c, err := s.Connection(ctx)
	if err != nil {
		return err
	}

	errChan := make(chan error, 1)
	go func() {
		defer func() {
			// Ignore any error that occurs since we're in a different goroutine.
			_ = c.Close()
		}()
		errChan <- conn.ExecuteCommand(context.Background(), c, request, result)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err = <-errChan:
		return err
	}
}

func runMayUseSecondary(ctx context.Context, s *SelectedServer, db string, command interface{}, result interface{}) error {
	request := msg.NewCommand(
		msg.NextRequestID(),
		db,
		slaveOk(s),
		command,
	)

	c, err := s.Connection(ctx)
	if err != nil {
		return err
	}
	errChan := make(chan error, 1)
	go func() {
		defer func() {
			// Ignore any error that occurs since we're in a different goroutine.
			_ = c.Close()
		}()

		if rpMeta := readPrefMeta(s.ReadPref, c.Model().Kind); rpMeta != nil {
			err := msg.AddMeta(request, map[string]interface{}{
				"$readPreference": rpMeta,
			})
			if err != nil {
				errChan <- err
			}
		}

		errChan <- conn.ExecuteCommand(context.Background(), c, request, result)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err = <-errChan:
		return err
	}
}
