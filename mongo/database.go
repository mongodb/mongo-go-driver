// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/dispatch"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/mongo/collectionopt"
	"github.com/mongodb/mongo-go-driver/mongo/dbopt"
	"github.com/mongodb/mongo-go-driver/mongo/runcmdopt"
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
}

func newDatabase(client *Client, name string, opts ...dbopt.Option) *Database {
	dbOpt, err := dbopt.BundleDatabase(opts...).Unbundle()
	if err != nil {
		return nil
	}

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
	}

	db.readSelector = description.CompositeSelector([]description.ServerSelector{
		description.ReadPrefSelector(db.readPreference),
		description.LatencySelector(db.client.localThreshold),
	})

	db.writeSelector = description.WriteSelector()

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
func (db *Database) Collection(name string, opts ...collectionopt.Option) *Collection {
	return newCollection(db, name, opts...)
}

// RunCommand runs a command on the database. A user can supply a custom
// context to this method, or nil to default to context.Background().
func (db *Database) RunCommand(ctx context.Context, runCommand interface{}, opts ...runcmdopt.Option) (bson.Reader, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	runCmd, err := runcmdopt.BundleRunCmd(opts...).Unbundle()
	if err != nil {
		return nil, err
	}
	rp := runCmd.ReadPreference
	if rp == nil {
		rp = db.readPreference // inherit from db if nothing specified in options
	}

	runCmdDoc, err := TransformDocument(runCommand)
	if err != nil {
		return nil, err
	}
	return dispatch.Read(ctx, command.Read{
		DB:       db.Name(),
		Command:  runCmdDoc,
		ReadPref: rp,
	}, db.client.topology, db.writeSelector)
}

// Drop drops this database from mongodb.
func (db *Database) Drop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	cmd := command.DropDatabase{
		DB: db.name,
	}
	_, err := dispatch.DropDatabase(ctx, cmd, db.client.topology, db.writeSelector)
	if err != nil && !command.IsNotFound(err) {
		return err
	}
	return nil
}

// ListCollections list collections from mongodb database.
func (db *Database) ListCollections(ctx context.Context, filter *bson.Document, options ...option.ListCollectionsOptioner) (command.Cursor, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := command.ListCollections{
		DB:       db.name,
		Filter:   filter,
		Opts:     options,
		ReadPref: db.readPreference,
	}
	cursor, err := dispatch.ListCollections(ctx, cmd, db.client.topology, db.readSelector)
	if err != nil && !command.IsNotFound(err) {
		return nil, err
	}
	return cursor, nil

}
