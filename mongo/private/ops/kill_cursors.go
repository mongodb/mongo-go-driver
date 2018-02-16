// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/internal"
)

// KillCursors kills one or more cursors.
func KillCursors(ctx context.Context, s *SelectedServer, ns Namespace,
	cursors []int64) (bson.Reader, error) {

	if err := ns.validate(); err != nil {
		return nil, err
	}

	cursorsArr := bson.NewArray()
	for _, cursor := range cursors {
		cursorsArr.Append(bson.VC.Int64(cursor))
	}

	command := bson.NewDocument()
	command.Append(
		bson.EC.String("killCursors", ns.Collection),
		bson.EC.Array("cursors", cursorsArr),
	)

	rdr, err := runMayUseSecondary(ctx, s, ns.DB, command)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute killCursors")
	}

	return rdr, nil
}
