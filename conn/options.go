package conn

import (
	"time"

	"github.com/10gen/mongo-go-driver/msg"
)

func newConfig(opts ...Option) *config {
	cfg := &config{
		codec:          msg.NewWireProtocolCodec(),
		connectTimeout: 30 * time.Second,
		dialer:         Dial,
		idleTimeout:    10 * time.Minute,
		lifeTimeout:    30 * time.Minute,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// Option configures a connection.
type Option func(*config)

type config struct {
	appName        string
	codec          msg.Codec
	connectTimeout time.Duration
	dialer         Dialer
	idleTimeout    time.Duration
	keepAlive      time.Duration
	lifeTimeout    time.Duration
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

// WithConnectTimeout configures the maximum amount of time
// a dial will wait for a connect to complete. The default
// is 30 seconds.
func WithConnectTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.connectTimeout = timeout
	}
}

// WithDialer defines the dialer for endpoints.
func WithDialer(dialer Dialer) Option {
	return func(c *config) {
		c.dialer = dialer
	}
}

// WithWrappedDialer wraps the current dialer.
func WithWrappedDialer(wrapper func(dialer Dialer) Dialer) Option {
	return func(c *config) {
		c.dialer = wrapper(c.dialer)
	}
}

// WithIdleTimeout configures the maximum idle time
// to allow for a connection.
func WithIdleTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.idleTimeout = timeout
	}
}

// WithKeepAlive configures the the keep-alive period for
// an active network connection. If zero, keep-alives are
// not enabled.
func WithKeepAlive(keepAlive time.Duration) Option {
	return func(c *config) {
		c.keepAlive = keepAlive
	}
}

// WithLifeTimeout configures the maximum life of a
// connection.
func WithLifeTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.lifeTimeout = timeout
	}
}
