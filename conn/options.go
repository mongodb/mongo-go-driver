package conn

import "github.com/10gen/mongo-go-driver/msg"

func newConfig(opts ...Option) *config {
	cfg := &config{
		codec:  msg.NewWireProtocolCodec(),
		dialer: DialEndpoint,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// Option configures a connection.
type Option func(*config)

type config struct {
	appName string
	codec   msg.Codec
	dialer  EndpointDialer
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
