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
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/command"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/dispatch"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
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

	t, err := topology.New(topology.WithConnString(func(connstring.ConnString) connstring.ConnString { return cs }))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbname := cs.Database
	if dbname == "" {
		dbname = "test"
	}

	cmd := command.Command{DB: dbname, Command: bson.NewDocument(bson.EC.String("count", *col))}
	rdr, err := dispatch.Command(ctx, cmd, t, description.WriteSelector())
	if err != nil {
		log.Fatalf("failed executing count command on %s.%s: %v", dbname, *col, err)
	}

	result, err := extjson.BsonToExtJSON(true, rdr)
	if err != nil {
		log.Fatalf("failed to convert BSON to extended JSON: %s", err)
	}
	log.Println(result)
}
