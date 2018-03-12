// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
)

// CreateIndexes creates one or more index in a collection.
func CreateIndexes(ctx context.Context, s *SelectedServer, ns Namespace,
	indexes *bson.Array) error {

	listIndexesCommand := bson.NewDocument(
		bson.EC.String("createIndexes", ns.Collection),
		bson.EC.Array("indexes", indexes),
	)

	_, err := runMustUsePrimary(ctx, s, ns.DB, listIndexesCommand)
	return err
}
