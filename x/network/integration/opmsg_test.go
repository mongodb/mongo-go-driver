// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"testing"

	"bytes"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
	"go.mongodb.org/mongo-driver/x/network/command"
	"go.mongodb.org/mongo-driver/x/network/connection"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
)

func createServerConn(t *testing.T) (*topology.SelectedServer, connection.Connection) {
	server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
	noerr(t, err)
	conn, err := server.Connection(context.Background())
	noerr(t, err)

	return server, conn
}

func compareDocs(t *testing.T, reader bson.Raw, doc bsonx.Doc) {
	marshaled, err := doc.MarshalBSON()
	if err != nil {
		t.Errorf("error marshaling document: %s", err)
	}

	if !bytes.Equal(reader, marshaled) {
		t.Errorf("documents do not match")
	}
}

func createNamespace(t *testing.T) command.Namespace {
	return command.Namespace{
		DB:         dbName,
		Collection: testutil.ColName(t),
	}
}

func compareResults(t *testing.T, channelConn *internal.ChannelConn, docs ...bsonx.Doc) {
	if len(channelConn.Written) != 1 {
		t.Errorf("expected 1 messages to be sent but got %d", len(channelConn.Written))
	}

	writtenMsg := (<-channelConn.Written).(wiremessage.Msg)
	if len(writtenMsg.Sections) != 2 {
		t.Errorf("expected 2 sections in message. got %d", len(writtenMsg.Sections))
	}

	docSequence := writtenMsg.Sections[1].(wiremessage.SectionDocumentSequence).Documents
	if len(docSequence) != len(docs) {
		t.Errorf("expected %d documents. got %d", len(docs), len(docSequence))
	}

	for i, doc := range docs {
		compareDocs(t, docSequence[i], doc)
	}
}

func createChannelConn() *internal.ChannelConn {
	errChan := make(chan error, 1)
	errChan <- errors.New("read error")

	return &internal.ChannelConn{
		Written:  make(chan wiremessage.WireMessage, 1),
		ReadResp: nil,
		ReadErr:  errChan,
	}
}

