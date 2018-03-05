// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"bytes"
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/internal"
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
)

// Distinct returns the distinct values for a specified field across a single collection.
func Distinct(ctx context.Context, s *SelectedServer, ns Namespace, field string,
	query *bson.Document, opts ...options.DistinctOptioner) ([]interface{}, error) {

	if err := ns.validate(); err != nil {
		return nil, err
	}

	command := bson.NewDocument()
	command.Append(bson.EC.String("distinct", ns.Collection), bson.EC.String("key", field))

	if query != nil {
		command.Append(bson.EC.SubDocument("query", query))
	}

	for _, option := range opts {
		if option == nil {
			continue
		}
		option.Option(command)
	}

	result := struct{ Values []interface{} }{}

	rdr, err := runMayUseSecondary(ctx, s, ns.DB, command)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute distinct")
	}

	err = bson.NewDecoder(bytes.NewReader(rdr)).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result.Values, nil
}
