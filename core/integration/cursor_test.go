// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestTailableCursorLoopsUntilDocsAvailable(t *testing.T) {
	server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
	noerr(t, err)

	// create capped collection
	createCmd := bson.NewDocument(
		bson.EC.String("create", testutil.ColName(t)),
		bson.EC.Boolean("capped", true),
		bson.EC.Int32("size", 1000))
	_, err = testutil.RunCommand(t, server.Server, dbName, createCmd)

	conn, err := server.Connection(context.Background())
	noerr(t, err)

	// Insert a document
	d := bson.NewDocument(bson.EC.Int32("_id", 1), bson.EC.Timestamp("ts", 5, 0))
	wc := writeconcern.New(writeconcern.WMajority())
	testutil.AutoInsertDocs(t, wc, d)

	rdr, err := d.MarshalBSON()
	noerr(t, err)

	// find that document, setting cursor type to TAILABLEAWAIT
	cursor, err := (&command.Find{
		NS:     command.Namespace{DB: dbName, Collection: testutil.ColName(t)},
		Filter: bson.NewDocument(bson.EC.SubDocument("ts", bson.NewDocument(bson.EC.Timestamp("$gte", 5, 0)))),
		Opts: []option.FindOptioner{
			option.OptLimit(0),
			option.OptBatchSize(1),
			option.OptCursorType(option.TailableAwait)},
	}).RoundTrip(context.Background(), server.SelectedDescription(), server, conn)
	noerr(t, err)

	// assert that there is a document returned
	assert.True(t, cursor.Next(context.Background()), "Cursor should have a next result")

	// make sure it's the right document
	var next bson.Reader
	err = cursor.Decode(&next)
	noerr(t, err)

	if !bytes.Equal(next[:len(rdr)], rdr) {
		t.Errorf("Did not get expected document. got %v; want %v", bson.Reader(next[:len(rdr)]), bson.Reader(rdr))
	}

	// insert another document in 500 MS
	d = bson.NewDocument(bson.EC.Int32("_id", 2), bson.EC.Timestamp("ts", 6, 0))

	rdr, err = d.MarshalBSON()
	noerr(t, err)

	go func() {
		time.Sleep(time.Millisecond * 500)
		testutil.AutoInsertDocs(t, wc, d)
	}()

	// context with timeout so test fails if loop does not work as expected
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	// assert that there is another document returned
	// cursor.Next should loop calling getMore until a document becomes available (in 500 ms)
	assert.True(t, cursor.Next(ctx), "Cursor should have a next result")

	noerr(t, cursor.Err())

	// make sure it's the right document the second time
	err = cursor.Decode(&next)
	noerr(t, err)

	if !bytes.Equal(next[:len(rdr)], rdr) {
		t.Errorf("Did not get expected document. got %v; want %v", bson.Reader(next[:len(rdr)]), bson.Reader(rdr))
	}
}
