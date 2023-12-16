package mnet

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/driverutil"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/ocsp"
)

// Dialer is used to make network connections.
type Dialer driverutil.Dialer

// DefaultDialer is the Dialer implementation that is used by this package.
// Changing this will also change the Dialer used for this package. This should
// only be changed why all of the connections being made need to use a different
// Dialer. Most of the time, using a WithDialer option is more appropriate than
// changing this variable.
var DefaultDialer Dialer = driverutil.DefaultDialer

// DialerFunc is a type implemented by functions that can be used as a Dialer.
type DialerFunc driverutil.DialerFunc

// DialContext implements the Dialer interface.
func (df DialerFunc) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return df(ctx, network, address)
}

// HandshakeInformation contains information extracted from a MongoDB connection
// handshake. This is a helper type that augments description.Server by also
// tracking server connection ID and authentication-related fields. We use this
// type rather than adding authentication-related fields to description.Server
// to avoid retaining sensitive information in a user-facing type. The server
// connection ID is stored in this type because unlike description.Server, all
// handshakes are correlated with a single network connection.
type HandshakeInformation struct {
	Description             description.Server
	SpeculativeAuthenticate bsoncore.Document
	ServerConnectionID      *int64
	SaslSupportedMechs      []string
}

// Handshaker is the interface implemented by types that can perform a MongoDB
// handshake over a provided driver.Connection. This is used during connection
// initialization. Implementations must be goroutine safe.
type Handshaker interface {
	GetHandshakeInformation(context.Context, address.Address, *Connection) (HandshakeInformation, error)
	FinishHandshake(context.Context, *Connection) error
}

type Options struct {
	ConnectTimeout           time.Duration
	CommandMonitor           *event.CommandMonitor
	Compressors              []string
	Dialer                   Dialer
	DisableOCSPEndpointCheck bool
	Handshaker               Handshaker
	HTTPClient               *http.Client
	IdleTimeout              time.Duration
	LoadBalanced             bool
	OCSPCache                ocsp.Cache
	ReadTimeout              time.Duration
	TLSConfig                *tls.Config
	WriteTimeout             time.Duration
	ZlibLevel                *int
	ZstdLevel                *int

	//Dialer             Dialer
	//GenerationNumberFn GenerationNumberFn
	//LoadBalanced       bool
}

// NewUnsafeConnectionOptions converts mnet.Options to
// driverutil.NewUnsafeConnectionOptions. This function is sealed by the
// internal package and is not meant for external use.
func NewUnsafeConnectionOptions(opts *Options) driverutil.UnsafeConnectionOptions {
	unsafeOpts := driverutil.UnsafeConnectionOptions{
		ConnectTimeout: opts.ConnectTimeout,
		CommandMonitor: opts.CommandMonitor,
		IdleTimeout:    opts.IdleTimeout,
		ReadTimeout:    opts.ReadTimeout,
		WriteTimeout:   opts.WriteTimeout,
	}

	return unsafeOpts
}
