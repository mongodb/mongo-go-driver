// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"bytes"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/stretchr/testify/require"
)

func TestDeleteResult_unmarshalInto(t *testing.T) {
	t.Parallel()

	doc := bson.NewDocument(
		bson.EC.Int64("n", 2),
		bson.EC.Int64("ok", 1))

	b, err := doc.MarshalBSON()
	require.Nil(t, err)

	var result DeleteResult
	err = bson.NewDecoder(bytes.NewReader(b)).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, result.DeletedCount, int64(2))
}

func TestDeleteResult_marshalFrom(t *testing.T) {
	t.Parallel()

	result := DeleteResult{DeletedCount: 1}
	var buf bytes.Buffer
	err := bson.NewEncoder(&buf).Encode(result)
	require.Nil(t, err)

	doc, err := bson.ReadDocument(buf.Bytes())
	require.Nil(t, err)

	require.Equal(t, doc.Len(), 1)
	e, err := doc.LookupErr("n")
	require.NoError(t, err)
	require.Equal(t, e.Type(), bson.TypeInt64)
	require.Equal(t, e.Int64(), int64(1))
}

func TestUpdateOneResult_unmarshalInto(t *testing.T) {
	t.Parallel()

	doc := bson.NewDocument(
		bson.EC.Int32("n", 1),
		bson.EC.Int32("nModified", 2),
		bson.EC.ArrayFromElements(
			"upserted",
			bson.VC.DocumentFromElements(
				bson.EC.Int32("index", 0),
				bson.EC.Int32("_id", 3),
			),
		))

	b, err := doc.MarshalBSON()
	require.Nil(t, err)

	var result UpdateResult
	err = bson.NewDecoder(bytes.NewReader(b)).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(1))
	require.Equal(t, result.ModifiedCount, int64(2))
	require.Equal(t, int(result.UpsertedID.(int32)), 3)
}
