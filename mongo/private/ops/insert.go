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
	"github.com/10gen/mongo-go-driver/mongo/writeconcern"
)

// Insert executes an insert command for the given set of  documents.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func Insert(ctx context.Context, s *SelectedServer, ns Namespace, writeConcern *writeconcern.WriteConcern,
	docs []interface{}, result interface{}, options ...options.InsertOption) error {

	if err := ns.validate(); err != nil {
		return err
	}

	command := bson.D{
		{Name: "insert", Value: ns.Collection},
		{Name: "documents", Value: docs},
	}

	for _, option := range options {
		command.AppendElem(option.InsertName(), option.InsertValue())
	}

	if writeConcern != nil {
		command.AppendElem("writeConcern", writeConcern)
	}

	err := runMustUsePrimary(ctx, s, ns.DB, command, result)
	if err != nil {
		return internal.WrapError(err, "failed to execute insert")
	}

	return nil
}
