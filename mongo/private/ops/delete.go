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
	deleteDocs []*bson.Document, result interface{}, opts ...options.DeleteOption) error {

	if err := ns.validate(); err != nil {
		return err
	}

	command := bson.NewDocument(1 + uint(len(opts))).Append(bson.C.String("delete", ns.Collection))
	// command := oldbson.D{
	// 	{Name: "delete", Value: ns.Collection},
	// }

	arr := bson.NewArray(uint(len(deleteDocs)))
	for _, doc := range deleteDocs {
		arr.Append(bson.AC.Document(doc))
	}
	command.Append(bson.C.Array("deletes", arr))

	for _, option := range opts {
		switch option.(type) {
		case options.OptCollation:
			for _, doc := range deleteDocs {
				option.Option(doc)
			}
		default:
			option.Option(command)
		}
		// switch name := option.DeleteName(); name {
		// case "collation":
		// 	for i := range deleteDocs {
		// 		deleteDocs[i].AppendElem("collation", option.DeleteValue())
		// 	}
		// default:
		// 	command.AppendElem(option.DeleteName(), option.DeleteValue())
		// }
	}

	// command.AppendElem("deletes", deleteDocs)

	if writeConcern != nil {
		elem, err := writeConcern.MarshalBSONElement()
		if err != nil {
			return err
		}
		command.Append(elem)
		// command.AppendElem("writeConcern", writeConcern)
	}

	err := runMustUsePrimary(ctx, s, ns.DB, command, result)
	if err != nil {
		return internal.WrapError(err, "failed to execute delete")
	}

	return nil
}
