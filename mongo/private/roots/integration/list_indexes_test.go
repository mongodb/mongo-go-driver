package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/addr"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/command"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
)

func TestCommandListIndexes(t *testing.T) {
	noerr := func(t *testing.T, err error) {
		t.Helper()
		if err != nil {
			t.Errorf("Unepexted error: %v", err)
			t.FailNow()
		}
	}
	dropcollection := func(t *testing.T, server *topology.Server) {
		conn, err := server.Connection(context.Background())
		noerr(t, err)
		_, err = (&command.Command{DB: dbName, Command: bson.NewDocument(bson.EC.String("drop", t.Name()))}).RoundTrip(context.Background(), server.SelectedDescription(), conn)
		if err != nil && !command.IsNotFound(err) {
			noerr(t, err)
		}
	}
	addindexes := func(t *testing.T, server *topology.Server, keys []string) {
		conn, err := server.Connection(context.Background())
		noerr(t, err)
		indexes := bson.NewDocument()
		for _, k := range keys {
			indexes.Append(bson.EC.Int32(k, 1))
		}
		name := strings.Join(keys, "_")
		indexes = bson.NewDocument(
			bson.EC.SubDocument("key", indexes),
			bson.EC.String("name", name),
		)

		cmd := bson.NewDocument(
			bson.EC.String("createIndexes", t.Name()),
			bson.EC.ArrayFromElements("indexes", bson.VC.Document(indexes)),
		)

		_, err = (&command.Command{DB: dbName, Command: cmd}).RoundTrip(context.Background(), server.SelectedDescription(), conn)
		noerr(t, err)
	}
	t.Run("InvalidDatabaseName", func(t *testing.T) {
		server, err := topology.NewServer(addr.Addr(*host))
		noerr(t, err)
		conn, err := server.Connection(context.Background())
		noerr(t, err)
		ns := command.Namespace{DB: "ex", Collection: "space"}
		cursor, err := (&command.ListIndexes{NS: ns}).RoundTrip(context.Background(), server.SelectedDescription(), server, conn)
		noerr(t, err)

		indexes := []string{}
		var next *bson.Document

		for cursor.Next(context.Background()) {
			err = cursor.Decode(next)
			noerr(t, err)

			elem, err := next.Lookup("name")
			noerr(t, err)
			if elem.Value().Type() != bson.TypeString {
				t.Errorf("Incorrect type for 'name'. got %v; want %v", elem.Value().Type(), bson.TypeString)
				t.FailNow()
			}
			indexes = append(indexes, elem.Value().StringValue())
		}

		if len(indexes) != 0 {
			t.Errorf("Expected no indexes from invalid database. got %d; want %d", len(indexes), 0)
		}
	})
	t.Run("InvalidCollectionName", func(t *testing.T) {
		server, err := topology.NewServer(addr.Addr(*host))
		noerr(t, err)
		conn, err := server.Connection(context.Background())
		noerr(t, err)
		ns := command.Namespace{DB: "ex", Collection: t.Name()}
		cursor, err := (&command.ListIndexes{NS: ns}).RoundTrip(context.Background(), server.SelectedDescription(), server, conn)
		noerr(t, err)

		indexes := []string{}
		var next *bson.Document

		for cursor.Next(context.Background()) {
			err = cursor.Decode(next)
			noerr(t, err)

			elem, err := next.Lookup("name")
			noerr(t, err)
			if elem.Value().Type() != bson.TypeString {
				t.Errorf("Incorrect type for 'name'. got %v; want %v", elem.Value().Type(), bson.TypeString)
				t.FailNow()
			}
			indexes = append(indexes, elem.Value().StringValue())
		}

		if len(indexes) != 0 {
			t.Errorf("Expected no indexes from invalid database. got %d; want %d", len(indexes), 0)
		}
	})
	t.Run("SingleBatch", func(t *testing.T) {
		server, err := topology.NewServer(addr.Addr(*host))
		noerr(t, err)
		conn, err := server.Connection(context.Background())
		noerr(t, err)
		dropcollection(t, server)
		addindexes(t, server, []string{"a"})
		addindexes(t, server, []string{"b"})
		addindexes(t, server, []string{"c"})
		addindexes(t, server, []string{"d", "e"})

		ns := command.NewNamespace(dbName, t.Name())
		cursor, err := (&command.ListIndexes{NS: ns}).RoundTrip(context.Background(), server.SelectedDescription(), server, conn)
		noerr(t, err)

		indexes := []string{}
		next := bson.NewDocument()

		for cursor.Next(context.Background()) {
			err = cursor.Decode(next)
			noerr(t, err)

			elem, err := next.Lookup("name")
			noerr(t, err)
			if elem.Value().Type() != bson.TypeString {
				t.Errorf("Incorrect type for 'name'. got %v; want %v", elem.Value().Type(), bson.TypeString)
				t.FailNow()
			}
			indexes = append(indexes, elem.Value().StringValue())
		}

		if len(indexes) != 5 {
			t.Errorf("Incorrect number of indexes. got %d; want %d", len(indexes), 5)
		}
		for i, want := range []string{"_id_", "a", "b", "c", "d_e"} {
			got := indexes[i]
			if got != want {
				t.Errorf("Mismatched index %d. got %s; want %s", i, got, want)
			}
		}
	})
	t.Run("MultipleBatch", func(t *testing.T) {
		server, err := topology.NewServer(addr.Addr(*host))
		noerr(t, err)
		conn, err := server.Connection(context.Background())
		noerr(t, err)
		dropcollection(t, server)
		addindexes(t, server, []string{"a"})
		addindexes(t, server, []string{"b"})
		addindexes(t, server, []string{"c"})

		ns := command.NewNamespace(dbName, t.Name())
		cursor, err := (&command.ListIndexes{NS: ns, Opts: []options.ListIndexesOptioner{options.OptBatchSize(1)}}).RoundTrip(context.Background(), server.SelectedDescription(), server, conn)
		noerr(t, err)

		indexes := []string{}
		next := bson.NewDocument()

		for cursor.Next(context.Background()) {
			err = cursor.Decode(next)
			noerr(t, err)

			elem, err := next.Lookup("name")
			noerr(t, err)
			if elem.Value().Type() != bson.TypeString {
				t.Errorf("Incorrect type for 'name'. got %v; want %v", elem.Value().Type(), bson.TypeString)
				t.FailNow()
			}
			indexes = append(indexes, elem.Value().StringValue())
		}

		if len(indexes) != 4 {
			t.Errorf("Incorrect number of indexes. got %d; want %d", len(indexes), 5)
		}
		for i, want := range []string{"_id_", "a", "b", "c"} {
			got := indexes[i]
			if got != want {
				t.Errorf("Mismatched index %d. got %s; want %s", i, got, want)
			}
		}
	})
}
