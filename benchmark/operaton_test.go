// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package benchmark

import (
	"context"
	"log"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

func BenchmarkNewInsert(b *testing.B) {
	cs, err := testutil.GetConnString()
	if err != nil {
		b.Error(err)
	}

	c, err := topology.New(topology.WithConnString(func(connstring.ConnString) connstring.ConnString {
		return cs
	}))
	if err != nil {
		b.Error(err)
	}

	err = c.Connect()
	if err != nil {
		b.Error(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var docs = make([]bsoncore.Document, 0, 10)
	for i := 0; i < 10; i++ {
		docs = append(docs, bsoncore.BuildDocument(nil, bsoncore.AppendInt32Element(nil, "_id", int32(i))))
	}

	b.Run("benchmark NewInsert", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				op := operation.NewInsert(docs...).Collection("foo").Database("test").Deployment(c).
					ServerSelector(description.WriteSelector())
				err = op.Execute(ctx)
				if err != nil {
					b.Error(err)
				}
			}
		})
	})
}

func BenchmarkNewCommand(b *testing.B) {
	cs, err := testutil.GetConnString()
	if err != nil {
		log.Fatal(err)
	}

	top, err := topology.New(topology.WithConnString(func(connstring.ConnString) connstring.ConnString { return cs }))
	if err != nil {
		log.Fatal(err)
	}
	err = top.Connect()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbname := cs.Database
	if dbname == "" {
		dbname = "test"
	}
	b.Run("benchmark NewCommand", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				op := operation.NewCommand(bsoncore.BuildDocument(nil, bsoncore.AppendStringElement(nil, "count", "test"))).
					Deployment(top).Database(dbname).ServerSelector(description.WriteSelector())
				err = op.Execute(ctx)
				if err != nil {
					b.Error(err)
				}
				op.Result()
			}
		})
	})
}
