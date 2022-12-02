// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package benchmark

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	singleAndMultiDataDir = "single_and_multi_document"
	tweetData             = "tweet.json"
	smallData             = "small_doc.json"
	largeData             = "large_doc.json"
)

func getClientDB(ctx context.Context) (*mongo.Database, error) {
	cs, err := testutil.GetConnString()
	if err != nil {
		return nil, err
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(cs.String()))
	if err != nil {
		return nil, err
	}
	if err = client.Connect(ctx); err != nil {
		return nil, err
	}

	db := client.Database(testutil.GetDBName(cs))
	return db, nil
}

func SingleRunCommand(ctx context.Context, tm TimerManager, iters int) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	db, err := getClientDB(ctx)
	if err != nil {
		return err
	}
	defer db.Client().Disconnect(ctx)

	cmd := bson.D{{internal.LegacyHelloLowercase, true}}

	tm.ResetTimer()
	for i := 0; i < iters; i++ {
		var doc bson.D
		err := db.RunCommand(ctx, cmd).Decode(&doc)
		if err != nil {
			return err
		}
		// read the document and then throw it away to prevent
		out, err := bson.Marshal(doc)
		if err != nil {
			return err
		}
		if len(out) == 0 {
			return errors.New("output of command is empty")
		}
	}
	tm.StopTimer()

	return nil
}

func SingleFindOneByID(ctx context.Context, tm TimerManager, iters int) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	db, err := getClientDB(ctx)
	if err != nil {
		return err
	}

	db = db.Client().Database("perftest")
	if err = db.Drop(ctx); err != nil {
		return err
	}

	doc, err := loadSourceDocument(getProjectRoot(), perfDataDir, singleAndMultiDataDir, tweetData)
	if err != nil {
		return err
	}
	coll := db.Collection("corpus")

	for i := 0; i < iters; i++ {
		idDoc := make(bson.D, 0, len(doc)+1)
		idDoc = append(idDoc, bson.E{"_id", i})
		idDoc = append(idDoc, doc...)
		res, err := coll.InsertOne(ctx, idDoc)
		if err != nil {
			return err
		}
		if res.InsertedID == nil {
			return errors.New("no inserted ID returned")
		}
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		var res bson.D
		err := coll.FindOne(ctx, bson.D{{"_id", i}}).Decode(&res)
		if err != nil {
			return err
		}
	}

	tm.StopTimer()

	if err = db.Drop(ctx); err != nil {
		return err
	}

	return nil
}

func singleInsertCase(ctx context.Context, tm TimerManager, iters int, data string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	db, err := getClientDB(ctx)
	if err != nil {
		return err
	}
	defer db.Client().Disconnect(ctx)

	db = db.Client().Database("perftest")
	if err = db.Drop(ctx); err != nil {
		return err
	}

	doc, err := loadSourceDocument(getProjectRoot(), perfDataDir, singleAndMultiDataDir, data)
	if err != nil {
		return err
	}

	err = db.RunCommand(ctx, bson.D{{"create", "corpus"}}).Err()
	if err != nil {
		return err
	}

	coll := db.Collection("corpus")

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		if _, err = coll.InsertOne(ctx, doc); err != nil {
			return err
		}
	}

	tm.StopTimer()

	if err = db.Drop(ctx); err != nil {
		return err
	}

	return nil
}

func SingleInsertSmallDocument(ctx context.Context, tm TimerManager, iters int) error {
	return singleInsertCase(ctx, tm, iters, smallData)
}

func SingleInsertLargeDocument(ctx context.Context, tm TimerManager, iters int) error {
	return singleInsertCase(ctx, tm, iters, largeData)
}
