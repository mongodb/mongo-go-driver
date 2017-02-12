package cluster

import (
	"github.com/10gen/mongo-go-driver/desc"
	"github.com/10gen/mongo-go-driver/server"
)

func newConfig(opts ...Option) *config {
	cfg := &config{
		seedList: []desc.Endpoint{desc.Endpoint("localhost:27017")},
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
	seedList       []desc.Endpoint
	serverOpts     []server.Option
}

// ConnectionModeOpt configures the cluster's connection mode.
func ConnectionModeOpt(mode ConnectionMode) Option {
	return func(c *config) {
		c.connectionMode = mode
	}
}

// ReplicaSetName configures the cluster's default replica set name.
func ReplicaSetName(name string) Option {
	return func(c *config) {
		c.replicaSetName = name
	}
}

// SeedList configures a cluster's seed list.
func SeedList(endpoints ...desc.Endpoint) Option {
	return func(c *config) {
		c.seedList = endpoints
	}
}

// ServerOptions configures a cluster's server options for
// when a new server needs to get created.
func ServerOptions(opts ...server.Option) Option {
	return func(c *config) {
		c.serverOpts = opts
	}
}
