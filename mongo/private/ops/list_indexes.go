// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/conn"
)

// ListIndexesOptions are the options for listing indexes.
type ListIndexesOptions struct {
	// The batch size for fetching results. A zero value indicates the server's default batch size.
	BatchSize int32
}

// ListIndexes lists the indexes on the given namespace.
func ListIndexes(ctx context.Context, s *SelectedServer, ns Namespace, options ListIndexesOptions) (Cursor, error) {

	listIndexesCommand := bson.NewDocument(bson.EC.String("listIndexes", ns.Collection))

	rdr, err := runMustUsePrimary(ctx, s, ns.DB, listIndexesCommand)
	switch err {
	case nil:
		return NewCursor(rdr, options.BatchSize, s)
	default:
		if conn.IsNsNotFound(err) {
			return NewExhaustedCursor()
		}
		return nil, err
	}

}
