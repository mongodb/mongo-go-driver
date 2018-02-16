// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"context"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
)

// ListCollectionsOptions are the options for listing collections.
type ListCollectionsOptions struct {
	// A query filter for the collections
	Filter interface{}
	// The batch size for fetching results. A zero value indicates the server's default batch size.
	BatchSize int32
	// The maximum execution time in milliseconds. A zero value indicates no maximum.
	MaxTime time.Duration
}

// ListCollections lists the collections in the given database with the given options.
func ListCollections(ctx context.Context, s *SelectedServer, db string, options ListCollectionsOptions) (Cursor, error) {
	if err := validateDB(db); err != nil {
		return nil, err
	}

	listCollectionsCommand := bson.NewDocument(
		bson.EC.Int32("listCollections", 1),
		bson.EC.Int64("maxTimeMS", int64(options.MaxTime/time.Millisecond)),
		bson.EC.SubDocumentFromElements("cursor", bson.EC.Int32("batchSize", options.BatchSize)))

	if options.Filter != nil {
		doc, err := bson.NewDocumentEncoder().EncodeDocument(options.Filter)
		if err != nil {
			return nil, err
		}
		listCollectionsCommand.Append(bson.EC.SubDocument("filter", doc))
	}

	rdr, err := runMustUsePrimary(ctx, s, db, listCollectionsCommand)
	if err != nil {
		return nil, err
	}

	return NewCursor(rdr, options.BatchSize, s)
}
