// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/connstring"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/dispatch"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/tag"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/core/uuid"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
	"github.com/mongodb/mongo-go-driver/mongo/dbopt"
	"github.com/mongodb/mongo-go-driver/mongo/listdbopt"
	"github.com/mongodb/mongo-go-driver/mongo/sessionopt"
)

const defaultLocalThreshold = 15 * time.Millisecond

var defaultRegistry = bson.NewRegistryBuilder().Build()

// Client performs operations on a given topology.
type Client struct {
	id              uuid.UUID
	topologyOptions []topology.Option
	topology        *topology.Topology
	connString      connstring.ConnString
	localThreshold  time.Duration
	retryWrites     bool
	clock           *session.ClusterClock
	readPreference  *readpref.ReadPref
	readConcern     *readconcern.ReadConcern
	writeConcern    *writeconcern.WriteConcern
	registry        *bsoncodec.Registry
	marshaller      BSONAppender
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
	err := c.topology.Connect(ctx)
	if err != nil {
		return err
	}

	return nil

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
	c.endSessions(ctx)
	return c.topology.Disconnect(ctx)
}

// Ping verifies that the client can connect to the topology.
// If readPreference is nil then will use the client's default read
// preference.
func (c *Client) Ping(ctx context.Context, rp *readpref.ReadPref) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if rp == nil {
		rp = c.readPreference
	}

	_, err := c.topology.SelectServer(ctx, description.ReadPrefSelector(rp))
	return err
}

// StartSession starts a new session.
func (c *Client) StartSession(opts ...sessionopt.Session) (Session, error) {
	if c.topology.SessionPool == nil {
		return nil, topology.ErrTopologyClosed
	}

	// By default the session inherits the default read/write concerns of the client
	defaultOpts := []sessionopt.Session{
		sessionopt.DefaultReadConcern(c.readConcern),
		sessionopt.DefaultReadPreference(c.readPreference),
		sessionopt.DefaultWriteConcern(c.writeConcern),
	}

	// If the user provided the default read/write concerns explicitly, this will overwrite with them.
	sessionOpts, err := sessionopt.BundleSession(append(defaultOpts, opts...)...).Unbundle(true)
	if err != nil {
		return nil, err
	}

	sess, err := session.NewClientSession(c.topology.SessionPool, c.id, session.Explicit, sessionOpts...)
	if err != nil {
		return nil, err
	}

	sess.RetryWrite = c.retryWrites

	return &sessionImpl{
		Client: sess,
		topo:   c.topology,
	}, nil
}

func (c *Client) endSessions(ctx context.Context) {
	cmd := command.EndSessions{
		Clock:      c.clock,
		SessionIDs: c.topology.SessionPool.IDSlice(),
	}

	_, _ = dispatch.EndSessions(ctx, cmd, c.topology, description.ReadPrefSelector(readpref.PrimaryPreferred()))
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
		registry:        clientOpt.Registry,
	}

	uuid, err := uuid.New()
	if err != nil {
		return nil, err
	}
	client.id = uuid

	topts := append(
		client.topologyOptions,
		topology.WithConnString(func(connstring.ConnString) connstring.ConnString { return client.connString }),
		topology.WithServerOptions(func(opts ...topology.ServerOption) []topology.ServerOption {
			return append(opts, topology.WithClock(func(clock *session.ClusterClock) *session.ClusterClock {
				return client.clock
			}))
		}),
	)
	topo, err := topology.New(topts...)
	if err != nil {
		return nil, err
	}
	client.topology = topo
	client.clock = &session.ClusterClock{}

	if client.readConcern == nil {
		client.readConcern = readConcernFromConnString(&client.connString)

		if client.readConcern == nil {
			// no read concern in conn string
			client.readConcern = readconcern.New()
		}
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

	if client.registry == nil {
		client.registry = defaultRegistry
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

// ValidSession returns an error if the session doesn't belong to the client
func (c *Client) ValidSession(sess *session.Client) error {
	if sess != nil && !uuid.Equal(sess.ClientID, c.id) {
		return ErrWrongClient
	}
	return nil
}

// Database returns a handle for a given database.
func (c *Client) Database(name string, opts ...dbopt.Option) *Database {
	return newDatabase(c, name, opts...)
}

// ConnectionString returns the connection string of the cluster the client is connected to.
func (c *Client) ConnectionString() string {
	return c.connString.Original
}

// ListDatabases returns a ListDatabasesResult.
func (c *Client) ListDatabases(ctx context.Context, filter interface{}, opts ...listdbopt.ListDatabases) (ListDatabasesResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	listDbOpts, _, err := listdbopt.BundleListDatabases(opts...).Unbundle(true)
	if err != nil {
		return ListDatabasesResult{}, err
	}

	sess := sessionFromContext(ctx)

	err = c.ValidSession(sess)
	if err != nil {
		return ListDatabasesResult{}, err
	}

	f, err := transformDocument(c.registry, filter)
	if err != nil {
		return ListDatabasesResult{}, err
	}

	cmd := command.ListDatabases{
		Filter:  f,
		Opts:    listDbOpts,
		Session: sess,
		Clock:   c.clock,
	}

	res, err := dispatch.ListDatabases(
		ctx, cmd,
		c.topology,
		description.ReadPrefSelector(readpref.Primary()),
		c.id,
		c.topology.SessionPool,
	)
	if err != nil {
		return ListDatabasesResult{}, err
	}

	return (ListDatabasesResult{}).fromResult(res), nil
}

// ListDatabaseNames returns a slice containing the names of all of the databases on the server.
func (c *Client) ListDatabaseNames(ctx context.Context, filter interface{}, opts ...listdbopt.ListDatabases) ([]string, error) {
	opts = append(opts, listdbopt.NameOnly(true))
	res, err := c.ListDatabases(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0)
	for _, spec := range res.Databases {
		names = append(names, spec.Name)
	}

	return names, nil
}

// WithSession allows a user to start a session themselves and manage
// its lifetime. The only way to provide a session to a CRUD method is
// to invoke that CRUD method with the mongo.SessionContext within the
// closure. The mongo.SessionContext can be used as a regular context,
// so methods like context.WithDeadline and context.WithTimeout are
// supported.
//
// If the context.Context already has a mongo.Session attached, that
// mongo.Session will be replaced with the one provided.
//
// Errors returned from the closure are transparently returned from
// this function.
func WithSession(ctx context.Context, sess Session, fn func(SessionContext) error) error {
	return fn(contextWithSession(ctx, sess))
}

// UseSession creates a default session, that is only valid for the
// lifetime of the closure. No cleanup outside of closing the session
// is done upon exiting the closure. This means that an outstanding
// transaction will be aborted, even if the closure returns an error.
//
// If ctx already contains a mongo.Session, that mongo.Session will be
// replaced with the newly created mongo.Session.
//
// Errors returned from the closure are transparently returned from
// this method.
func (c *Client) UseSession(ctx context.Context, fn func(SessionContext) error) error {
	return c.UseSessionWithOptions(ctx, []sessionopt.Session{}, fn)
}

// UseSessionWithOptions works like UseSession but allows the caller
// to specify the options used to create the session.
func (c *Client) UseSessionWithOptions(ctx context.Context, opts []sessionopt.Session, fn func(SessionContext) error) error {
	defaultSess, err := c.StartSession(opts...)
	if err != nil {
		return err
	}

	defer defaultSess.EndSession(ctx)

	sessCtx := sessionContext{
		Context: context.WithValue(ctx, sessionKey{}, defaultSess),
		Session: defaultSess,
	}

	return fn(sessCtx)
}
