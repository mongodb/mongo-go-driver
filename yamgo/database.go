package yamgo

import (
	"context"

	"github.com/10gen/mongo-go-driver/readpref"
)

type Database struct {
	client *Client
	name   string
}

func (db *Database) Client() *Client {
	return db.client
}

func (db *Database) Name() string {
	return db.name
}

func (db *Database) Collection(name string) *Collection {
	return &Collection{db: db, name: name}
}

func (db *Database) RunCommand(ctx context.Context, readPref *readpref.ReadPref, command interface{}, result interface{}) error {
	return db.client.RunCommand(ctx, db.name, command, result)
}
