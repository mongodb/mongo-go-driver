package integration

import (
	"context"
	"testing"

	"bytes"
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/connection"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"github.com/mongodb/mongo-go-driver/internal"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
)

func createServerConn(t *testing.T) (*topology.SelectedServer, connection.Connection) {
	server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
	noerr(t, err)
	conn, err := server.Connection(context.Background())
	noerr(t, err)

	return server, conn
}

func compareDocs(t *testing.T, reader bson.Reader, doc *bson.Document) {
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

func compareResults(t *testing.T, channelConn *internal.ChannelConn, docs ...*bson.Document) {
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
		doc := bson.NewDocument(bson.EC.String("x", "testing single doc insert"))

		cmd := &command.Insert{
			NS: command.Namespace{
				DB:         dbName,
				Collection: testutil.ColName(t),
			},
			Docs: []*bson.Document{doc},
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
		doc := bson.NewDocument(
			bson.EC.SubDocument("$set", bson.NewDocument(
				bson.EC.String("x", "updated x"),
			),
			))

		updateDocs := []*bson.Document{
			bson.NewDocument(
				bson.EC.SubDocument("q", bson.NewDocument()),
				bson.EC.SubDocument("u", doc),
				bson.EC.Boolean("multi", true),
			),
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
		doc := bson.NewDocument(bson.EC.String("x", "testing single doc insert"))

		deleteDocs := []*bson.Document{
			bson.NewDocument(
				bson.EC.SubDocument("q", doc),
				bson.EC.Int32("limit", 0)),
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

		doc1 := bson.NewDocument(bson.EC.String("x", "testing multi doc insert"))
		doc2 := bson.NewDocument(bson.EC.Int32("y", 50))

		cmd := &command.Insert{
			NS: command.Namespace{
				DB:         dbName,
				Collection: testutil.ColName(t),
			},
			Docs: []*bson.Document{doc1, doc2},
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

		doc1 := bson.NewDocument(
			bson.EC.SubDocument("$set", bson.NewDocument(
				bson.EC.String("x", "updated x"),
			)),
		)

		doc2 := bson.NewDocument(
			bson.EC.SubDocument("$set", bson.NewDocument(
				bson.EC.String("y", "updated y"),
			)),
		)

		updateDocs := []*bson.Document{
			bson.NewDocument(
				bson.EC.SubDocument("q", bson.NewDocument()),
				bson.EC.SubDocument("u", doc1),
				bson.EC.Boolean("multi", true),
			),
			bson.NewDocument(
				bson.EC.SubDocument("q", bson.NewDocument()),
				bson.EC.SubDocument("u", doc2),
				bson.EC.Boolean("multi", true),
			),
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

		doc1 := bson.NewDocument(
			bson.EC.String("x", "x"),
		)
		doc2 := bson.NewDocument(
			bson.EC.String("y", "y"),
		)

		deleteDocs := []*bson.Document{
			bson.NewDocument(
				bson.EC.SubDocument("q", doc1),
				bson.EC.Int32("limit", 0),
			),
			bson.NewDocument(
				bson.EC.SubDocument("q", doc2),
				bson.EC.Int32("limit", 0),
			),
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
