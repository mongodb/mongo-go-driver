package driverutil

import (
	"context"
	"net"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
)

// Dialer is used to make network connections.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// DefaultDialer is the Dialer implementation that is used by this package.
// Changing this will also change the Dialer used for this package. This should
// only be changed why all of the connections being made need to use a different
// Dialer. Most of the time, using a WithDialer option is more appropriate than
// changing this variable.
var DefaultDialer Dialer = &net.Dialer{}

// DialerFunc is a type implemented by functions that can be used as a Dialer.
type DialerFunc func(ctx context.Context, network, address string) (net.Conn, error)

// DialContext implements the Dialer interface.
func (df DialerFunc) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return df(ctx, network, address)
}

// GenerationNumberFn is a callback type used by a connection to fetch its generation number given its service ID.
type GenerationNumberFn func(serviceID *primitive.ObjectID) uint64

type UnsafeConnectionOptions struct {
	CommandMonitor     *event.CommandMonitor
	ConnectTimeout     time.Duration
	Dialer             Dialer
	GenerationNumberFn GenerationNumberFn
	IdleTimeout        time.Duration
	LoadBalanced       bool
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
}
