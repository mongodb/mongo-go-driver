package topologyx

import (
	"net"
	"time"

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/x/network/compressor"
)

type connectionConfig struct {
	appName        string
	connectTimeout time.Duration
	dialer         Dialer
	handshaker     Handshaker
	idleTimeout    time.Duration
	lifeTimeout    time.Duration
	cmdMonitor     *event.CommandMonitor
	readTimeout    time.Duration
	writeTimeout   time.Duration
	tlsConfig      *TLSConfig
	compressors    []compressor.Compressor
}

func newConnectionConfig(opts ...ConnectionOption) (*connectionConfig, error) {
	cfg := &connectionConfig{
		connectTimeout: 30 * time.Second,
		dialer:         nil,
		idleTimeout:    10 * time.Minute,
		lifeTimeout:    30 * time.Minute,
	}

	for _, opt := range opts {
		err := opt(cfg)
		if err != nil {
			return nil, err
		}
	}

	if cfg.dialer == nil {
		cfg.dialer = &net.Dialer{Timeout: cfg.connectTimeout}
	}

	return cfg, nil
}

// ConnectionOption is used to configure a connection.
type ConnectionOption func(*connectionConfig) error

// WithAppName sets the application name which gets sent to MongoDB when it
// first connects.
func WithAppName(fn func(string) string) ConnectionOption {
	return func(c *connectionConfig) error {
		c.appName = fn(c.appName)
		return nil
	}
}

// WithCompressors sets the compressors that can be used for communication.
func WithCompressors(fn func([]compressor.Compressor) []compressor.Compressor) ConnectionOption {
	return func(c *connectionConfig) error {
		c.compressors = fn(c.compressors)
		return nil
	}
}

// WithConnectTimeout configures the maximum amount of time a dial will wait for a
// connect to complete. The default is 30 seconds.
func WithConnectTimeout(fn func(time.Duration) time.Duration) ConnectionOption {
	return func(c *connectionConfig) error {
		c.connectTimeout = fn(c.connectTimeout)
		return nil
	}
}

// WithDialer configures the Dialer to use when making a new connection to MongoDB.
func WithDialer(fn func(Dialer) Dialer) ConnectionOption {
	return func(c *connectionConfig) error {
		c.dialer = fn(c.dialer)
		return nil
	}
}

// WithHandshaker configures the Handshaker that wll be used to initialize newly
// dialed connections.
func WithHandshaker(fn func(Handshaker) Handshaker) ConnectionOption {
	return func(c *connectionConfig) error {
		c.handshaker = fn(c.handshaker)
		return nil
	}
}

// WithIdleTimeout configures the maximum idle time to allow for a connection.
func WithIdleTimeout(fn func(time.Duration) time.Duration) ConnectionOption {
	return func(c *connectionConfig) error {
		c.idleTimeout = fn(c.idleTimeout)
		return nil
	}
}

// WithLifeTimeout configures the maximum life of a connection.
func WithLifeTimeout(fn func(time.Duration) time.Duration) ConnectionOption {
	return func(c *connectionConfig) error {
		c.lifeTimeout = fn(c.lifeTimeout)
		return nil
	}
}

// WithReadTimeout configures the maximum read time for a connection.
func WithReadTimeout(fn func(time.Duration) time.Duration) ConnectionOption {
	return func(c *connectionConfig) error {
		c.readTimeout = fn(c.readTimeout)
		return nil
	}
}

// WithWriteTimeout configures the maximum write time for a connection.
func WithWriteTimeout(fn func(time.Duration) time.Duration) ConnectionOption {
	return func(c *connectionConfig) error {
		c.writeTimeout = fn(c.writeTimeout)
		return nil
	}
}

// WithTLSConfig configures the TLS options for a connection.
func WithTLSConfig(fn func(*TLSConfig) *TLSConfig) ConnectionOption {
	return func(c *connectionConfig) error {
		c.tlsConfig = fn(c.tlsConfig)
		return nil
	}
}

// WithMonitor configures a event for command monitoring.
func WithMonitor(fn func(*event.CommandMonitor) *event.CommandMonitor) ConnectionOption {
	return func(c *connectionConfig) error {
		c.cmdMonitor = fn(c.cmdMonitor)
		return nil
	}
}
