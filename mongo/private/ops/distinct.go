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

// Distinct returns the distinct values for a specified field across a single collection.
func Distinct(ctx context.Context, s *SelectedServer, ns Namespace, readConcern *readconcern.ReadConcern,
	field string, query *bson.Document, opts ...options.DistinctOption) ([]interface{}, error) {

	if err := ns.validate(); err != nil {
		return nil, err
	}

	command := bson.NewDocument()
	command.Append(bson.C.String("distinct", ns.Collection), bson.C.String("key", field))
	// command := oldbson.D{
	// 	{Name: "distinct", Value: ns.Collection},
	// 	{Name: "key", Value: field},
	// }

	if query != nil {
		command.Append(bson.C.SubDocument("query", query))
		// command.AppendElem("query", query)
	}

	for _, option := range opts {
		option.Option(command)
		// switch name := option.DistinctName(); name {
		// case "maxTimeMS":
		// 	command.AppendElem(
		// 		name,
		// 		int64(option.DistinctValue().(time.Duration)/time.Millisecond),
		// 	)
		// default:
		// 	command.AppendElem(name, option.DistinctValue())
		//
		// }
	}

	if readConcern != nil {
		elem, err := readConcern.MarshalBSONElement()
		if err != nil {
			return nil, err
		}
		command.Append(elem)
		// command.AppendElem("readConcern", readConcern)
	}

	result := struct{ Values []interface{} }{}

	_, err := runMayUseSecondary(ctx, s, ns.DB, command, &result)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute count")
	}

	return result.Values, nil
}
