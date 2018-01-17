// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"bytes"
	"context"

	"github.com/10gen/mongo-go-driver/mongo/internal"
	"github.com/10gen/mongo-go-driver/mongo/options"
	"github.com/10gen/mongo-go-driver/mongo/readconcern"
	"github.com/skriptble/wilson/bson"
)

// Distinct returns the distinct values for a specified field across a single collection.
func Distinct(ctx context.Context, s *SelectedServer, ns Namespace, readConcern *readconcern.ReadConcern,
	field string, query *bson.Document, opts ...options.DistinctOptioner) ([]interface{}, error) {

	if err := ns.validate(); err != nil {
		return nil, err
	}

	command := bson.NewDocument()
	command.Append(bson.C.String("distinct", ns.Collection), bson.C.String("key", field))

	if query != nil {
		command.Append(bson.C.SubDocument("query", query))
	}

	for _, option := range opts {
		if option == nil {
			continue
		}
		option.Option(command)
	}

	if readConcern != nil {
		elem, err := readConcern.MarshalBSONElement()
		if err != nil {
			return nil, err
		}
		command.Append(elem)
	}

	result := struct{ Values []interface{} }{}

	rdr, err := runMayUseSecondary(ctx, s, ns.DB, command)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute count")
	}

	err = bson.NewDecoder(bytes.NewReader(rdr)).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result.Values, nil
}
