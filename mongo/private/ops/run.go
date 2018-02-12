// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/model"
	"github.com/mongodb/mongo-go-driver/mongo/private/conn"
	"github.com/mongodb/mongo-go-driver/mongo/private/msg"
)

// Run executes an arbitrary command against the given database.
func Run(ctx context.Context, s *SelectedServer, db string, command interface{}) (bson.Reader, error) {
	return runMayUseSecondary(ctx, s, db, command)
}

func runMustUsePrimary(ctx context.Context, s *SelectedServer, db string, command interface{}) (bson.Reader, error) {

	request := msg.NewCommand(
		msg.NextRequestID(),
		db,
		s.ClusterKind == model.Single && s.Model().Kind != model.Mongos, // slaveOk
		command,
	)

	c, err := s.Connection(ctx)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	return conn.ExecuteCommand(ctx, c, request)
}

func runMayUseSecondary(ctx context.Context, s *SelectedServer, db string, command interface{}) (bson.Reader, error) {
	request := msg.NewCommand(
		msg.NextRequestID(),
		db,
		slaveOk(s),
		command,
	)

	c, err := s.Connection(ctx)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	if rpMeta := readPrefMeta(s.ReadPref, c.Model().Kind); rpMeta != nil {
		err := msg.AddMeta(request, map[string]*bson.Document{
			"$readPreference": rpMeta,
		})

		if err != nil {
			return nil, err
		}
	}

	return conn.ExecuteCommand(ctx, c, request)
}
