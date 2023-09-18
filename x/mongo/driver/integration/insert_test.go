// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

func TestInsert(t *testing.T) {
	t.Skip()
	topo, err := topology.New(nil)
	if err != nil {
		t.Fatalf("Couldn't connect topology: %v", err)
	}
	_ = topo.Connect()

	doc := bsoncore.BuildDocument(nil, bsoncore.AppendDoubleElement(nil, "pi", 3.14159))

	iop := operation.NewInsert(doc).Database("foo").Collection("bar").Deployment(topo)
	err = iop.Execute(context.Background())
	if err != nil {
		t.Fatalf("Couldn't execute insert operation: %v", err)
	}
	t.Log(iop.Result())

	fop := operation.NewFind(bsoncore.BuildDocument(nil, bsoncore.AppendDoubleElement(nil, "pi", 3.14159))).
		Database("foo").Collection("bar").Deployment(topo).BatchSize(1)
	err = fop.Execute(context.Background())
	if err != nil {
		t.Fatalf("Couldn't execute find operation: %v", err)
	}
	cur, err := fop.Result(driver.CursorOptions{BatchSize: 2})
	if err != nil {
		t.Fatalf("Couldn't get cursor result from find operation: %v", err)
	}
	for cur.Next(context.Background()) {
		batch := cur.Batch()
		docs, err := batch.Documents()
		if err != nil {
			t.Fatalf("Couldn't iterate batch: %v", err)
		}
		for i, doc := range docs {
			t.Log(i, doc)
		}
	}
}
