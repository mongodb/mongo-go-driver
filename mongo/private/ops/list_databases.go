// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"context"
	"time"

	oldbson "github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/mongo/internal"
)

// ListDatabasesOptions are the options for listing databases.
type ListDatabasesOptions struct {
	// The maximum execution time in milliseconds.  A zero value indicates no maximum.
	MaxTime time.Duration
}

// ListDatabases lists the databases with the given options
func ListDatabases(ctx context.Context, s *SelectedServer, options ListDatabasesOptions) (Cursor, error) {

	listDatabasesCommand := struct {
		ListDatabases int32 `bson:"listDatabases"`
		MaxTimeMS     int64 `bson:"maxTimeMS,omitempty"`
	}{
		ListDatabases: 1,
		MaxTimeMS:     int64(options.MaxTime / time.Millisecond),
	}

	var result struct {
		Databases []oldbson.Raw `bson:"databases"`
	}
	err := runMustUsePrimary(ctx, s, "admin", listDatabasesCommand, &result)
	if err != nil {
		return nil, err
	}

	return &listDatabasesCursor{
		databases: result.Databases,
		current:   0,
	}, nil
}

type listDatabasesCursor struct {
	databases []oldbson.Raw
	current   int
	err       error
}

func (cursor *listDatabasesCursor) Next(_ context.Context, result interface{}) bool {
	if cursor.current < len(cursor.databases) {
		err := oldbson.Unmarshal(cursor.databases[cursor.current].Data, result)
		if err != nil {
			cursor.err = internal.WrapError(err, "unable to parse listDatabases result")
			return false
		}

		cursor.current++
		return true
	}
	return false
}

// Err returns the error status of the cursor.
func (cursor *listDatabasesCursor) Err() error {
	return cursor.err
}

// Close closes the cursor. Ordinarily this is a no-op as the server
// closes the cursor when it is exhausted. Returns the error status
// of this cursor so that clients do not have to call Err() separately.
func (cursor *listDatabasesCursor) Close(_ context.Context) error {
	return nil
}
