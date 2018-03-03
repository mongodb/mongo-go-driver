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

func TestListDatabases(t *testing.T) {
	noerr := func(t *testing.T, err error) {
		t.Helper()
		if err != nil {
			t.Errorf("Unepexted error: %v", err)
			t.FailNow()
		}
	}
	insertdocs := func(t *testing.T, server *topology.Server, documents []*bson.Document) {
		conn, err := server.Connection(context.Background())
		noerr(t, err)
		wc := writeconcern.New(writeconcern.WMajority())
		elem, err := wc.MarshalBSONElement()
		noerr(t, err)

		opt := options.OptWriteConcern{WriteConcern: elem, Acknowledged: wc.Acknowledged()}
		res, err := (&command.Insert{
			NS:   command.Namespace{DB: dbName, Collection: t.Name()},
			Docs: documents,
			Opts: []options.InsertOptioner{opt},
		}).RoundTrip(context.Background(), server.SelectedDescription(), conn)
		noerr(t, err)
		if res.N != len(documents) {
			t.Errorf("Failed to insert all documents. inserted %d; attempted %d", res.N, len(documents))
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
	server, err := topology.NewServer(addr.Addr(*host))
	noerr(t, err)
	conn, err := server.Connection(context.Background())
	noerr(t, err)

	dropcollection(t, server)
	insertdocs(t, server, []*bson.Document{bson.NewDocument(bson.EC.Int32("_id", 1))})

	res, err := (&command.ListDatabases{}).RoundTrip(context.Background(), server.SelectedDescription(), conn)
	noerr(t, err)
	var found bool
	for _, db := range res.Databases {
		if db.Name == dbName {
			found = true
		}
	}
	if !found {
		t.Error("Should have found database in listDatabases result, but didn't.")
	}
}
