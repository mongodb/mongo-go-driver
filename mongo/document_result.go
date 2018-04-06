// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
)

// ErrNoDocuments is returned by Decode when an operation that returns a
// DocumentResult doesn't return any documents.
var ErrNoDocuments = errors.New("mongo: no documents in result")

// DocumentResult represents a single document returned from an operation. If
// the operation returned an error, the Err method of DocumentResult will
// return that error.
type DocumentResult struct {
	err error
	cur Cursor
	rdr bson.Reader
}

// Decode will attempt to decode the first document into v. If there was an
// error from the operation that created this DocumentResult then the error
// will be returned. If there were no returned documents, ErrNoDocuments is
// returned.
func (dr *DocumentResult) Decode(v interface{}) error {
	switch {
	case dr.err != nil:
		return dr.err
	case dr.rdr != nil:
		if v == nil {
			return nil
		}
		return bson.Unmarshal(dr.rdr, v)
	case dr.cur != nil:
		if !dr.cur.Next(context.TODO()) {
			if err := dr.cur.Err(); err != nil {
				return err
			}
			return ErrNoDocuments
		}
		if v == nil {
			return nil
		}
		return dr.cur.Decode(v)
	}

	return ErrNoDocuments
}
