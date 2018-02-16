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

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/extjson"
	"github.com/mongodb/mongo-go-driver/mongo/connstring"
	"github.com/mongodb/mongo-go-driver/mongo/private/cluster"
	"github.com/mongodb/mongo-go-driver/mongo/private/ops"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
)

var uri = flag.String("uri", "mongodb://localhost:27017", "the mongodb uri to use")
var col = flag.String("c", "test", "the collection name to use")

func main() {

	flag.Parse()

	if *uri == "" {
		log.Fatalf("uri flag must have a value")
	}

	cs, err := connstring.Parse(*uri)
	if err != nil {
		log.Fatal(err)
	}

	c, err := cluster.New(
		cluster.WithConnString(cs),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	s, err := c.SelectServer(timeoutCtx, cluster.WriteSelector(), readpref.Primary())
	if err != nil {
		log.Fatalf("%v: %v", err, c.Model().Servers[0].LastError)
	}

	dbname := cs.Database
	if dbname == "" {
		dbname = "test"
	}

	rdr, err := ops.Run(
		ctx,
		&ops.SelectedServer{
			Server:   s,
			ReadPref: readpref.Primary(),
		},
		dbname,
		bson.NewDocument(bson.EC.String("count", *col)),
	)
	if err != nil {
		log.Fatalf("failed executing count command on %s.%s: %v", dbname, *col, err)
	}

	result, err := extjson.BsonToExtJSON(true, rdr)
	if err != nil {
		log.Fatalf("failed to convert BSON to extended JSON: %s", err)
	}
	log.Println(result)
}
