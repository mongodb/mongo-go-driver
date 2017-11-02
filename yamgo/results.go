// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package yamgo

import (
	"github.com/10gen/mongo-go-driver/bson"
)

// InsertOneResult is a result of an InsertOne operation.
// TODO GODRIVER-76: Document which types for interface{} are valid.
type InsertOneResult struct {
	// The identifier that was inserted.
	InsertedID interface{}
}

// DeleteOneResult is a result of an DeleteOne operation.
type DeleteOneResult struct {
	// The number of documents that were deleted.
	DeletedCount int64 "n"
}

// UpdateOneResult is a result of an update operation.
// TODO GODRIVER-76: Document which types for interface{} are valid.
type UpdateOneResult struct {
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

type updateOneServerResponse struct {
	N         int64
	NModified int64 "nModified"
	Upserted  []docID
}

// SetBSON is used by the BSON library to deserialize raw BSON into an UpdateOneResult.
func (result *UpdateOneResult) SetBSON(raw bson.Raw) error {
	var response updateOneServerResponse
	err := bson.Unmarshal(raw.Data, &response)
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
