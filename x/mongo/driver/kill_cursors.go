// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"context"
	"github.com/mongodb/mongo-go-driver/x/network/connection"
	"github.com/mongodb/mongo-go-driver/x/network/wiremessage"

	"github.com/mongodb/mongo-go-driver/x/network/command"
	"github.com/mongodb/mongo-go-driver/x/network/result"
)

// KillCursors handles the full cycle dispatch and execution of an aggregate command against the provided
// topology.
func KillCursors(
	ctx context.Context,
	ns command.Namespace,
	cursor *BatchCursor,
) (result.KillCursors, error) {
	desc := cursor.server.SelectedDescription()
	conn, err := cursor.server.Connection(ctx)
	if err != nil {
		return result.KillCursors{}, err
	}
	defer conn.Close()

	if desc.WireVersion.Max < 4 {
		return result.KillCursors{}, legacyKillCursors(ctx, cursor, ns, conn)
	}

	cmd := command.KillCursors{
		NS:  ns,
		IDs: []int64{cursor.ID()},
	}

	return cmd.RoundTrip(ctx, desc, conn)
}

func legacyKillCursors(ctx context.Context, cursor *BatchCursor, ns command.Namespace, conn connection.Connection) error {
	kc := wiremessage.KillCursors{
		NumberOfCursorIDs: 1,
		CursorIDs:         []int64{cursor.ID()},
		CollectionName:    ns.Collection,
		DatabaseName:      ns.DB,
	}

	return conn.WriteWireMessage(ctx, kc)
}
