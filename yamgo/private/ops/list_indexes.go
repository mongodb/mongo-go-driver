// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"context"

	"github.com/10gen/mongo-go-driver/yamgo/private/conn"
)

// ListIndexesOptions are the options for listing indexes.
type ListIndexesOptions struct {
	// The batch size for fetching results. A zero value indicates the server's default batch size.
	BatchSize int32
}

// ListIndexes lists the indexes on the given namespace.
func ListIndexes(ctx context.Context, s *SelectedServer, ns Namespace, options ListIndexesOptions) (Cursor, error) {

	listIndexesCommand := struct {
		ListIndexes string `bson:"listIndexes"`
	}{
		ListIndexes: ns.Collection,
	}
	var result cursorReturningResult
	err := runMustUsePrimary(ctx, s, ns.DB, listIndexesCommand, &result)
	switch err {
	case nil:
		return NewCursor(&result.Cursor, options.BatchSize, s)
	default:
		if conn.IsNsNotFound(err) {
			return NewExhaustedCursor()
		}
		return nil, err
	}

}
