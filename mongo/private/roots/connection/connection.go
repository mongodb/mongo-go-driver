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

type Addr string

func (Addr) Network() string    { return "" }
func (Addr) String() string     { return "" }
func (Addr) Canonicalize() Addr { return Addr("") }

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

// Listener is a MongoDB Wire Protocol Connection listeners, it returns connections
// that can speak the MongoDB Wire Protocol.
type Listener interface {
	// Accept waits for and returns the next Connection to the listener.
	Accept() (Connection, error)

	// Close closes the listener.
	Close() error

	// Addr returns the listener's network address.
	Addr() Addr
}

// Listen creates a new listener on the provided network and address.
func Listen(network, address string) (Listener, error) { return nil, nil }

// Server is used to handle incoming Connections. It handles the boilerplate of accepting a
// Connection and cleaning it up after running a Handler. This also makes it easier to build
// higher level processors, like proxies, by handling the life cycle of the underlying
// connection.
type Server struct {
	Addr    Addr
	Handler Handler
}

func (*Server) ListenAndServe() error { return nil }
func (*Server) Serve(Listener) error  { return nil }

// Handler handles an individual Connection. Returning signals that the Connection
// is no longer needed and can be closed.
type Handler interface {
	HandleConnection(Connection)
}

// Proxy implements a MongoDB proxy. It will use the given pool to connect to a
// MongoDB server and proxy the traffic between connections it is given and the
// server. It will pass each of the wireops it reads from the handled connection
// to a Processor. If an error is returned from the processor, the wireop will
// not be forwarded onto the server. If there is not an error the returned message
// will be passed onto the server. If both the return message and the error are nil,
// the original wiremessage will be passed onto the server.
type Proxy struct {
	Processor wiremessage.Transformer
	Pool      Pool
}

// HandleConnection implements the Handler interface.
func (*Proxy) HandleConnection(Connection) { return }

// Pool is used to pool Connections to a server.
type Pool interface {
	Get(context.Context) (Connection, error)
	Close() error
	Drain() error
}

func NewPool(size, capacity uint64, opt ...Option) (Pool, error) { return nil, nil }

type config struct{}

type Option func(*config) error

func New(context.Context, net.Addr, ...Option) (Connection, error) { return nil, nil }

func WithAppName(func(string) string) Option                      { return nil }
func WithConnectTimeout(func(time.Duration) time.Duration) Option { return nil }
func WithDialer(func(Dialer) Dialer) Option                       { return nil }
func WithConfigurer(func(Configurer) Configurer) Option           { return nil }
func WithIdleTimeout(func(time.Duration) time.Duration) Option    { return nil }
func WithLifeTimeout(func(time.Duration) time.Duration) Option    { return nil }
func WithReadTimeout(func(time.Duration) time.Duration) Option    { return nil }
func WithWriteTimeout(func(time.Duration) time.Duration) Option   { return nil }
func WithTLSConfig(func(*TLSConfig) *TLSConfig) Option            { return nil }
