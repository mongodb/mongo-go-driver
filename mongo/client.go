// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo/connstring"
	"github.com/mongodb/mongo-go-driver/mongo/private/cluster"
	"github.com/mongodb/mongo-go-driver/mongo/private/ops"
	"github.com/mongodb/mongo-go-driver/mongo/readconcern"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
)

const defaultLocalThreshold = 15 * time.Millisecond

// Client performs operations on a given cluster.
type Client struct {
	cluster        *cluster.Cluster
	connString     connstring.ConnString
	localThreshold time.Duration
	readPreference *readpref.ReadPref
	readConcern    *readconcern.ReadConcern
	writeConcern   *writeconcern.WriteConcern
}

// NewClient creates a new client to connect to a cluster specified by the uri.
func NewClient(uri string) (*Client, error) {
	cs, err := connstring.Parse(uri)
	if err != nil {
		return nil, err
	}

	return NewClientFromConnString(cs)
}

// NewClientWithOptions creates a new client to connect to to a cluster specified by the connection
// string and the options manually passed in. If the same option is configured in both the
// connection string and the manual options, the manual option will be ignored.
func NewClientWithOptions(uri string, opts *ClientOptions) (*Client, error) {
	cs, err := connstring.Parse(uri)
	if err != nil {
		return nil, err
	}

	for opts.opt != nil || opts.err != nil {
		if opts.err != nil {
			return nil, opts.err
		}

		opts.opt.ClientOption(&cs)
		opts = opts.next

	}

	return NewClientFromConnString(cs)
}

// NewClientFromConnString creates a new client to connect to a cluster, with configuration
// specified by the connection string.
func NewClientFromConnString(cs connstring.ConnString) (*Client, error) {
	clst, err := cluster.New(cluster.WithConnString(cs))
	if err != nil {
		return nil, err
	}

	// TODO(GODRIVER-92): Allow custom localThreshold
	client := &Client{
		cluster:        clst,
		connString:     cs,
		localThreshold: defaultLocalThreshold,
		readPreference: readpref.Primary(),
		readConcern:    readConcernFromConnString(&cs),
		writeConcern:   writeConcernFromConnString(&cs),
	}

	return client, nil
}

func readConcernFromConnString(cs *connstring.ConnString) *readconcern.ReadConcern {
	if len(cs.ReadConcernLevel) == 0 {
		return nil
	}

	rc := &readconcern.ReadConcern{}
	readconcern.Level(cs.ReadConcernLevel)(rc)

	return rc
}

func writeConcernFromConnString(cs *connstring.ConnString) *writeconcern.WriteConcern {
	var wc *writeconcern.WriteConcern

	if len(cs.WString) > 0 {
		if wc == nil {
			wc = writeconcern.New()
		}

		writeconcern.WTagSet(cs.WString)(wc)
	} else if cs.WNumberSet {
		if wc == nil {
			wc = writeconcern.New()
		}

		writeconcern.W(cs.WNumber)(wc)
	}

	if cs.JSet {
		if wc == nil {
			wc = writeconcern.New()
		}

		writeconcern.J(cs.J)(wc)
	}

	if cs.WTimeoutSet {
		if wc == nil {
			wc = writeconcern.New()
		}

		writeconcern.WTimeout(cs.WTimeout)(wc)
	}

	return wc
}

// Database returns a handle for a given database.
func (client *Client) Database(name string) *Database {
	return newDatabase(client, name)
}

// ConnectionString returns the connection string of the cluster the client is connected to.
func (client *Client) ConnectionString() string {
	return client.connString.Original
}

func (client *Client) selectServer(ctx context.Context, selector cluster.ServerSelector,
	readPref *readpref.ReadPref) (*ops.SelectedServer, error) {

	s, err := client.cluster.SelectServer(ctx, selector, readPref)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (client *Client) listDatabasesHelper(ctx context.Context, filter interface{},
	nameOnly bool) (Cursor, error) {

	// The spec indicates that we should not run the listDatabase command on a secondary in a
	// replica set.
	selector := cluster.CompositeSelector([]cluster.ServerSelector{
		readpref.Selector(readpref.Primary()),
	})

	s, err := client.selectServer(ctx, selector, readpref.Primary())
	if err != nil {
		return nil, err
	}

	f, err := TransformDocument(filter)
	if err != nil {
		return nil, err
	}

	return ops.ListDatabases(ctx, s, f, ops.ListDatabasesOptions{NameOnly: nameOnly})
}

// ListDatabases returns a Cursor iterating over descriptions of each of the databases on the server.
func (client *Client) ListDatabases(ctx context.Context, filter interface{}) (Cursor, error) {
	return client.listDatabasesHelper(ctx, filter, false)
}

// ListDatabaseNames returns a slice containing the names of all of the databases on the server.
func (client *Client) ListDatabaseNames(ctx context.Context, filter interface{}) ([]string, error) {
	c, err := client.listDatabasesHelper(ctx, filter, true)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0)
	for c.Next(ctx) {
		var db struct{ Name string }
		if err := c.Decode(&db); err != nil {
			return nil, err
		}

		names = append(names, db.Name)
	}
	if err := c.Err(); err != nil {
		return nil, err
	}

	return names, nil
}
