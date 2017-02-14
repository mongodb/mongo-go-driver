package cluster

import (
	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/server"
)

func newConfig(opts ...Option) *config {
	cfg := &config{
		seedList: []conn.Endpoint{conn.Endpoint("localhost:27017")},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// Option configures a cluster.
type Option func(*config)

type config struct {
	connectionMode ConnectionMode
	replicaSetName string
	seedList       []conn.Endpoint
	serverOpts     []server.Option
}

// WithConnectionMode configures the cluster's connection mode.
func WithConnectionMode(mode ConnectionMode) Option {
	return func(c *config) {
		c.connectionMode = mode
	}
}

// WithReplicaSetName configures the cluster's default replica set name.
func WithReplicaSetName(name string) Option {
	return func(c *config) {
		c.replicaSetName = name
	}
}

// WithSeedList configures a cluster's seed list.
func WithSeedList(endpoints ...conn.Endpoint) Option {
	return func(c *config) {
		c.seedList = endpoints
	}
}

// WithServerOptions configures a cluster's server options for
// when a new server needs to get created.
func WithServerOptions(opts ...server.Option) Option {
	return func(c *config) {
		c.serverOpts = opts
	}
}
