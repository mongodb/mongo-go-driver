// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"context"
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/internal"
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
)

// Count counts how many documents in a collection match a given query.
func Count(ctx context.Context, s *SelectedServer, ns Namespace, query *bson.Document,
	opts ...options.CountOptioner) (int64, error) {

	if err := ns.validate(); err != nil {
		return 0, err
	}

	command := bson.NewDocument()
	command.Append(bson.EC.String("count", ns.Collection), bson.EC.SubDocument("query", query))

	for _, option := range opts {
		if option == nil {
			continue
		}
		option.Option(command)
	}

	rdr, err := runMayUseSecondary(ctx, s, ns.DB, command)
	if err != nil {
		return 0, internal.WrapError(err, "failed to execute count")
	}

	val, err := rdr.Lookup("n")
	switch {
	case err == bson.ErrElementNotFound:
		return 0, errors.New("Invalid response from server, no n field")
	case err != nil:
		return 0, err
	}

	switch val.Value().Type() {
	case bson.TypeDouble:
		return int64(val.Value().Double()), nil
	case bson.TypeInt32:
		return int64(val.Value().Int32()), nil
	case bson.TypeInt64:
		return int64(val.Value().Int64()), nil
	default:
		return 0, errors.New("Invalid response from server, value field is not a number")
	}
}
