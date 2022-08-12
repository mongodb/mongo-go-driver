// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"log"
	"time"

	"flag"

	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

var uri = flag.String("uri", "mongodb://localhost:27017", "the mongodb uri to use")
var col = flag.String("c", "test", "the collection name to use")

func main() {

	flag.Parse()

	if *uri == "" {
		log.Fatalf("uri flag must have a value")
	}

	cfg, err := topology.NewConfig(options.Client().ApplyURI(*uri))
	if err != nil {
		log.Fatal(err)
	}

	t, err := topology.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	err = t.Connect()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	op := operation.NewCommand(bsoncore.BuildDocument(nil, bsoncore.AppendStringElement(nil, "count", *col))).
		Deployment(t).ServerSelector(description.WriteSelector())
	err = op.Execute(ctx)
	if err != nil {
		log.Fatalf("failed executing count command: %v", err)
	}
	rdr := op.Result()
	log.Println(rdr)
}
