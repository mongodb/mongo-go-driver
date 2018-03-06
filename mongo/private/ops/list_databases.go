// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
)

// ListDatabasesOptions are the options for listing databases.
type ListDatabasesOptions struct {
	// Specifies whether or not the name is the only value that should be returned.
	NameOnly bool
	// The maximum execution time in milliseconds.  A zero value indicates no maximum.
	MaxTime time.Duration
}

// ListDatabases lists the databases with the given options
func ListDatabases(ctx context.Context, s *SelectedServer, filter *bson.Document,
	options ListDatabasesOptions) (Cursor, error) {

	listDatabasesCommand := bson.NewDocument(
		bson.EC.Int32("listDatabases", 1))

	if filter != nil {
		listDatabasesCommand.Append(
			bson.EC.SubDocument("filter", filter))
	}

	if options.NameOnly {
		listDatabasesCommand.Append(
			bson.EC.Boolean("nameOnly", options.NameOnly))
	}

	if options.MaxTime != 0 {
		listDatabasesCommand.Append(bson.EC.Int64("maxTimeMS", int64(options.MaxTime/time.Millisecond)))
	}

	rdr, err := runMustUsePrimary(ctx, s, "admin", listDatabasesCommand)
	if err != nil {
		return nil, err
	}

	dbs, err := rdr.Lookup("databases")
	if err != nil {
		return nil, err
	}
	if dbs.Value().Type() != bson.TypeArray {
		return nil, fmt.Errorf("returned databases element of wrong type. Should be Array but is %s", dbs.Value().Type())
	}

	return &listDatabasesCursor{
		databases: dbs.Value().MutableArray(),
		current:   -1,
	}, nil
}

type listDatabasesCursor struct {
	databases *bson.Array
	current   int
	err       error
}

func (cursor *listDatabasesCursor) ID() int64 {
	return -1
}

func (cursor *listDatabasesCursor) Next(_ context.Context) bool {
	cursor.current++
	if cursor.current < cursor.databases.Len() {
		return true
	}
	return false
}

func (cursor *listDatabasesCursor) Decode(v interface{}) error {
	br, err := cursor.DecodeBytes()
	if err != nil {
		return err
	}

	dec := bson.NewDecoder(bytes.NewReader(br))
	err = dec.Decode(v)
	return err
}

func (cursor *listDatabasesCursor) DecodeBytes() (bson.Reader, error) {
	br, err := cursor.databases.Lookup(uint(cursor.current))
	if err != nil {
		return nil, err
	}
	if br.Type() != bson.TypeEmbeddedDocument {
		return nil, errors.New("Non-Document in batch of documents for cursor")
	}
	return br.ReaderDocument(), nil
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
