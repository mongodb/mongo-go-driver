package integration

import (
	"context"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/addr"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/command"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
)

func TestCommandListCollections(t *testing.T) {
	noerr := func(t *testing.T, err error) {
		t.Helper()
		if err != nil {
			t.Errorf("Unepexted error: %v", err)
			t.FailNow()
		}
	}
	insertdocs := func(t *testing.T, server *topology.Server, collection string, documents ...*bson.Document) {
		conn, err := server.Connection(context.Background())
		noerr(t, err)
		wc := writeconcern.New(writeconcern.WMajority())
		elem, err := wc.MarshalBSONElement()
		noerr(t, err)

		opt := options.OptWriteConcern{WriteConcern: elem, Acknowledged: wc.Acknowledged()}
		res, err := (&command.Insert{
			NS:   command.Namespace{DB: dbName, Collection: collection},
			Docs: documents,
			Opts: []options.InsertOptioner{opt},
		}).RoundTrip(context.Background(), server.SelectedDescription(), conn)
		noerr(t, err)
		if res.N != len(documents) {
			t.Errorf("Failed to insert all documents. inserted %d; attempted %d", res.N, len(documents))
		}
	}
	dropcollection := func(t *testing.T, server *topology.Server, collection string) {
		conn, err := server.Connection(context.Background())
		noerr(t, err)
		_, err = (&command.Command{DB: dbName, Command: bson.NewDocument(bson.EC.String("drop", collection))}).RoundTrip(context.Background(), server.SelectedDescription(), conn)
		if err != nil && !command.IsNotFound(err) {
			noerr(t, err)
		}
	}

	t.Run("InvalidDatabaseName", func(t *testing.T) {
		server, err := topology.NewServer(addr.Addr(*host))
		noerr(t, err)
		conn, err := server.Connection(context.Background())
		noerr(t, err)

		_, err = (&command.ListCollections{}).RoundTrip(context.Background(), server.SelectedDescription(), server, conn)
		cmderr, ok := err.(command.CommandError)
		if !ok {
			t.Errorf("Incorrect type of command returned. got %T; want %T", err, command.CommandError{})
			t.FailNow()
		}
		if cmderr.Code != 73 {
			t.Errorf("Incorrect error code returned from server. got %d; want %d", cmderr.Code, 73)
		}
	})
	t.Run("SingleBatch", func(t *testing.T) {
		server, err := topology.NewServer(addr.Addr(*host))
		noerr(t, err)
		conn, err := server.Connection(context.Background())
		noerr(t, err)

		collOne := t.Name()
		collTwo := t.Name() + "2"
		collThree := t.Name() + "3"
		dropcollection(t, server, collOne)
		dropcollection(t, server, collTwo)
		dropcollection(t, server, collThree)
		insertdocs(t, server, collOne, bson.NewDocument(bson.EC.Int32("_id", 1)))
		insertdocs(t, server, collTwo, bson.NewDocument(bson.EC.Int32("_id", 2)))
		insertdocs(t, server, collThree, bson.NewDocument(bson.EC.Int32("_id", 3)))

		cursor, err := (&command.ListCollections{DB: dbName}).RoundTrip(context.Background(), server.SelectedDescription(), server, conn)
		noerr(t, err)

		names := map[string]bool{}
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
			names[elem.Value().StringValue()] = true
		}

		for _, required := range []string{collOne, collTwo, collThree} {
			_, ok := names[required]
			if !ok {
				t.Errorf("listCollections command did not return all collections. Missing %s", required)
			}
		}

	})
	t.Run("MultipleBatches", func(t *testing.T) {})
}
