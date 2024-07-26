// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
)

// LegacyHelloLowercase is the lowercase, legacy version of the hello command.
var LegacyHelloLowercase = "ismaster"

const (
	singleAndMultiDataDir = "single_and_multi_document"
	tweetData             = "tweet.json"
	smallData             = "small_doc.json"
	largeData             = "large_doc.json"
)

// AddOptionsToURI appends connection string options to a URI.
func AddOptionsToURI(uri string, opts ...string) string {
	if !strings.ContainsRune(uri, '?') {
		if uri[len(uri)-1] != '/' {
			uri += "/"
		}

		uri += "?"
	} else {
		uri += "&"
	}

	for _, opt := range opts {
		uri += opt
	}

	return uri
}

// AddTLSConfigToURI checks for the environmental variable indicating that the tests are being run
// on an SSL-enabled server, and if so, returns a new URI with the necessary configuration.
func AddTLSConfigToURI(uri string) string {
	caFile := os.Getenv("MONGO_GO_DRIVER_CA_FILE")
	if len(caFile) == 0 {
		return uri
	}

	return AddOptionsToURI(uri, "ssl=true&sslCertificateAuthorityFile=", caFile)
}

func GetConnString() (*connstring.ConnString, error) {
	mongodbURI := os.Getenv("MONGODB_URI")
	if mongodbURI == "" {
		mongodbURI = "mongodb://localhost:27017"
	}

	mongodbURI = AddTLSConfigToURI(mongodbURI)

	cs, err := connstring.ParseAndValidate(mongodbURI)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func GetDBName(cs *connstring.ConnString) string {
	if cs.Database != "" {
		return cs.Database
	}

	return fmt.Sprintf("mongo-go-driver-%d", os.Getpid())
}

func getClientDB() (*mongo.Database, error) {
	cs, err := GetConnString()
	if err != nil {
		return nil, err
	}
	client, err := mongo.Connect(options.Client().ApplyURI(cs.String()))
	if err != nil {
		return nil, err
	}

	db := client.Database(GetDBName(cs))
	return db, nil
}

func SingleRunCommand(ctx context.Context, tm TimerManager, iters int) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	db, err := getClientDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Client().Disconnect(ctx) }()

	cmd := bson.D{{LegacyHelloLowercase, true}}

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

	db, err := getClientDB()
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

	return db.Drop(ctx)
}

func singleInsertCase(ctx context.Context, tm TimerManager, iters int, data string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	db, err := getClientDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Client().Disconnect(ctx) }()

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

	return db.Drop(ctx)
}

func SingleInsertSmallDocument(ctx context.Context, tm TimerManager, iters int) error {
	return singleInsertCase(ctx, tm, iters, smallData)
}

func SingleInsertLargeDocument(ctx context.Context, tm TimerManager, iters int) error {
	return singleInsertCase(ctx, tm, iters, largeData)
}
