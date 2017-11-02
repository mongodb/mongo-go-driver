// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package yamgo

import (
	"testing"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/stretchr/testify/require"
)

func TestDeleteOneResult_unmarshalInto(t *testing.T) {
	t.Parallel()

	doc := bson.D{
		bson.NewDocElem("n", 2),
		bson.NewDocElem("ok", 1),
	}

	bytes, err := bson.Marshal(doc)
	require.Nil(t, err)

	var result DeleteOneResult
	err = bson.Unmarshal(bytes, &result)
	require.Nil(t, err)
	require.Equal(t, result.DeletedCount, int64(2))
}

func TestDeleteOneResult_marshalFrom(t *testing.T) {
	t.Parallel()

	result := DeleteOneResult{DeletedCount: 1}
	bytes, err := bson.Marshal(result)
	require.Nil(t, err)

	var doc bson.M
	err = bson.Unmarshal(bytes, &doc)
	require.Nil(t, err)
	require.Equal(t, len(doc), 1)
	require.Equal(t, doc["n"], int64(1))
}

func TestUpdateServerResponse_unmarshalInto(t *testing.T) {
	t.Parallel()

	doc := bson.D{
		bson.NewDocElem("n", 1),
		bson.NewDocElem("nModified", 2),
		bson.NewDocElem("upserted", [...]interface{}{
			bson.D{
				bson.NewDocElem("index", 0),
				bson.NewDocElem("_id", 3),
			},
		}),
	}

	bytes, err := bson.Marshal(doc)
	require.Nil(t, err)

	var result updateOneServerResponse
	err = bson.Unmarshal(bytes, &result)
	require.Nil(t, err)
	require.Equal(t, result.N, int64(1))
	require.Equal(t, result.NModified, int64(2))
	require.Equal(t, len(result.Upserted), 1)
	require.Equal(t, result.Upserted[0].ID, 3)
}

func TestUpdateServerResponse_marshalFrom(t *testing.T) {
	t.Parallel()

	result := updateOneServerResponse{
		N:         1,
		NModified: 2,
		Upserted:  []docID{{ID: 3}},
	}
	bytes, err := bson.Marshal(result)
	require.Nil(t, err)

	var doc bson.D
	err = bson.Unmarshal(bytes, &doc)
	require.Nil(t, err)

	expected := bson.D{
		bson.NewDocElem("n", int64(1)),
		bson.NewDocElem("nModified", int64(2)),
		bson.NewDocElem("upserted", []interface{}{
			bson.D{bson.NewDocElem("_id", 3)},
		}),
	}

	require.Equal(t, doc, expected)
}

func TestUpdateOneResult_unmarshalInto(t *testing.T) {
	t.Parallel()

	doc := bson.D{
		bson.NewDocElem("n", 1),
		bson.NewDocElem("nModified", 2),
		bson.NewDocElem("upserted", [...]interface{}{
			bson.D{
				bson.NewDocElem("index", 0),
				bson.NewDocElem("_id", 3),
			},
		}),
	}

	bytes, err := bson.Marshal(doc)
	require.Nil(t, err)

	var result UpdateOneResult
	err = bson.Unmarshal(bytes, &result)
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(1))
	require.Equal(t, result.ModifiedCount, int64(2))
	require.Equal(t, result.UpsertedID, 3)
}
