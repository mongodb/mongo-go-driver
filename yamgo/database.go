// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package yamgo

import (
	"github.com/10gen/mongo-go-driver/yamgo/private/cluster"
	"github.com/10gen/mongo-go-driver/yamgo/readconcern"
	"github.com/10gen/mongo-go-driver/yamgo/readpref"
	"github.com/10gen/mongo-go-driver/yamgo/writeconcern"
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
