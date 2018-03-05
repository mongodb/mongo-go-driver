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

// Delete executes an delete command with a given set of delete documents and options.
func Delete(ctx context.Context, s *SelectedServer, ns Namespace, deleteDocs []*bson.Document,
	opts ...options.DeleteOptioner) (bson.Reader, error) {

	if err := ns.validate(); err != nil {
		return nil, err
	}

	command := bson.NewDocument(bson.EC.String("delete", ns.Collection))

	arr := bson.NewArray()
	for _, doc := range deleteDocs {
		arr.Append(bson.VC.Document(doc))
	}
	command.Append(bson.EC.Array("deletes", arr))

	for _, option := range opts {
		switch option.(type) {
		case nil:
		case options.OptCollation:
			for _, doc := range deleteDocs {
				option.Option(doc)
			}
		default:
			option.Option(command)
		}
	}

	rdr, err := runMustUsePrimary(ctx, s, ns.DB, command)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute delete")
	}

	return rdr, nil
}
