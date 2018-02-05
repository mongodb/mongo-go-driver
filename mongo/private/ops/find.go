// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"context"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/mongo/internal"
	"github.com/10gen/mongo-go-driver/mongo/options"
	"github.com/10gen/mongo-go-driver/mongo/readconcern"
)

// Find executes a query.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func Find(ctx context.Context, s *SelectedServer, ns Namespace, readConcern *readconcern.ReadConcern,
	filter *bson.Document, findOptions ...options.FindOptioner) (Cursor, error) {

	if err := ns.validate(); err != nil {
		return nil, err
	}

	command := bson.NewDocument()
	command.Append(bson.C.String("find", ns.Collection))

	if filter != nil {
		command.Append(bson.C.SubDocument("filter", filter))
	}

	var limit int64
	var batchSize int32

	for _, option := range findOptions {
		switch t := option.(type) {
		case nil:
			continue
		case options.OptLimit:
			limit = int64(t)
			option.Option(command)
		case options.OptBatchSize:
			batchSize = int32(t)
			option.Option(command)
		default:
			option.Option(command)
		}
	}

	if limit != 0 && batchSize != 0 && limit <= int64(batchSize) {
		command.Append(bson.C.Boolean("singleBatch", true))
	}

	if readConcern != nil {
		elem, err := readConcern.MarshalBSONElement()
		if err != nil {
			return nil, err
		}
		command.Append(elem)
	}

	rdr, err := runMayUseSecondary(ctx, s, ns.DB, command)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute update")
	}

	return NewCursor(rdr, batchSize, s)
}
