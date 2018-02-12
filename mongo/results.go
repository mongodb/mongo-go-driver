// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"bytes"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
)

// InsertOneResult is a result of an InsertOne operation.
//
// InsertedID will be a Go type that corresponds to a BSON type.
type InsertOneResult struct {
	// The identifier that was inserted.
	InsertedID interface{}
}

// InsertManyResult is a result of an InsertMany operation.
type InsertManyResult struct {
	// Maps the indexes of inserted documents to their _id fields.
	InsertedIDs []interface{}
}

// DeleteResult is a result of an DeleteOne operation.
type DeleteResult struct {
	// The number of documents that were deleted.
	DeletedCount int64 `bson:"n"`
}

// UpdateResult is a result of an update operation.
//
// UpsertedID will be a Go type that corresponds to a BSON type.
type UpdateResult struct {
	// The number of documents that matched the filter.
	MatchedCount int64
	// The number of documents that were modified.
	ModifiedCount int64
	// The identifier of the inserted document if an upsert took place.
	UpsertedID interface{}
}

// UnmarshalBSON implements the bson.Unmarshaler interface.
func (result *UpdateResult) UnmarshalBSON(b []byte) error {
	itr, err := bson.Reader(b).Iterator()
	if err != nil {
		return err
	}

	for itr.Next() {
		elem := itr.Element()
		switch elem.Key() {
		case "n":
			switch elem.Value().Type() {
			case bson.TypeInt32:
				result.MatchedCount = int64(elem.Value().Int32())
			case bson.TypeInt64:
				result.MatchedCount = elem.Value().Int64()
			default:
				return fmt.Errorf("Received invalid type for n, should be Int32 or Int64, received %s", elem.Value().Type())
			}
		case "nModified":
			switch elem.Value().Type() {
			case bson.TypeInt32:
				result.ModifiedCount = int64(elem.Value().Int32())
			case bson.TypeInt64:
				result.ModifiedCount = elem.Value().Int64()
			default:
				return fmt.Errorf("Received invalid type for nModified, should be Int32 or Int64, received %s", elem.Value().Type())
			}
		case "upserted":
			switch elem.Value().Type() {
			case bson.TypeArray:
				e, err := elem.Value().ReaderArray().ElementAt(0)
				if err != nil {
					break
				}
				if e.Value().Type() != bson.TypeEmbeddedDocument {
					break
				}
				var d struct {
					ID interface{} `bson:"_id"`
				}
				err = bson.NewDecoder(bytes.NewReader(e.Value().ReaderDocument())).Decode(&d)
				if err != nil {
					return err
				}
				result.UpsertedID = d.ID
			default:
				return fmt.Errorf("Received invalid type for upserted, should be Array, received %s", elem.Value().Type())
			}
		}
	}

	return nil
}
