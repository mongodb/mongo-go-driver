// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"time"

	"github.com/10gen/mongo-go-driver/bson"

	"github.com/10gen/mongo-go-driver/mongo"
	"github.com/10gen/mongo-go-driver/mongo/private/cluster"
	"github.com/10gen/mongo-go-driver/mongo/private/conn"
	"github.com/10gen/mongo-go-driver/mongo/private/msg"
	"github.com/10gen/mongo-go-driver/mongo/private/ops"
	"github.com/10gen/mongo-go-driver/mongo/readpref"
)

var concurrency = flag.Int("concurrency", 24, "how much concurrency should be used")
var ns = flag.String("namespace", "test.foo", "the namespace to use for test data")

func main() {

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	c, err := cluster.New()

	if err != nil {
		log.Fatalf("unable to create cluster: %s", err)
	}

	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sig
		cancel()
		close(done)
	}()

	log.Println("prepping")
	err = prep(ctx, c)
	if err != nil {
		log.Fatalf("unable to prep: %s", err)
	}
	log.Println("done prepping")

	log.Println("working")
	for i := 0; i < *concurrency; i++ {
		go work(ctx, i, c)
	}

	<-done
	log.Println("interupt received: shutting down")
	c.Close()
	log.Println("finished")
}

func prep(ctx context.Context, c *cluster.Cluster) error {

	var docs []bson.D
	for i := 0; i < 1000; i++ {
		docs = append(docs, bson.D{{"_id", i}})
	}

	ns := ops.ParseNamespace(*ns)
	deleteCommand := bson.D{
		{"delete", ns.Collection},
		{"deletes", []bson.D{
			bson.D{
				{"q", bson.M{}},
				{"limit", 0},
			},
		}},
	}
	deleteRequest := msg.NewCommand(
		msg.NextRequestID(),
		ns.DB,
		false,
		deleteCommand,
	)
	insertCommand := bson.D{
		{"insert", ns.Collection},
		{"documents", docs},
	}
	insertRequest := msg.NewCommand(
		msg.NextRequestID(),
		ns.DB,
		false,
		insertCommand,
	)

	s, err := c.SelectServer(ctx, cluster.WriteSelector(), readpref.Primary())
	if err != nil {
		return err
	}

	connection, err := s.Connection(ctx)
	if err != nil {
		return err
	}
	defer connection.Close()

	return conn.ExecuteCommands(ctx, connection, []msg.Request{deleteRequest, insertRequest}, []interface{}{&bson.D{}, &bson.D{}})
}

func work(ctx context.Context, idx int, c *cluster.Cluster) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ns := ops.ParseNamespace(*ns)
	rp := readpref.Nearest()
	for {
		select {
		case <-ctx.Done():
		default:

			limit := r.Intn(999) + 1

			s, err := c.SelectServer(ctx, readpref.Selector(rp), rp)
			if err != nil {
				log.Printf("%d-failed selecting a server: %s", idx, err)
				continue
			}

			pipeline := []bson.D{
				bson.D{{"$limit", limit}},
			}

			cursor, err := ops.Aggregate(ctx, &ops.SelectedServer{s, c.Model().Kind, rp}, ns, nil, pipeline, mongo.BatchSize(200))
			if err != nil {
				log.Printf("%d-failed executing aggregate: %s", idx, err)
				continue
			}

			count := 0
			var result bson.D
			for cursor.Next(ctx, &result) {
				count++
			}
			if cursor.Err() != nil {
				cursor.Close(ctx)
				log.Printf("%d-failed iterating aggregate results: %s", idx, cursor.Err())
				return
			}
			cursor.Close(ctx)

			log.Printf("%d-iterated %d docs", idx, count)
		}
	}
}
