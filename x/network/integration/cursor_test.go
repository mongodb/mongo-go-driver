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
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver/session"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver/uuid"
	"github.com/mongodb/mongo-go-driver/x/network/command"
	"github.com/mongodb/mongo-go-driver/x/network/description"
	"github.com/stretchr/testify/assert"
)

func TestTailableCursorLoopsUntilDocsAvailable(t *testing.T) {
	server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
	noerr(t, err)

	// create capped collection
	createCmd := bsonx.Doc{
		{"create", bsonx.String(testutil.ColName(t))},
		{"capped", bsonx.Boolean(true)},
		{"size", bsonx.Int32(1000)}}
	_, err = testutil.RunCommand(t, server.Server, dbName, createCmd)

	// Insert a document
	d := bsonx.Doc{{"_id", bsonx.Int32(1)}, {"ts", bsonx.Timestamp(5, 0)}}
	wc := writeconcern.New(writeconcern.WMajority())
	testutil.AutoInsertDocs(t, wc, d)

	rdr, err := d.MarshalBSON()
	noerr(t, err)

	clientID, err := uuid.New()
	noerr(t, err)

	cursor, err := driver.Find(
		context.Background(),
		command.Find{
			NS:     command.Namespace{DB: dbName, Collection: testutil.ColName(t)},
			Filter: bsonx.Doc{{"ts", bsonx.Document(bsonx.Doc{{"$gte", bsonx.Timestamp(5, 0)}})}},
		},
		testutil.Topology(t),
		description.WriteSelector(),
		clientID,
		&session.Pool{},
		bson.DefaultRegistry,
		options.Find().SetCursorType(options.TailableAwait),
	)
	noerr(t, err)

	// assert that there is a document returned
	assert.True(t, cursor.Next(context.Background()), "Cursor should have a next result")

	// make sure it's the right document
	var next bson.Raw
	next = cursor.Batch(next)

	if !bytes.Equal(next[:len(rdr)], rdr) {
		t.Errorf("Did not get expected document. got %v; want %v", bson.Raw(next[:len(rdr)]), bson.Raw(rdr))
	}

	// insert another document in 500 MS
	d = bsonx.Doc{{"_id", bsonx.Int32(2)}, {"ts", bsonx.Timestamp(6, 0)}}

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
	next = cursor.Batch(next[:0])

	if !bytes.Equal(next[:len(rdr)], rdr) {
		t.Errorf("Did not get expected document. got %v; want %v", bson.Raw(next[:len(rdr)]), bson.Raw(rdr))
	}
}
