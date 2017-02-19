package server

import (
	"time"

	"github.com/10gen/mongo-go-driver/conn"
)

func newConfig(opts ...Option) *config {
	cfg := &config{
		dialer:            conn.Dial,
		heartbeatInterval: 10 * time.Second,
		maxConns:          100,
		maxIdleConns:      100,
	}

	cfg.apply(opts...)

	return cfg
}

// Option configures a server.
type Option func(*config)

type config struct {
	connOpts          []conn.Option
	dialer            conn.Dialer
	heartbeatInterval time.Duration
	maxConns          uint16
	maxIdleConns      uint16
}

func (c *config) reconfig(opts ...Option) *config {
	cfg := &config{
		connOpts:          c.connOpts,
		dialer:            c.dialer,
		heartbeatInterval: c.heartbeatInterval,
		maxConns:          c.maxConns,
		maxIdleConns:      c.maxIdleConns,
	}

	cfg.apply(opts...)
	return cfg
}

func (c *config) apply(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}

// WithConnectionDialer configures the dialer to use
// to create a new connection.
func WithConnectionDialer(dialer conn.Dialer) Option {
	return func(c *config) {
		c.dialer = dialer
	}
}

// WithConnectionOptions configures server's connections.
func WithConnectionOptions(opts ...conn.Option) Option {
	return func(c *config) {
		c.connOpts = opts
	}
}

// WithHeartbeatInterval configures a server's heartbeat interval.
// This option will be ignored when creating a Server with a
// pre-existing monitor.
func WithHeartbeatInterval(interval time.Duration) Option {
	return func(c *config) {
		c.heartbeatInterval = interval
	}
}

// WithMaxConnections configures maximum number of connections to
// allow for a given server. If max is 0, then there is no upper
// limit on the number of connections.
func WithMaxConnections(max uint16) Option {
	return func(c *config) {
		c.maxConns = max
	}
}

// WithMaxIdleConnections configures the maximum number of idle connections
// allowed for the server.
func WithMaxIdleConnections(size uint16) Option {
	return func(c *config) {
		c.maxIdleConns = size
	}
}
