// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"context"

	"github.com/10gen/mongo-go-driver/mongo/internal"
	"github.com/10gen/mongo-go-driver/mongo/options"
	"github.com/10gen/mongo-go-driver/mongo/readconcern"
	"github.com/skriptble/wilson/bson"
)

// Find executes a query.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func Find(ctx context.Context, s *SelectedServer, ns Namespace, readConcern *readconcern.ReadConcern,
	filter *bson.Document, findOptions ...options.FindOption) (Cursor, error) {

	if err := ns.validate(); err != nil {
		return nil, err
	}

	command := bson.NewDocument(1 + uint(len(findOptions)))
	command.Append(bson.C.String("find", ns.Collection))
	// command := oldbson.D{
	// 	{Name: "find", Value: ns.Collection},
	// }

	if filter != nil {
		command.Append(bson.C.SubDocument("filter", filter))
		// command.AppendElem("filter", filter)
	}

	var limit int64
	var batchSize int32
	// limit := int64(0)
	// batchSize := int32(0)

	for _, option := range findOptions {
		switch t := option.(type) {
		case options.OptLimit:
			limit = int64(t)
			option.Option(command)
		case options.OptBatchSize:
			batchSize = int32(t)
			option.Option(command)
		default:
			option.Option(command)
		}
		// switch name := option.FindName(); name {
		// // upsert, multi, and collation are specified in each update documents
		// case "cursorType":
		// 	value := option.FindValue()
		// 	if value == options.Tailable {
		// 		command.AppendElem("tailable", true)
		// 	}
		//
		// 	if value == options.TailableAwait {
		// 		command.AppendElem("tailable", true)
		// 		command.AppendElem("awaitData", true)
		// 	}
		//
		// case "maxTimeMS":
		// 	command.AppendElem(
		// 		name,
		// 		int64(option.FindValue().(time.Duration)/time.Millisecond),
		// 	)
		//
		// // other options are specified in the top-level command document
		// default:
		// 	if name == "limit" {
		// 		limit = int64(option.FindValue().(options.OptLimit))
		// 	}
		//
		// 	if name == "batchSize" {
		// 		batchSize = int32(option.FindValue().(options.OptBatchSize))
		// 	}
		//
		// 	command.AppendElem(name, option.FindValue())
		// }
	}

	if limit != 0 && batchSize != 0 && limit <= int64(batchSize) {
		command.Append(bson.C.Boolean("singleBatch", true))
		// command.AppendElem("singleBatch", true)
	}

	if readConcern != nil {
		elem, err := readConcern.MarshalBSONElement()
		if err != nil {
			return nil, err
		}
		command.Append(elem)
		// command.AppendElem("readConcern", readConcern)
	}

	var result cursorReturningResult

	err := runMayUseSecondary(ctx, s, ns.DB, command, &result)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute update")
	}

	return NewCursor(&result.Cursor, batchSize, s)
}
