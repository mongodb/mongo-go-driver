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

// FindOneAndUpdate modifies and returns a single document.
func FindOneAndUpdate(ctx context.Context, s *SelectedServer, ns Namespace, query *bson.Document,
	update *bson.Document, opts ...options.FindOneAndUpdateOptioner) (Cursor, error) {

	if err := ns.validate(); err != nil {
		return nil, err
	}

	command := bson.NewDocument()
	command.Append(
		bson.EC.String("findAndModify", ns.Collection),
		bson.EC.SubDocument("query", query),
		bson.EC.SubDocument("update", update),
	)

	for _, option := range opts {
		if option == nil {
			continue
		}
		option.Option(command)
	}

	rdr, err := runMustUsePrimary(ctx, s, ns.DB, command)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute find_one_and_update")
	}

	val, err := rdr.Lookup("value")
	switch {
	case err == bson.ErrElementNotFound:
		return nil, errors.New("invalid response from server, no value field")
	case err != nil:
		return nil, err
	}

	switch val.Value().Type() {
	case bson.TypeNull:
		return &singleResultCursor{}, nil
	case bson.TypeEmbeddedDocument:
		return &singleResultCursor{rdr: val.Value().ReaderDocument()}, nil
	default:
		return nil, errors.New("invalid response from server, value field is not a document")
	}
}
