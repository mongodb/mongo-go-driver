// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"

	"github.com/10gen/mongo-go-driver/mongo/private/cluster"
	"github.com/10gen/mongo-go-driver/mongo/private/ops"
	"github.com/10gen/mongo-go-driver/mongo/readconcern"
	"github.com/10gen/mongo-go-driver/mongo/readpref"
	"github.com/10gen/mongo-go-driver/mongo/writeconcern"
)

// Database performs operations on a given database.
type Database struct {
	client         *Client
	name           string
	readConcern    *readconcern.ReadConcern
	writeConcern   *writeconcern.WriteConcern
	readPreference *readpref.ReadPref
	readSelector   cluster.ServerSelector
	writeSelector  cluster.ServerSelector
}

func newDatabase(client *Client, name string) *Database {
	db := &Database{
		client:         client,
		name:           name,
		readPreference: client.readPreference,
		readConcern:    client.readConcern,
		writeConcern:   client.writeConcern,
	}

	latencySelector := cluster.LatencySelector(client.localThreshold)

	db.readSelector = cluster.CompositeSelector([]cluster.ServerSelector{
		readpref.Selector(db.readPreference),
		latencySelector,
	})

	db.writeSelector = readpref.Selector(readpref.Primary())

	return db
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
	return newCollection(db, name)
}

// RunCommand runs a command on the database.
func (db *Database) RunCommand(command interface{}, result interface{}) error {
	return db.RunCommandWithContext(context.Background(), command, result)
}

// RunCommandWithContext runs a command on the database. A user can supply a custom context to this method.
func (db *Database) RunCommandWithContext(ctx context.Context, command interface{}, result interface{}) error {
	s, err := db.client.selectServer(ctx, readpref.Selector(readpref.Primary()), readpref.Primary())
	if err != nil {
		return err
	}

	return ops.RunCommand(ctx, s, db.Name(), command, result)
}
