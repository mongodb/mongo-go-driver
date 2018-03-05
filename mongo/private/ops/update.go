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
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
)

// Update executes an update command with a given set of update documents and options.
func Update(ctx context.Context, s *SelectedServer, ns Namespace, updateDocs []*bson.Document,
	opt ...options.UpdateOptioner) (bson.Reader, error) {

	if err := ns.validate(); err != nil {
		return nil, err
	}

	command := bson.NewDocument(bson.EC.String("update", ns.Collection))

	arr := bson.NewArray()
	for _, doc := range updateDocs {
		arr.Append(bson.VC.Document(doc))
	}
	command.Append(bson.EC.Array("updates", arr))

	for _, option := range opt {
		switch option.(type) {
		case nil:
			continue
		case options.OptUpsert, options.OptCollation, options.OptArrayFilters:
			for _, doc := range updateDocs {
				option.Option(doc)
			}
		default:
			option.Option(command)
		}
	}

	rdr, err := runMustUsePrimary(ctx, s, ns.DB, command)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute update")
	}

	return rdr, nil
}
