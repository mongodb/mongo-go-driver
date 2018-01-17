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

// Update executes an update command with a given set of update documents and options.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func Update(ctx context.Context, s *SelectedServer, ns Namespace, writeConcern *writeconcern.WriteConcern,
	updateDocs []*bson.Document, result interface{}, opt ...options.UpdateOption) error {

	if err := ns.validate(); err != nil {
		return err
	}

	command := bson.NewDocument(1 + uint(len(updateDocs))).Append(bson.C.String("update", ns.Collection))
	// command := oldbson.D{
	// 	{Name: "update", Value: ns.Collection},
	// }

	arr := bson.NewArray(uint(len(updateDocs)))
	for _, doc := range updateDocs {
		arr.Append(bson.AC.Document(doc))
	}
	command.Append(bson.C.Array("updates", arr))

	for _, option := range opt {
		switch option.(type) {
		case options.OptUpsert, options.OptCollation:
			for _, doc := range updateDocs {
				option.Option(doc)
			}
		default:
			option.Option(command)
		}
		// switch name := option.UpdateName(); name {
		// // upsert, multi, and collation are specified in each update documents
		// case "upsert":
		// 	fallthrough
		// case "multi":
		// 	fallthrough
		// case "collation":
		// 	for i, doc := range updateDocs {
		// 		doc.AppendElem(name, option.UpdateValue())
		// 		updateDocs[i] = doc
		// 	}
		//
		// // other options are specified in the top-level command document
		// default:
		// 	command.AppendElem(name, option.UpdateValue())
		// }
	}

	// command.AppendElem("updates", updateDocs)

	if writeConcern != nil {
		elem, err := writeConcern.MarshalBSONElement()
		if err != nil {
			return err
		}
		command.Append(elem)
		// command.AppendElem("writeConcern", writeConcern)
	}

	_, err := runMustUsePrimary(ctx, s, ns.DB, command, result)
	if err != nil {
		return internal.WrapError(err, "failed to execute update")
	}

	return nil
}