func TestOpMsg(t *testing.T) {
	server, _ := createServerConn(t)
	desc := server.Description()

	if desc.WireVersion.Max < wiremessage.OpmsgWireVersion {
		t.Skip("skipping op_msg for wire version < 6")
	}

	t.Run("SingleDocInsert", func(t *testing.T) {
		ctx := context.TODO()
		server, conn := createServerConn(t)
		doc := bsonx.Doc{{"x", bsonx.String("testing single doc insert")}}

		cmd := &command.Insert{
			NS: command.Namespace{
				DB:         dbName,
				Collection: testutil.ColName(t),
			},
			Docs: []bsonx.Doc{doc},
		}

		res, err := cmd.RoundTrip(ctx, server.SelectedDescription(), conn)
		noerr(t, err)

		if len(res.WriteErrors) != 0 {
			t.Errorf("expected no write errors. got %d", len(res.WriteErrors))
		}
	})

	t.Run("SingleDocUpdate", func(t *testing.T) {
		ctx := context.TODO()
		server, conn := createServerConn(t)
		doc := bsonx.Doc{
			{"$set", bsonx.Document(bsonx.Doc{
				{"x", bsonx.String("updated x")},
			}),
			}}

		updateDocs := []bsonx.Doc{
			{
				{"q", bsonx.Document(bsonx.Doc{})},
				{"u", bsonx.Document(doc)},
				{"multi", bsonx.Boolean(true)},
			},
		}

		cmd := &command.Update{
			NS:   createNamespace(t),
			Docs: updateDocs,
		}

		res, err := cmd.RoundTrip(ctx, server.SelectedDescription(), conn)
		noerr(t, err)

		if len(res.WriteErrors) != 0 {
			t.Errorf("expected no write errors. got %d", len(res.WriteErrors))
		}
	})

	t.Run("SingleDocDelete", func(t *testing.T) {
		ctx := context.TODO()
		server, conn := createServerConn(t)
		doc := bsonx.Doc{{"x", bsonx.String("testing single doc insert")}}

		deleteDocs := []bsonx.Doc{
			{
				{"q", bsonx.Document(doc)},
				{"limit", bsonx.Int32(0)}},
		}
		cmd := &command.Delete{
			NS:      createNamespace(t),
			Deletes: deleteDocs,
		}

		res, err := cmd.RoundTrip(ctx, server.SelectedDescription(), conn)
		noerr(t, err)

		if len(res.WriteErrors) != 0 {
			t.Errorf("expected no write errors. got %d", len(res.WriteErrors))
		}
	})

	t.Run("MultiDocInsert", func(t *testing.T) {
		ctx := context.TODO()
		server, conn := createServerConn(t)

		doc1 := bsonx.Doc{{"x", bsonx.String("testing multi doc insert")}}
		doc2 := bsonx.Doc{{"y", bsonx.Int32(50)}}

		cmd := &command.Insert{
			NS: command.Namespace{
				DB:         dbName,
				Collection: testutil.ColName(t),
			},
			Docs: []bsonx.Doc{doc1, doc2},
		}

		channelConn := createChannelConn()

		_, err := cmd.RoundTrip(ctx, server.SelectedDescription(), channelConn)
		if err == nil {
			t.Errorf("expected read error. got nil")
		}

		compareResults(t, channelConn, doc1, doc2)

		// write to server
		res, err := cmd.RoundTrip(ctx, server.SelectedDescription(), conn)
		noerr(t, err)

		if len(res.WriteErrors) != 0 {
			t.Errorf("expected no write errors. got %d", len(res.WriteErrors))
		}
	})

	t.Run("MultiDocUpdate", func(t *testing.T) {
		ctx := context.TODO()
		server, conn := createServerConn(t)

		doc1 := bsonx.Doc{
			{"$set", bsonx.Document(bsonx.Doc{
				{"x", bsonx.String("updated x")},
			})},
		}

		doc2 := bsonx.Doc{
			{"$set", bsonx.Document(bsonx.Doc{
				{"y", bsonx.String("updated y")},
			})},
		}

		updateDocs := []bsonx.Doc{
			{
				{"q", bsonx.Document(bsonx.Doc{})},
				{"u", bsonx.Document(doc1)},
				{"multi", bsonx.Boolean(true)},
			},
			{
				{"q", bsonx.Document(bsonx.Doc{})},
				{"u", bsonx.Document(doc2)},
				{"multi", bsonx.Boolean(true)},
			},
		}

		cmd := &command.Update{
			NS:   createNamespace(t),
			Docs: updateDocs,
		}

		channelConn := createChannelConn()
		_, err := cmd.RoundTrip(ctx, server.SelectedDescription(), channelConn)
		if err == nil {
			t.Errorf("expected read error. got nil")
		}

		compareResults(t, channelConn, updateDocs...)

		// write to server
		res, err := cmd.RoundTrip(ctx, server.SelectedDescription(), conn)
		noerr(t, err)

		if len(res.WriteErrors) != 0 {
			t.Errorf("expected no write errors. got %d", len(res.WriteErrors))
		}
	})

	t.Run("MultiDocDelete", func(t *testing.T) {
		ctx := context.TODO()
		server, conn := createServerConn(t)

		doc1 := bsonx.Doc{{"x", bsonx.String("x")}}
		doc2 := bsonx.Doc{{"y", bsonx.String("y")}}

		deleteDocs := []bsonx.Doc{
			{
				{"q", bsonx.Document(doc1)},
				{"limit", bsonx.Int32(0)},
			},
			{
				{"q", bsonx.Document(doc2)},
				{"limit", bsonx.Int32(0)},
			},
		}

		cmd := &command.Delete{
			NS:      createNamespace(t),
			Deletes: deleteDocs,
		}

		channelConn := createChannelConn()
		_, err := cmd.RoundTrip(ctx, server.SelectedDescription(), channelConn)
		if err == nil {
			t.Errorf("expected read error. got nil")
		}

		compareResults(t, channelConn, deleteDocs...)

		// write to server
		res, err := cmd.RoundTrip(ctx, server.SelectedDescription(), conn)
		noerr(t, err)

		if len(res.WriteErrors) != 0 {
			t.Errorf("expected no write errors. got %d", len(res.WriteErrors))
		}
	})
}
