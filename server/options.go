package server

import (
	"time"

	"github.com/10gen/mongo-go-driver/conn"
)

func newConfig(opts ...Option) *config {
	cfg := &config{
		dialer: conn.Dial,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	defaultMonitorOpts := []MonitorOption{
		WithHeartbeatConnectionOptions(cfg.connOpts...),
		WithHearbeatDialer(cfg.dialer),
	}

	cfg.monitorOpts = append(defaultMonitorOpts, cfg.monitorOpts...)

	return cfg
}

// Option configures a server.
type Option func(*config)

type config struct {
	connOpts    []conn.Option
	dialer      conn.Dialer
	monitorOpts []MonitorOption
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

// WithMonitorOptions configures the monitor. This option
// will be ignored on a server that does not create it's own
// monitor.
func WithMonitorOptions(opts ...MonitorOption) Option {
	return func(c *config) {
		c.monitorOpts = opts
	}
}

func newMonitorConfig(opts ...MonitorOption) *monitorConfig {
	cfg := &monitorConfig{
		dialer:            conn.Dial,
		heartbeatInterval: time.Duration(10) * time.Second,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// MonitorOption configures a Monitor.
type MonitorOption func(c *monitorConfig)

type monitorConfig struct {
	connOpts          []conn.Option
	dialer            conn.Dialer
	heartbeatInterval time.Duration
}

// WithHeartbeatConnectionOptions configures server's connections. By
// default, it will use the configured ConnectionOptions.
func WithHeartbeatConnectionOptions(opts ...conn.Option) MonitorOption {
	return func(c *monitorConfig) {
		c.connOpts = opts
	}
}

// WithHearbeatDialer configures the dialer to be used
// for hearbeat connections. By default, it will use the
// configured ConnectionDialer.
func WithHearbeatDialer(dialer conn.Dialer) MonitorOption {
	return func(c *monitorConfig) {
		c.dialer = dialer
	}
}

// WithHeartbeatInterval configures a server's heartbeat interval.
func WithHeartbeatInterval(interval time.Duration) MonitorOption {
	return func(c *monitorConfig) {
		c.heartbeatInterval = interval
	}
}
