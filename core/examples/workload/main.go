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

	"github.com/mongodb/mongo-go-driver/bson"

	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/dispatch"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/topology"
)

var concurrency = flag.Int("concurrency", 24, "how much concurrency should be used")
var ns = flag.String("namespace", "test.foo", "the namespace to use for test data")

func main() {

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	c, err := topology.New()

	if err != nil {
		log.Fatalf("unable to create topology: %s", err)
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
	log.Println("interrupt received: shutting down")
	_ = c.Disconnect(ctx)
	log.Println("finished")
}

func prep(ctx context.Context, c *topology.Topology) error {

	var docs = make([]*bson.Document, 0, 1000)
	for i := 0; i < 1000; i++ {
		docs = append(docs, bson.NewDocument(bson.EC.Int32("_id", int32(i))))
	}

	ns := command.ParseNamespace(*ns)

	s, err := c.SelectServer(ctx, description.WriteSelector())
	if err != nil {
		return err
	}

	conn, err := s.Connection(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	deletes := []*bson.Document{bson.NewDocument(
		bson.EC.SubDocument("q", bson.NewDocument()),
		bson.EC.Int32("limit", 0),
	)}
	_, err = (&command.Delete{WriteConcern: nil, NS: ns, Deletes: deletes}).RoundTrip(ctx, s.Description(), conn)
	if err != nil {
		return err
	}
	_, err = (&command.Insert{WriteConcern: nil, NS: ns, Docs: docs}).RoundTrip(ctx, s.Description(), conn)
	return err
}

func work(ctx context.Context, idx int, c *topology.Topology) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ns := command.ParseNamespace(*ns)
	rp := readpref.Nearest()
	for {
		select {
		case <-ctx.Done():
		default:

			limit := r.Intn(999) + 1

			pipeline := bson.NewArray(
				bson.VC.DocumentFromElements(
					bson.EC.Int32("$limit", int32(limit)),
				),
			)

			cmd := command.Aggregate{
				NS:       ns,
				Pipeline: pipeline,
				Opts:     []option.AggregateOptioner{option.OptBatchSize(200)},
				ReadPref: rp,
			}
			cursor, err := dispatch.Aggregate(ctx, cmd, c, description.ReadPrefSelector(rp), description.ReadPrefSelector(rp))
			if err != nil {
				log.Printf("%d-failed executing aggregate: %s", idx, err)
				continue
			}

			count := 0
			for cursor.Next(ctx) {
				count++
			}
			if cursor.Err() != nil {
				_ = cursor.Close(ctx)
				log.Printf("%d-failed iterating aggregate results: %s", idx, cursor.Err())
				return
			}
			_ = cursor.Close(ctx)

			log.Printf("%d-iterated %d docs", idx, count)
		}
	}
}
