package connection

import (
	"time"
)

type config struct {
	appName        string
	connectTimeout time.Duration
	dialer         Dialer
	configurer     Configurer
	idleTimeout    time.Duration
	lifeTimeout    time.Duration
	readTimeout    time.Duration
	writeTimeout   time.Duration
	tlsConfig      *TLSConfig
}

func newConfig(opts ...Option) (*config, error) {
	cfg := &config{
		connectTimeout: 30 * time.Second,
		dialer:         DefaultDialer,
		idleTimeout:    10 * time.Minute,
		lifeTimeout:    30 * time.Minute,
	}

	for _, opt := range opts {
		err := opt(cfg)
		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// Option is used to configure a connection.
type Option func(*config) error

// WithAppName sets the application name which gets sent to MongoDB when it
// first connects.
func WithAppName(fn func(string) string) Option {
	return func(c *config) error {
		c.appName = fn(c.appName)
		return nil
	}
}

// WithConnectTimeout configures the maximum amount of time a dial will wait for a
// connect to complete. The default is 30 seconds.
func WithConnectTimeout(fn func(time.Duration) time.Duration) Option {
	return func(c *config) error {
		c.connectTimeout = fn(c.connectTimeout)
		return nil
	}
}

// WithDialer configures the Dialer to use when making a new connection to MongoDB.
func WithDialer(fn func(Dialer) Dialer) Option {
	return func(c *config) error {
		c.dialer = fn(c.dialer)
		return nil
	}
}

// WithConfigurer configures the Configurers that will be used to configure newly
// dialed connections.
func WithConfigurer(fn func(Configurer) Configurer) Option {
	return func(c *config) error {
		c.configurer = fn(c.configurer)
		return nil
	}
}

// WithIdleTimeout configures the maximum idle time to allow for a connection.
func WithIdleTimeout(fn func(time.Duration) time.Duration) Option {
	return func(c *config) error {
		c.idleTimeout = fn(c.idleTimeout)
		return nil
	}
}

// WithLifeTimeout configures the maximum life of a connection.
func WithLifeTimeout(fn func(time.Duration) time.Duration) Option {
	return func(c *config) error {
		c.lifeTimeout = fn(c.lifeTimeout)
		return nil
	}
}

// WithReadTimeout configures the maximum read time for a connection.
func WithReadTimeout(fn func(time.Duration) time.Duration) Option {
	return func(c *config) error {
		c.readTimeout = fn(c.readTimeout)
		return nil
	}
}

// WithWriteTimeout configures the maximum write time for a connection.
func WithWriteTimeout(fn func(time.Duration) time.Duration) Option {
	return func(c *config) error {
		c.writeTimeout = fn(c.writeTimeout)
		return nil
	}
}

// WithTLSConfig configures the TLS options for a connection.
func WithTLSConfig(fn func(*TLSConfig) *TLSConfig) Option {
	return func(c *config) error {
		c.tlsConfig = fn(c.tlsConfig)
		return nil
	}
}
