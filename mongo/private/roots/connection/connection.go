// Package connection contains the types for building and pooling connections that can speak the
// MongoDB Wire Protocol. Since this low level library is meant to be used in the context of either
// a driver or a server there are some extra identifiers on a connection so one can keep track of
// what a connection is. This package purposefully hides the underlying network and abstracts the
// writing to and reading from a connection to wireops.Op's. This package also provides types for
// listening for and accepting Connections, as well as some types for handling connections and
// proxying connections to another server.
package connection

import (
	"context"
	"net"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Connection is used to read and write wire protocol messages to a network.
type Connection interface {
	WriteWireMessage(context.Context, wiremessage.WireMessage) error
	ReadWireMessage(context.Context) (wiremessage.WireMessage, error)
	Close() error
	ID() string
}

// Dialer is used to make network connections.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// DefaultDialer is the Dialer implementation that is used by this package. Changing this
// will also change the Dialer used for this package. This should only be changed why all
// of the connections being made need to use a different Dialer. Most of the time, using a
// WithDialer option is more appropriate than changing this variable.
var DefaultDialer Dialer = &net.Dialer{}

// Configurer replaces the Opener type since no one outside of the connection package
// can do anything with a connection.Option. This is mainly useful for things like
// authenticating a Connection once it's been dialed and has gone through the
// isMaster and buildInfo steps.
type Configurer interface {
	Configure(Connection) (Connection, error)
}

type config struct{}

// Option is used to configure a connection.
type Option func(*config) error

// New opens a connection to a given Addr.
func New(context.Context, net.Addr, ...Option) (Connection, error) { return nil, nil }

// WithAppName sets the application name which gets sent to MongoDB when it
// first connects.
func WithAppName(func(string) string) Option { return nil }

// WithConnectTimeout configures the maximum amount of time a dial will wait for a
// connect to complete. The default is 30 seconds.
func WithConnectTimeout(func(time.Duration) time.Duration) Option { return nil }

// WithDialer configures the Dialer to use when making a new connection to MongoDB.
func WithDialer(func(Dialer) Dialer) Option { return nil }

// WithConfigurer configures the Configurers that will be used to configure newly
// dialed connections.
func WithConfigurer(func(Configurer) Configurer) Option { return nil }

// WithIdleTimeout configures the maximum idle time to allow for a connection.
func WithIdleTimeout(func(time.Duration) time.Duration) Option { return nil }

// WithLifeTimeout configures the maximum life of a connection.
func WithLifeTimeout(func(time.Duration) time.Duration) Option { return nil }

// WithReadTimeout configures the maximum read time for a connection.
func WithReadTimeout(func(time.Duration) time.Duration) Option { return nil }

// WithWriteTimeout configures the maximum write time for a connection.
func WithWriteTimeout(func(time.Duration) time.Duration) Option { return nil }

// WithTLSConfig configures the TLS options for a connection.
func WithTLSConfig(func(*TLSConfig) *TLSConfig) Option { return nil }
