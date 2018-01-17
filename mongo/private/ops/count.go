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

// Count counts how many documents in a collection match a given query.
func Count(ctx context.Context, s *SelectedServer, ns Namespace, readConcern *readconcern.ReadConcern,
	query *bson.Document, opts ...options.CountOption) (int, error) {

	if err := ns.validate(); err != nil {
		return 0, err
	}

	command := bson.NewDocument()
	command.Append(bson.C.String("count", ns.Collection), bson.C.SubDocument("query", query))
	// command := oldbson.D{
	// 	{Name: "count", Value: ns.Collection},
	// 	{Name: "query", Value: query},
	// }

	for _, option := range opts {
		option.Option(command)
		// switch name := option.CountName(); name {
		// case "maxTimeMS":
		// 	command.AppendElem(
		// 		name,
		// 		int64(option.CountValue().(time.Duration)/time.Millisecond),
		// 	)
		// default:
		// 	command.AppendElem(name, option.CountValue())
		//
		// }
	}

	if readConcern != nil {
		elem, err := readConcern.MarshalBSONElement()
		if err != nil {
			return 0, err
		}
		command.Append(elem)
		// command.AppendElem("readConcern", readConcern)
	}

	result := struct{ N int }{}

	_, err := runMayUseSecondary(ctx, s, ns.DB, command, &result)
	if err != nil {
		return 0, internal.WrapError(err, "failed to execute count")
	}

	return result.N, nil
}
