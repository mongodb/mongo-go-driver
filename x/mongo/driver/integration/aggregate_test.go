// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"bytes"
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/integtest"
	"go.mongodb.org/mongo-driver/v2/internal/serverselector"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/operation"
)

func TestAggregate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("Multiple Batches", func(t *testing.T) {
		ds := []bsoncore.Document{
			bsoncore.BuildDocument(nil, bsoncore.AppendInt32Element(nil, "_id", 1)),
			bsoncore.BuildDocument(nil, bsoncore.AppendInt32Element(nil, "_id", 2)),
			bsoncore.BuildDocument(nil, bsoncore.AppendInt32Element(nil, "_id", 3)),
			bsoncore.BuildDocument(nil, bsoncore.AppendInt32Element(nil, "_id", 4)),
			bsoncore.BuildDocument(nil, bsoncore.AppendInt32Element(nil, "_id", 5)),
		}
		wc := writeconcern.Majority()
		autoInsertDocs(t, wc, ds...)

		op := operation.NewAggregate(bsoncore.BuildArray(nil,
			bsoncore.BuildDocumentValue(
				bsoncore.BuildDocumentElement(nil,
					"$match", bsoncore.BuildDocumentElement(nil,
						"_id", bsoncore.AppendInt32Element(nil, "$gt", 2),
					),
				),
			),
			bsoncore.BuildDocumentValue(
				bsoncore.BuildDocumentElement(nil,
					"$sort", bsoncore.AppendInt32Element(nil, "_id", 1),
				),
			),
		)).Collection(integtest.ColName(t)).Database(dbName).Deployment(integtest.Topology(t)).
			ServerSelector(&serverselector.Write{}).BatchSize(2)
		err := op.Execute(context.Background())
		noerr(t, err)
		cursor, err := op.Result(driver.CursorOptions{BatchSize: 2})
		noerr(t, err)

		var got []bsoncore.Document
		for i := 0; i < 2; i++ {
			if !cursor.Next(context.Background()) {
				t.Error("Cursor should have results, but does not have a next result")
			}
			docs, err := cursor.Batch().Documents()
			noerr(t, err)
			got = append(got, docs...)
		}
		readers := ds[2:]
		for i, g := range got {
			if !bytes.Equal(g[:len(readers[i])], readers[i]) {
				t.Errorf("Did not get expected document. got %v; want %v", bson.Raw(g[:len(readers[i])]), readers[i])
			}
		}

		if cursor.Next(context.Background()) {
			t.Error("Cursor should be exhausted but has more results")
		}
	})
	t.Run("AllowDiskUse", func(t *testing.T) {
		ds := []bsoncore.Document{
			bsoncore.BuildDocument(nil, bsoncore.AppendInt32Element(nil, "_id", 1)),
			bsoncore.BuildDocument(nil, bsoncore.AppendInt32Element(nil, "_id", 2)),
		}
		wc := writeconcern.Majority()
		autoInsertDocs(t, wc, ds...)

		op := operation.NewAggregate(bsoncore.BuildArray(nil)).Collection(integtest.ColName(t)).Database(dbName).
			Deployment(integtest.Topology(t)).ServerSelector(&serverselector.Write{}).AllowDiskUse(true)
		err := op.Execute(context.Background())
		if err != nil {
			t.Errorf("Expected no error from allowing disk use, but got %v", err)
		}
	})
}
