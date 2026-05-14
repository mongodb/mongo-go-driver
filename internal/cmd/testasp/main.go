// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// testasp is a small smoke test for the streamprocessing client. It connects
// to an Atlas Stream Processing workspace, creates a processor reading from
// the sample_stream_solar source, samples some documents, fetches stats, and
// stops the processor. The processor is always dropped on exit.
//
// Usage:
//
//	go run ./internal/cmd/testasp \
//	    -uri  mongodb://atlas-stream-<id>-<suffix>.<region>.a.query.mongodb.net/ \
//	    -username <user> \
//	    -password <pass>
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/uuid"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/streamprocessing"
)

func main() {
	uri := flag.String("uri", "", "workspace endpoint (mongodb://atlas-stream-...)")
	username := flag.String("username", "", "username for SCRAM auth")
	password := flag.String("password", "", "password for SCRAM auth")
	sampleLimit := flag.Int("sample-limit", 20, "max documents to sample on the initial call")
	settleSeconds := flag.Int("settle", 3, "seconds to wait after Start before sampling")
	samplePolls := flag.Int("sample-polls", 10, "max getMore polls before giving up on samples")
	samplePollInterval := flag.Duration("sample-poll-interval", 2*time.Second, "wait between getMore polls")
	sampleTarget := flag.Int("sample-target", 5, "stop polling once this many documents have been collected")
	verbose := flag.Bool("verbose", false, "include per-operator detail in stats")
	sinkConn := flag.String("sink-connection", "inny", "Atlas cluster connection name to use as the $merge sink")
	sinkDB := flag.String("sink-db", "testasp", "database name passed to $merge.into.db")
	sinkColl := flag.String("sink-collection", "solar_output", "collection name passed to $merge.into.coll")
	apm := flag.Bool("apm", false, "print wire command/reply payloads for ASP commands")
	flag.Parse()

	if *uri == "" || *username == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "all of -uri, -username, -password are required")
		flag.Usage()
		os.Exit(2)
	}

	// Top-level context honors Ctrl-C so cleanup still runs.
	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	clientOpts := options.Client().
		ApplyURI(*uri).
		SetAuth(options.Credential{Username: *username, Password: *password})

	if *apm {
		clientOpts = clientOpts.SetMonitor(&event.CommandMonitor{
			Started: func(_ context.Context, e *event.CommandStartedEvent) {
				if !isASPCommand(e.CommandName) {
					return
				}
				log.Printf("APM > %s: %s", e.CommandName, oneline(bson.Raw(e.Command)))
			},
			Succeeded: func(_ context.Context, e *event.CommandSucceededEvent) {
				if !isASPCommand(e.CommandName) {
					return
				}
				log.Printf("APM < %s: %s", e.CommandName, oneline(bson.Raw(e.Reply)))
			},
			Failed: func(_ context.Context, e *event.CommandFailedEvent) {
				if !isASPCommand(e.CommandName) {
					return
				}
				log.Printf("APM x %s: %v", e.CommandName, e.Failure)
			},
		})
	}

	client, err := streamprocessing.Connect(clientOpts)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer func() {
		discCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.Disconnect(discCtx); err != nil {
			log.Printf("disconnect: %v", err)
		}
	}()

	sps := client.StreamProcessors()
	id, err := uuid.New()
	if err != nil {
		log.Fatalf("uuid: %v", err)
	}
	name := fmt.Sprintf("testasp-%s", id)
	log.Printf("processor name: %s", name)

	// Always drop on exit. Use background context so cleanup still runs after
	// rootCtx has been cancelled.
	defer func() {
		dropCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := sps.Get(name).Drop(dropCtx); err != nil {
			log.Printf("drop (cleanup): %v", err)
		} else {
			log.Printf("dropped processor %s", name)
		}
	}()

	pipeline := []bson.D{
		{{Key: "$source", Value: bson.D{{Key: "connectionName", Value: "sample_stream_solar"}}}},
		{{Key: "$merge", Value: bson.D{
			{Key: "into", Value: bson.D{
				{Key: "connectionName", Value: *sinkConn},
				{Key: "db", Value: *sinkDB},
				{Key: "coll", Value: *sinkColl},
			}},
		}}},
	}
	log.Printf("pipeline sink: connection=%s db=%s coll=%s", *sinkConn, *sinkDB, *sinkColl)

	createCtx, cancel := context.WithTimeout(rootCtx, 30*time.Second)
	defer cancel()
	if err := sps.Create(createCtx, name, pipeline); err != nil {
		log.Fatalf("create: %v", err)
	}
	log.Printf("created")

	sp := sps.Get(name)
	startCtx, cancel := context.WithTimeout(rootCtx, 30*time.Second)
	defer cancel()
	if err := sp.Start(startCtx, nil); err != nil {
		log.Fatalf("start: %v", err)
	}
	log.Printf("started; settling for %ds", *settleSeconds)
	time.Sleep(time.Duration(*settleSeconds) * time.Second)

	// Exercise getStreamProcessor and log the parsed info so we notice if any
	// spec-described fields are missing from the server response.
	infoCtx, cancel := context.WithTimeout(rootCtx, 30*time.Second)
	defer cancel()
	info, err := sps.GetInfo(infoCtx, name)
	if err != nil {
		log.Printf("getInfo: %v", err)
	} else {
		var lastChange string
		if info.LastStateChange != nil {
			lastChange = info.LastStateChange.Format(time.RFC3339)
		}
		log.Printf("info: name=%q state=%q pipelineStages=%d lastStateChange=%q errorMsg=%q",
			info.Name, info.State, len(info.Pipeline), lastChange, info.ErrorMsg)
	}

	// Open the sample cursor. The ASP sample cursor is a tailable tap: it
	// carries documents that arrive while it is polled, so an initial call
	// often returns empty and we need to poll repeatedly.
	sampleCtx, cancel := context.WithTimeout(rootCtx, 30*time.Second)
	defer cancel()
	res, err := sp.GetStreamProcessorSamples(sampleCtx,
		options.GetStreamProcessorSamples().SetLimit(int32(*sampleLimit)))
	if err != nil {
		log.Fatalf("sample (initial): %v", err)
	}
	collected := append([]bson.Raw(nil), res.Documents...)
	log.Printf("initial batch: %d document(s); cursor=%d", len(res.Documents), res.CursorID)
	for i, d := range res.Documents {
		log.Printf("  [%d] %s", i, oneline(d))
	}

	cursorID := res.CursorID
	for poll := 1; poll <= *samplePolls && cursorID != 0 && len(collected) < *sampleTarget; poll++ {
		select {
		case <-rootCtx.Done():
			log.Printf("interrupted; stopping sample polling")
			cursorID = 0
		case <-time.After(*samplePollInterval):
		}
		if cursorID == 0 {
			break
		}

		pollCtx, cancel := context.WithTimeout(rootCtx, 30*time.Second)
		more, err := sp.GetStreamProcessorSamples(pollCtx,
			options.GetStreamProcessorSamples().
				SetCursorID(cursorID).
				SetBatchSize(5))
		cancel()
		if err != nil {
			log.Printf("sample (poll %d): %v", poll, err)
			break
		}
		cursorID = more.CursorID
		collected = append(collected, more.Documents...)
		log.Printf("poll %d: %d new document(s); total=%d; cursor=%d",
			poll, len(more.Documents), len(collected), cursorID)
		for i, d := range more.Documents {
			log.Printf("  [%d] %s", len(collected)-len(more.Documents)+i, oneline(d))
		}
	}
	log.Printf("sampling complete: %d document(s) collected", len(collected))

	// Stats.
	statsCtx, cancel := context.WithTimeout(rootCtx, 30*time.Second)
	defer cancel()
	var statsOpts []options.Lister[options.GetStreamProcessorStatsOptions]
	if *verbose {
		statsOpts = append(statsOpts, options.GetStreamProcessorStats().SetVerbose(true))
	}
	stats, err := sp.Stats(statsCtx, statsOpts...)
	if err != nil {
		log.Printf("stats: %v", err)
	} else {
		log.Printf("stats: %s", oneline(stats))
	}

	// Stop.
	stopCtx, cancel := context.WithTimeout(rootCtx, 30*time.Second)
	defer cancel()
	if err := sp.Stop(stopCtx); err != nil {
		log.Fatalf("stop: %v", err)
	}
	log.Printf("stopped")
}

// oneline renders a BSON document as compact extended JSON.
func oneline(raw bson.Raw) string {
	s, err := bson.MarshalExtJSON(raw, false, false)
	if err != nil {
		return fmt.Sprintf("<unmarshalable: %v>", err)
	}
	return string(s)
}

// isASPCommand reports whether the command name belongs to the Atlas Stream
// Processing wire protocol.
func isASPCommand(name string) bool {
	switch name {
	case "createStreamProcessor",
		"startStreamProcessor",
		"stopStreamProcessor",
		"dropStreamProcessor",
		"getStreamProcessor",
		"getStreamProcessorStats",
		"startSampleStreamProcessor",
		"getMoreSampleStreamProcessor":
		return true
	}
	return false
}
