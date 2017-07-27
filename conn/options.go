package conn

import (
	"time"

	"github.com/10gen/mongo-go-driver/msg"
	"github.com/10gen/mongo-go-driver/msg/compress"
)

func newConfig(opts ...Option) (*config, error) {
	cfg := &config{
		codec:          msg.NewWireProtocolCodec(),
		compressors:    nil,
		connectTimeout: 30 * time.Second,
		dialer:         dial,
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

// Option configures a connection.
type Option func(*config) error

type config struct {
	appName        string
	codec          msg.Codec
	compressors    []compress.Compressor
	connectTimeout time.Duration
	dialer         Dialer
	idleTimeout    time.Duration
	keepAlive      time.Duration
	lifeTimeout    time.Duration
	readTimeout    time.Duration
	writeTimeout   time.Duration
}

// WithAppName sets the application name which gets
// sent to MongoDB on first connection.
func WithAppName(name string) Option {
	return func(c *config) error {
		c.appName = name
		return nil
	}
}

// WithCodec sets the codec to use to encode and
// decode messages.
func WithCodec(codec msg.Codec) Option {
	return func(c *config) error {
		c.codec = codec
		return nil
	}
}

// WithCompressors sets the supported compressors.
func WithCompressors(compressors ...compress.Compressor) Option {
	return func(c *config) error {
		c.compressors = compressors
		return nil
	}
}

// WithConnectTimeout configures the maximum amount of time
// a dial will wait for a connect to complete. The default
// is 30 seconds.
func WithConnectTimeout(timeout time.Duration) Option {
	return func(c *config) error {
		c.connectTimeout = timeout
		return nil
	}
}

// WithDialer defines the dialer for endpoints.
func WithDialer(dialer Dialer) Option {
	return func(c *config) error {
		c.dialer = dialer
		return nil
	}
}

// WithWrappedDialer wraps the current dialer.
func WithWrappedDialer(wrapper func(dialer Dialer) Dialer) Option {
	return func(c *config) error {
		c.dialer = wrapper(c.dialer)
		return nil
	}
}

// WithIdleTimeout configures the maximum idle time
// to allow for a connection.
func WithIdleTimeout(timeout time.Duration) Option {
	return func(c *config) error {
		c.idleTimeout = timeout
		return nil
	}
}

// WithKeepAlive configures the the keep-alive period for
// an active network connection. If zero, keep-alives are
// not enabled.
func WithKeepAlive(keepAlive time.Duration) Option {
	return func(c *config) error {
		c.keepAlive = keepAlive
		return nil
	}
}

// WithLifeTimeout configures the maximum life of a
// connection.
func WithLifeTimeout(timeout time.Duration) Option {
	return func(c *config) error {
		c.lifeTimeout = timeout
		return nil
	}
}

// WithReadTimeout configures the maximum read time
// for a connection.
func WithReadTimeout(timeout time.Duration) Option {
	return func(c *config) error {
		c.readTimeout = timeout
		return nil
	}
}

// WithWriteTimeout configures the maximum read time
// for a connection.
func WithWriteTimeout(timeout time.Duration) Option {
	return func(c *config) error {
		c.writeTimeout = timeout
		return nil
	}
}
