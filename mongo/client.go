// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"time"

	"sync"

	"sync/atomic"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/connstring"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/dispatch"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/tag"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
	"github.com/mongodb/mongo-go-driver/mongo/dbopt"
)

const defaultLocalThreshold = 15 * time.Millisecond

// Client performs operations on a given topology.
type Client struct {
	topologyOptions []topology.Option
	topology        *topology.Topology
	connString      connstring.ConnString
	localThreshold  time.Duration
	clusterTimeLock sync.Mutex
	clusterTime     atomic.Value // holds a bson.Document
	clusterTimeChan chan *bson.Document
	sessionPool     *session.Pool
	readPreference  *readpref.ReadPref
	readConcern     *readconcern.ReadConcern
	writeConcern    *writeconcern.WriteConcern
}

// Connect creates a new Client and then initializes it using the Connect method.
func Connect(ctx context.Context, uri string, opts ...clientopt.Option) (*Client, error) {
	c, err := NewClientWithOptions(uri, opts...)
	if err != nil {
		return nil, err
	}
	err = c.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// NewClient creates a new client to connect to a cluster specified by the uri.
func NewClient(uri string) (*Client, error) {
	cs, err := connstring.Parse(uri)
	if err != nil {
		return nil, err
	}

	return newClient(cs, nil)
}

// NewClientWithOptions creates a new client to connect to to a cluster specified by the connection
// string and the options manually passed in. If the same option is configured in both the
// connection string and the manual options, the manual option will be ignored.
func NewClientWithOptions(uri string, opts ...clientopt.Option) (*Client, error) {
	cs, err := connstring.Parse(uri)
	if err != nil {
		return nil, err
	}

	return newClient(cs, opts...)
}

// NewClientFromConnString creates a new client to connect to a cluster, with configuration
// specified by the connection string.
func NewClientFromConnString(cs connstring.ConnString) (*Client, error) {
	return newClient(cs, nil)
}

// Connect initializes the Client by starting background monitoring goroutines.
// This method must be called before a Client can be used.
func (c *Client) Connect(ctx context.Context) error {
	// Update on cluster time changes.
	go func() {
		for msg := range c.clusterTimeChan {
			c.SetClusterTime(msg)
		}
	}()

	return c.topology.Connect(ctx)
}

// Disconnect closes sockets to the topology referenced by this Client. It will
// shut down any monitoring goroutines, close the idle connection pool, and will
// wait until all the in use connections have been returned to the connection
// pool and closed before returning. If the context expires via cancellation,
// deadline, or timeout before the in use connections have returned, the in use
// connections will be closed, resulting in the failure of any in flight read
// or write operations. If this method returns with no errors, all connections
// associated with this Client have been closed.
func (c *Client) Disconnect(ctx context.Context) error {
	close(c.clusterTimeChan)
	c.endSessions(ctx)
	return c.topology.Disconnect(ctx)
}

// StartSession starts a new session.
func (c *Client) StartSession() (*Session, error) {
	sess, err := session.NewClientSession(c.sessionPool, session.Explicit, c.clusterTimeChan)
	if err != nil {
		return nil, err
	}

	return &Session{
		Sess: sess,
	}, nil
}

func (c *Client) startImplicitSession() (*session.Client, error) {
	sess, err := session.NewClientSession(c.sessionPool, session.Implicit, c.clusterTimeChan)
	if err != nil {
		return nil, err
	}

	return sess, nil
}

func (c *Client) endSessions(ctx context.Context) {
	cmd := command.EndSessions{
		ClusterTime: c.ClusterTime(),
		SessionPool: c.sessionPool,
	}

	_, _ = dispatch.EndSesions(ctx, cmd, c.topology, description.WriteSelector())
}

// SetClusterTime updates the client cluster time with the provided cluster time document if
// the provided time is greater than the client's own cluster time.
func (c *Client) SetClusterTime(ctdoc *bson.Document) {
	c.clusterTimeLock.Lock()
	c.clusterTime.Store(session.MaxClusterTime(c.ClusterTime(), ctdoc))
	c.clusterTimeLock.Unlock()
}

// ClusterTime returns the client cluster time.
func (c *Client) ClusterTime() *bson.Document {
	if ct, ok := c.clusterTime.Load().(bson.Document); ok {
		return &ct
	}
	return nil
}

func newClient(cs connstring.ConnString, opts ...clientopt.Option) (*Client, error) {
	clientOpt, err := clientopt.BundleClient(opts...).Unbundle(cs)
	if err != nil {
		return nil, err
	}

	client := &Client{
		topologyOptions: clientOpt.TopologyOptions,
		connString:      clientOpt.ConnString,
		localThreshold:  defaultLocalThreshold,
	}

	topts := append(
		client.topologyOptions,
		topology.WithConnString(func(connstring.ConnString) connstring.ConnString { return client.connString }),
	)
	topo, err := topology.New(topts...)
	if err != nil {
		return nil, err
	}
	client.topology = topo

	subscription, err := topo.Subscribe()
	if err != nil {
		return nil, err
	}
	client.sessionPool = session.NewPool(subscription.C)

	client.clusterTimeChan = make(chan *bson.Document)

	if client.readConcern == nil {
		client.readConcern = readConcernFromConnString(&client.connString)
	}
	if client.writeConcern == nil {
		client.writeConcern = writeConcernFromConnString(&client.connString)
	}
	if client.readPreference == nil {
		rp, err := readPreferenceFromConnString(&client.connString)
		if err != nil {
			return nil, err
		}
		if rp != nil {
			client.readPreference = rp
		} else {
			client.readPreference = readpref.Primary()
		}
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

func readPreferenceFromConnString(cs *connstring.ConnString) (*readpref.ReadPref, error) {
	var rp *readpref.ReadPref
	var err error
	options := make([]readpref.Option, 0, 1)

	tagSets := tag.NewTagSetsFromMaps(cs.ReadPreferenceTagSets)
	if len(tagSets) > 0 {
		options = append(options, readpref.WithTagSets(tagSets...))
	}

	if cs.MaxStaleness != 0 {
		options = append(options, readpref.WithMaxStaleness(cs.MaxStaleness))
	}

	if len(cs.ReadPreference) > 0 {
		if rp == nil {
			mode, _ := readpref.ModeFromString(cs.ReadPreference)
			rp, err = readpref.New(mode, options...)
			if err != nil {
				return nil, err
			}
		}
	}

	return rp, nil
}

// Database returns a handle for a given database.
func (c *Client) Database(name string, opts ...dbopt.Option) *Database {
	return newDatabase(c, name, opts...)
}

// ConnectionString returns the connection string of the cluster the client is connected to.
func (c *Client) ConnectionString() string {
	return c.connString.Original
}

func (c *Client) listDatabasesHelper(ctx context.Context, filter interface{},
	nameOnly bool) (ListDatabasesResult, error) {

	f, err := TransformDocument(filter)
	if err != nil {
		return ListDatabasesResult{}, err
	}

	opts := []option.ListDatabasesOptioner{}

	if nameOnly {
		opts = append(opts, option.OptNameOnly(nameOnly))
	}

	cmd := command.ListDatabases{Filter: f, Opts: opts}

	// The spec indicates that we should not run the listDatabase command on a secondary in a
	// replica set.
	res, err := dispatch.ListDatabases(ctx, cmd, c.topology, description.ReadPrefSelector(readpref.Primary()))
	if err != nil {
		return ListDatabasesResult{}, err
	}
	return (ListDatabasesResult{}).fromResult(res), nil
}

// ListDatabases returns a ListDatabasesResult.
func (c *Client) ListDatabases(ctx context.Context, filter interface{}) (ListDatabasesResult, error) {
	return c.listDatabasesHelper(ctx, filter, false)
}

// ListDatabaseNames returns a slice containing the names of all of the databases on the server.
func (c *Client) ListDatabaseNames(ctx context.Context, filter interface{}) ([]string, error) {
	res, err := c.listDatabasesHelper(ctx, filter, true)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0)
	for _, spec := range res.Databases {
		names = append(names, spec.Name)
	}

	return names, nil
}
