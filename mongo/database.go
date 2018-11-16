// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/readconcern"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver"
	"github.com/mongodb/mongo-go-driver/x/network/command"
	"github.com/mongodb/mongo-go-driver/x/network/description"
)

// Database performs operations on a given database.
type Database struct {
	client         *Client
	name           string
	readConcern    *readconcern.ReadConcern
	writeConcern   *writeconcern.WriteConcern
	readPreference *readpref.ReadPref
	readSelector   description.ServerSelector
	writeSelector  description.ServerSelector
	registry       *bsoncodec.Registry
}

func newDatabase(client *Client, name string, opts ...*options.DatabaseOptions) *Database {
	dbOpt := options.MergeDatabaseOptions(opts...)

	rc := client.readConcern
	if dbOpt.ReadConcern != nil {
		rc = dbOpt.ReadConcern
	}

	rp := client.readPreference
	if dbOpt.ReadPreference != nil {
		rp = dbOpt.ReadPreference
	}

	wc := client.writeConcern
	if dbOpt.WriteConcern != nil {
		wc = dbOpt.WriteConcern
	}

	db := &Database{
		client:         client,
		name:           name,
		readPreference: rp,
		readConcern:    rc,
		writeConcern:   wc,
		registry:       client.registry,
	}

	db.readSelector = description.CompositeSelector([]description.ServerSelector{
		description.ReadPrefSelector(db.readPreference),
		description.LatencySelector(db.client.localThreshold),
	})

	db.writeSelector = description.CompositeSelector([]description.ServerSelector{
		description.WriteSelector(),
		description.LatencySelector(db.client.localThreshold),
	})

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
func (db *Database) Collection(name string, opts ...*options.CollectionOptions) *Collection {
	return newCollection(db, name, opts...)
}

// RunCommand runs a command on the database. A user can supply a custom
// context to this method, or nil to default to context.Background().
func (db *Database) RunCommand(ctx context.Context, runCommand interface{}, opts ...*options.RunCmdOptions) (bson.Raw, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	sess := sessionFromContext(ctx)

	runCmd := options.MergeRunCmdOptions(opts...)
	rp := runCmd.ReadPreference
	if rp == nil {
		if sess != nil && sess.TransactionRunning() {
			rp = sess.CurrentRp // override with transaction read pref if specified
		}
		if rp == nil {
			rp = readpref.Primary() // set to primary if nothing specified in options
		}
	}

	readSelect := description.CompositeSelector([]description.ServerSelector{
		description.ReadPrefSelector(rp),
		description.LatencySelector(db.client.localThreshold),
	})

	runCmdDoc, err := transformDocument(db.registry, runCommand)
	if err != nil {
		return nil, err
	}
	result, err := driver.Read(ctx,
		command.Read{
			DB:       db.Name(),
			Command:  runCmdDoc,
			ReadPref: rp,
			Session:  sess,
			Clock:    db.client.clock,
		},
		db.client.topology,
		readSelect,
		db.client.id,
		db.client.topology.SessionPool,
	)

	return result, replaceTopologyErr(err)
}

// Drop drops this database from mongodb.
func (db *Database) Drop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	sess := sessionFromContext(ctx)

	err := db.client.ValidSession(sess)
	if err != nil {
		return err
	}

	cmd := command.DropDatabase{
		DB:      db.name,
		Session: sess,
		Clock:   db.client.clock,
	}
	_, err = driver.DropDatabase(
		ctx, cmd,
		db.client.topology,
		db.writeSelector,
		db.client.id,
		db.client.topology.SessionPool,
	)
	if err != nil && !command.IsNotFound(err) {
		return replaceTopologyErr(err)
	}
	return nil
}

// ListCollections list collections from mongodb database.
func (db *Database) ListCollections(ctx context.Context, filter interface{}, opts ...*options.ListCollectionsOptions) (Cursor, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	sess := sessionFromContext(ctx)

	err := db.client.ValidSession(sess)
	if err != nil {
		return nil, err
	}

	var filterDoc bsonx.Doc
	if filter != nil {
		filterDoc, err = transformDocument(db.registry, filter)
		if err != nil {
			return nil, err
		}
	}

	cmd := command.ListCollections{
		DB:       db.name,
		Filter:   filterDoc,
		ReadPref: db.readPreference,
		Session:  sess,
		Clock:    db.client.clock,
	}

	cursor, err := driver.ListCollections(
		ctx, cmd,
		db.client.topology,
		db.readSelector,
		db.client.id,
		db.client.topology.SessionPool,
		opts...,
	)
	if err != nil && !command.IsNotFound(err) {
		return nil, replaceTopologyErr(err)
	}

	return cursor, nil

}

// ReadConcern returns the read concern of this database.
func (db *Database) ReadConcern() *readconcern.ReadConcern {
	return db.readConcern
}

// ReadPreference returns the read preference of this database.
func (db *Database) ReadPreference() *readpref.ReadPref {
	return db.readPreference
}

// WriteConcern returns the write concern of this database.
func (db *Database) WriteConcern() *writeconcern.WriteConcern {
	return db.writeConcern
}
