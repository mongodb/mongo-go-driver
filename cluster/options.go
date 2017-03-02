package cluster

import (
	"time"

	"github.com/10gen/mongo-go-driver/auth"
	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/connstring"
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

// WithConnString configures the cluster using the connection
// string.
func WithConnString(cs connstring.ConnString) Option {
	return func(c *config) {

		var connOpts []conn.Option

		if cs.AppName != "" {
			connOpts = append(connOpts, conn.WithAppName(cs.AppName))
		}

		switch cs.Connect {
		case connstring.SingleConnect:
			c.mode = SingleMode
		}

		c.seedList = []conn.Endpoint{}
		for _, host := range cs.Hosts {
			c.seedList = append(c.seedList, conn.Endpoint(host))
		}

		if cs.HeartbeatInterval > 0 {
			c.serverOpts = append(c.serverOpts, server.WithHeartbeatInterval(cs.HeartbeatInterval))
		}

		if cs.MaxConnIdleTime > 0 {
			connOpts = append(connOpts, conn.WithIdleTimeout(cs.MaxConnIdleTime))
		}

		if cs.MaxConnLifeTime > 0 {
			connOpts = append(connOpts, conn.WithIdleTimeout(cs.MaxConnLifeTime))
		}

		if cs.MaxConnsPerHostSet {
			c.serverOpts = append(c.serverOpts, server.WithMaxConnections(cs.MaxConnsPerHost))
		}

		if cs.MaxIdleConnsPerHostSet {
			c.serverOpts = append(c.serverOpts, server.WithMaxIdleConnections(cs.MaxIdleConnsPerHost))
		}

		if cs.ReplicaSet != "" {
			c.replicaSetName = cs.ReplicaSet
		}

		if cs.ServerSelectionTimeout > 0 {
			c.serverSelectionTimeout = cs.ServerSelectionTimeout
		}

		if cs.Username != "" {
			var source string
			if cs.AuthSource != "" {
				source = cs.AuthSource
			} else if cs.Database != "" {
				source = cs.Database
			} else {
				source = "admin"
			}

			if authenticator, err := auth.CreateAuthenticator(
				cs.AuthMechanism,
				source,
				cs.Username,
				cs.Password,
				cs.AuthMechanismProperties); err != nil {

				c.serverOpts = append(
					c.serverOpts,
					server.WithWrappedConnectionDialer(func(current conn.Dialer) conn.Dialer {
						return auth.Dialer(current, authenticator)
					}),
				)
			}
		}

		if len(connOpts) > 0 {
			c.serverOpts = append(c.serverOpts, server.WithMoreConnectionOptions(connOpts...))
		}
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
// when a new server needs to get created. The options provided
// overwrite all previously configured options.
func WithServerOptions(opts ...server.Option) Option {
	return func(c *config) {
		c.serverOpts = opts
	}
}

// WithMoreServerOptions configures a cluster's server options for
// when a new server needs to get created. The options provided are
// appended to any current options and may override previously
// configured options.
func WithMoreServerOptions(opts ...server.Option) Option {
	return func(c *config) {
		c.serverOpts = append(c.serverOpts, opts...)
	}
}
