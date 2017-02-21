package cluster

import (
	"time"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/server"
)

func newConfig(opts ...Option) *config {
	cfg := &config{
		seedList:               []conn.Endpoint{conn.Endpoint("localhost:27017")},
		serverSelectionTimeout: 30 * time.Second,
	}

	cfg.apply(opts...)

	return cfg
}

// Option configures a cluster.
type Option func(*config)

type config struct {
	mode                   MonitorMode
	replicaSetName         string
	seedList               []conn.Endpoint
	serverOpts             []server.Option
	serverSelectionTimeout time.Duration
}

func (c *config) reconfig(opts ...Option) *config {
	cfg := &config{
		mode:                   c.mode,
		replicaSetName:         c.replicaSetName,
		seedList:               c.seedList,
		serverOpts:             c.serverOpts,
		serverSelectionTimeout: c.serverSelectionTimeout,
	}

	cfg.apply(opts...)
	return cfg
}

func (c *config) apply(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}

// WithMode configures the cluster's monitor mode.
// This option will be ignored when the cluster is created with a
// pre-existing monitor.
func WithMode(mode MonitorMode) Option {
	return func(c *config) {
		c.mode = mode
	}
}

// WithReplicaSetName configures the cluster's default replica set name.
// This option will be ignored when the cluster is created with a
// pre-existing monitor.
func WithReplicaSetName(name string) Option {
	return func(c *config) {
		c.replicaSetName = name
	}
}

// WithSeedList configures a cluster's seed list.
// This option will be ignored when the cluster is created with a
// pre-existing monitor.
func WithSeedList(endpoints ...conn.Endpoint) Option {
	return func(c *config) {
		c.seedList = endpoints
	}
}

// WithServerSelectionTimeout configures a cluster's server selection timeout.
func WithServerSelectionTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.serverSelectionTimeout = timeout
	}
}

// WithServerOptions configures a cluster's server options for
// when a new server needs to get created.
func WithServerOptions(opts ...server.Option) Option {
	return func(c *config) {
		c.serverOpts = opts
	}
}
