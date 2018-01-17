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
	"github.com/10gen/mongo-go-driver/mongo/writeconcern"
	"github.com/skriptble/wilson/bson"
)

// Delete executes an delete command with a given set of delete documents and options.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func Delete(ctx context.Context, s *SelectedServer, ns Namespace, writeConcern *writeconcern.WriteConcern,
	deleteDocs []*bson.Document, opts ...options.DeleteOptioner) (bson.Reader, error) {

	if err := ns.validate(); err != nil {
		return nil, err
	}

	command := bson.NewDocument(bson.C.String("delete", ns.Collection))

	arr := bson.NewArray()
	for _, doc := range deleteDocs {
		arr.Append(bson.AC.Document(doc))
	}
	command.Append(bson.C.Array("deletes", arr))

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

	if writeConcern != nil {
		elem, err := writeConcern.MarshalBSONElement()
		if err != nil {
			return nil, err
		}
		command.Append(elem)
	}

	rdr, err := runMustUsePrimary(ctx, s, ns.DB, command)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute delete")
	}

	return rdr, nil
}
