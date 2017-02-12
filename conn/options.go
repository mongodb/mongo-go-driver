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

// AppName sets the application name which gets
// sent to MongoDB on first connection.
func AppName(name string) Option {
	return func(c *config) {
		c.appName = name
	}
}

// Codec sets the codec to use to encode and
// decode messages.
func Codec(codec msg.Codec) Option {
	return func(c *config) {
		c.codec = codec
	}
}

// EndpointDialerOpt defines the dialer for endpoints. Use this
// configuration option to enable things like TLS.
func EndpointDialerOpt(dialer EndpointDialer) Option {
	return func(c *config) {
		c.dialer = dialer
	}
}
