package server

import (
	"time"

	"github.com/10gen/mongo-go-driver/conn"
)

func newConfig(opts ...Option) *config {
	cfg := &config{
		connDialer:        conn.Dial,
		heartbeatInterval: time.Duration(10) * time.Second,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.heartbeatDialer == nil {
		cfg.heartbeatDialer = cfg.connDialer
	}

	return cfg
}

// Option configures a server.
type Option func(*config)

type config struct {
	connOpts          []conn.Option
	connDialer        conn.Dialer
	heartbeatDialer   conn.Dialer
	heartbeatInterval time.Duration
}

// WithConnectionDialer configures the dialer to use
// to create a new connection.
func WithConnectionDialer(dialer conn.Dialer) Option {
	return func(c *config) {
		c.connDialer = dialer
	}
}

// WithConnectionOptions configures server's connections.
func WithConnectionOptions(opts ...conn.Option) Option {
	return func(c *config) {
		c.connOpts = opts
	}
}

// WithHearbeatDialer configures the dialer to be used
// for hearbeat connections. By default, it will use the
// configured ConnectionDialer.
func WithHearbeatDialer(dialer conn.Dialer) Option {
	return func(c *config) {
		c.heartbeatDialer = dialer
	}
}

// WithHeartbeatInterval configures a server's heartbeat interval.
func WithHeartbeatInterval(interval time.Duration) Option {
	return func(c *config) {
		c.heartbeatInterval = interval
	}
}
