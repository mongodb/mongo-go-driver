// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	oldbson "github.com/10gen/mongo-go-driver/bson"
)

// InsertOneResult is a result of an InsertOne operation.
// TODO GODRIVER-76: Document which types for interface{} are valid.
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
	DeletedCount int64 "n"
}

// UpdateResult is a result of an update operation.
// TODO GODRIVER-76: Document which types for interface{} are valid.
type UpdateResult struct {
	// The number of documents that matched the filter.
	MatchedCount int64
	// The number of documents that were modified.
	ModifiedCount int64
	// The identifier of the inserted document if an upsert took place.
	UpsertedID interface{}
}

type docID struct {
	ID interface{} "_id"
}

type updateServerResponse struct {
	N         int64
	NModified int64 "nModified"
	Upserted  []docID
}

// SetBSON is used by the BSON library to deserialize raw BSON into an UpdateOneResult.
func (result *UpdateResult) SetBSON(raw oldbson.Raw) error {
	var response updateServerResponse
	err := oldbson.Unmarshal(raw.Data, &response)
	if err != nil {
		return err
	}

	result.MatchedCount = response.N
	result.ModifiedCount = response.NModified

	if len(response.Upserted) > 0 {
		result.UpsertedID = response.Upserted[0].ID
	}

	return nil
}
