package integration

import (
	"context"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
)

func TestListDatabases(t *testing.T) {
	noerr := func(t *testing.T, err error) {
		// t.Helper()
		if err != nil {
			t.Errorf("Unepexted error: %v", err)
			t.FailNow()
		}
	}
	server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
	noerr(t, err)
	conn, err := server.Connection(context.Background())
	noerr(t, err)

	wc := writeconcern.New(writeconcern.WMajority())
	testutil.AutoDropCollection(t)
	testutil.AutoInsertDocs(t, wc, bson.NewDocument(bson.EC.Int32("_id", 1)))

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
