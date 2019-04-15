package driver // import "go.mongodb.org/mongo-driver/x/mongo/driver"

import (
	"context"

	"go.mongodb.org/mongo-driver/x/network/address"
	"go.mongodb.org/mongo-driver/x/network/description"
)

// Deployment is implemented by types that can select a server from a deployment.
type Deployment interface {
	SelectServer(context.Context, description.ServerSelector) (Server, error)
	Description() description.Topology
}

// Server represents a MongoDB server. Implementations should pool connections and handle the
// retrieving and returning of connections.
type Server interface {
	Connection(context.Context) (Connection, error)
}

// Connection represents a connection to a MongoDB server.
type Connection interface {
	WriteWireMessage(context.Context, []byte) error
	ReadWireMessage(ctx context.Context, dst []byte) ([]byte, error)
	Description() description.Server
	Close() error
	ID() string
	Address() address.Address
}

// ErrorProcessor implementations can handle processing errors, which may modify their internal state.
// If this type is implemented by a Server, then Operation.Execute will call it's ProcessError
// method after it decodes a wire message.
type ErrorProcessor interface {
	ProcessError(error)
}

// Handshaker is the interface implemented by types that can perform a MongoDB
// handshake over a provided driver.Connection. This is used during connection
// initialization. Implementations must be goroutine safe.
type Handshaker interface {
	Handshake(context.Context, address.Address, Connection) (description.Server, error)
}

// HandshakerFunc is an adapter to allow the use of ordinary functions as
// connection handshakers.
type HandshakerFunc func(context.Context, address.Address, Connection) (description.Server, error)

// Handshake implements the Handshaker interface.
func (hf HandshakerFunc) Handshake(ctx context.Context, addr address.Address, conn Connection) (description.Server, error) {
	return hf(ctx, addr, conn)
}

// TODO(GODRUVER-617): We can likely use 1 type for both the RetryType and the RetryMode by using
// 2 bits for the mode and 1 bit for the type. Although in the practical sense, we might not want to
// do that since the type of retryability is tied to the operation itself and isn't going change,
// e.g. and insert operation will always be a write, however some operations are both reads and
// writes, for instance aggregate is a read but with a $out parameter it's a write.

// RetryType specifies whether a retry is a read, write, or disabled.
type RetryType uint

// THese are the availables types of retry.
const (
	_ RetryType = iota
	RetryWrite
	RetryRead
)

// RetryMode specifies the way that retries are handled for retryable operations.
type RetryMode uint

// These are the modes available for retrying.
const (
	// RetryNone disables retrying.
	RetryNone RetryMode = iota
	// RetryOnce will enable retrying the entire operation once.
	RetryOnce
	// RetryOncePerCommand will enable retrying each command associated with an operation. For
	// example, if an insert is batch split into 4 commands then each of those commands is eligible
	// for one retry.
	RetryOncePerCommand
	// RetryContext will enable retrying until the context.Context's deadline is exceeded or it is
	// cancelled.
	RetryContext
)

// Enabled returns if this RetryMode enables retrying.
func (rm RetryMode) Enabled() bool {
	return rm == RetryOnce || rm == RetryOncePerCommand || rm == RetryContext
}
