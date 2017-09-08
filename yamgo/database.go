package yamgo

import (
	"context"

	"github.com/10gen/mongo-go-driver/yamgo/private/cluster"
	"github.com/10gen/mongo-go-driver/yamgo/private/ops"
	"github.com/10gen/mongo-go-driver/yamgo/readpref"
)

// Database performs operations on a given database.
type Database struct {
	client *Client
	name   string
}

// Client returns the Client the database was created from.
func (db *Database) Client() *Client {
	return db.client
}

// Name returns the name of the database.
func (db *Database) Name() string {
	return db.name
}

// Collection gets a handle for a given collection in the database.
func (db *Database) Collection(name string) *Collection {
	return &Collection{db: db, name: name}
}

func (db *Database) selectServer(ctx context.Context, selector cluster.ServerSelector,
	pref *readpref.ReadPref) (*ops.SelectedServer, error) {

	return db.client.selectServer(ctx, selector, pref)
}
