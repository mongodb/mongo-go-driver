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

// DropIndexes drops one or all of the indexes in a collection.
func DropIndexes(ctx context.Context, s *SelectedServer, ns Namespace,
	index string) (bson.Reader, error) {

	listIndexesCommand := bson.NewDocument(
		bson.EC.String("dropIndexes", ns.Collection),
		bson.EC.String("index", index),
	)

	return runMustUsePrimary(ctx, s, ns.DB, listIndexesCommand)
}
