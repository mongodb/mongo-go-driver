package conn

import (
	"time"

	"github.com/10gen/mongo-go-driver/msg"
)

func newConfig(opts ...Option) *config {
	cfg := &config{
		codec:       msg.NewWireProtocolCodec(),
		dialer:      DialEndpoint,
		idleTimeout: 10 * time.Minute,
		lifeTimeout: 30 * time.Minute,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// Option configures a connection.
type Option func(*config)

type config struct {
	appName     string
	codec       msg.Codec
	dialer      EndpointDialer
	idleTimeout time.Duration
	lifeTimeout time.Duration
}

// WithAppName sets the application name which gets
// sent to MongoDB on first connection.
func WithAppName(name string) Option {
	return func(c *config) {
		c.appName = name
	}
}

// WithCodec sets the codec to use to encode and
// decode messages.
func WithCodec(codec msg.Codec) Option {
	return func(c *config) {
		c.codec = codec
	}
}

// WithEndpointDialer defines the dialer for endpoints.
func WithEndpointDialer(dialer EndpointDialer) Option {
	return func(c *config) {
		c.dialer = dialer
	}
}

// WithIdleTimeout configures the maximum idle time
// to allow for a connection.
func WithIdleTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.idleTimeout = timeout
	}
}

// WithLifeTimeout configures the maximum life of a
// connection.
func WithLifeTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.lifeTimeout = timeout
	}
}
