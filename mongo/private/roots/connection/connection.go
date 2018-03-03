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
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/addr"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/version"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

var globalClientConnectionID uint64

func nextClientConnectionID() uint64 {
	return atomic.AddUint64(&globalClientConnectionID, 1)
}

// Connection is used to read and write wire protocol messages to a network.
type Connection interface {
	WriteWireMessage(context.Context, wiremessage.WireMessage) error
	ReadWireMessage(context.Context) (wiremessage.WireMessage, error)
	Close() error
	Expired() bool
	Alive() bool
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

// Handshaker is the interface implemented by types that can perform a MongoDB
// handshake over a provided ReadWriter. This is used during connection
// initialization.
type Handshaker interface {
	Handshake(context.Context, addr.Addr, wiremessage.ReadWriter) (description.Server, error)
}

// HandshakerFunc is an adapter to allow the use of ordinary functions as
// connection handshakers.
type HandshakerFunc func(context.Context, addr.Addr, wiremessage.ReadWriter) (description.Server, error)

// Handshake implements the Handshaker interface.
func (hf HandshakerFunc) Handshake(ctx context.Context, address addr.Addr, rw wiremessage.ReadWriter) (description.Server, error) {
	return hf(ctx, address, rw)
}

type connection struct {
	addr             addr.Addr
	id               string
	conn             net.Conn
	dead             bool
	idleTimeout      time.Duration
	idleDeadline     time.Time
	lifetimeDeadline time.Time
	readTimeout      time.Duration
	writeTimeout     time.Duration
	readBuf          []byte
	writeBuf         []byte
}

// New opens a connection to a given Addr.
//
// The server description returned is nil if there was no handshaker provided.
func New(ctx context.Context, address addr.Addr, opts ...Option) (Connection, *description.Server, error) {
	cfg, err := newConfig(opts...)
	if err != nil {
		return nil, nil, err
	}

	nc, err := cfg.dialer.DialContext(ctx, address.Network(), address.String())
	if err != nil {
		return nil, nil, err
	}

	if cfg.tlsConfig != nil {
		tlsConfig := cfg.tlsConfig.Clone()
		if !tlsConfig.InsecureSkipVerify {
			hostname := address.String()
			colonPos := strings.LastIndex(hostname, ":")
			if colonPos == -1 {
				colonPos = len(hostname)
			}

			hostname = hostname[:colonPos]
			tlsConfig.ServerName = hostname
		}

		client := tls.Client(nc, tlsConfig.Config)

		errChan := make(chan error, 1)
		go func() {
			errChan <- client.Handshake()
		}()

		select {
		case err := <-errChan:
			if err != nil {
				return nil, nil, err
			}
		case <-ctx.Done():
			return nil, nil, errors.New("server connection cancelled/timeout during TLS handshake")
		}
		nc = client
	}

	var lifetimeDeadline time.Time
	if cfg.lifeTimeout > 0 {
		lifetimeDeadline = time.Now().Add(cfg.lifeTimeout)
	}

	id := fmt.Sprintf("%s[-%d]", address, nextClientConnectionID())

	c := &connection{
		id:               id,
		conn:             nc,
		addr:             address,
		idleTimeout:      cfg.idleTimeout,
		lifetimeDeadline: lifetimeDeadline,
		readTimeout:      cfg.readTimeout,
		writeTimeout:     cfg.writeTimeout,
		readBuf:          make([]byte, 256),
		writeBuf:         make([]byte, 0, 256),
	}

	c.bumpIdleDeadline()

	var desc *description.Server
	if cfg.handshaker != nil {
		d, err := cfg.handshaker.Handshake(ctx, c.addr, c)
		if err != nil {
			return nil, nil, err
		}
		desc = &d
	}

	return c, desc, nil
}

func (c *connection) Alive() bool {
	return !c.dead
}

func (c *connection) Expired() bool {
	now := time.Now()
	if !c.idleDeadline.IsZero() && now.After(c.idleDeadline) {
		return true
	}

	if !c.lifetimeDeadline.IsZero() && now.After(c.lifetimeDeadline) {
		return true
	}

	return c.dead
}

func (c *connection) WriteWireMessage(ctx context.Context, wm wiremessage.WireMessage) error {
	var err error
	if c.dead {
		return Error{
			ConnectionID: c.id,
			message:      "connection is dead",
		}
	}

	select {
	case <-ctx.Done():
		return Error{
			ConnectionID: c.id,
			Wrapped:      ctx.Err(),
			message:      "failed to write",
		}
	default:
	}

	deadline := time.Time{}
	if c.writeTimeout != 0 {
		deadline = time.Now().Add(c.writeTimeout)
	}

	if dl, ok := ctx.Deadline(); ok && (deadline.IsZero() || dl.Before(deadline)) {
		deadline = dl
	}

	if err := c.conn.SetWriteDeadline(deadline); err != nil {
		return Error{
			ConnectionID: c.id,
			Wrapped:      err,
			message:      "failed to set write deadline",
		}
	}

	// Truncate the write buffer
	c.writeBuf = c.writeBuf[:0]

	c.writeBuf, err = wm.AppendWireMessage(c.writeBuf)
	if err != nil {
		return Error{
			ConnectionID: c.id,
			Wrapped:      err,
			message:      "unable to encode wire message",
		}
	}

	_, err = c.conn.Write(c.writeBuf)
	if err != nil {
		c.Close()
		return Error{
			ConnectionID: c.id,
			Wrapped:      err,
			message:      "unable to write wire message to network",
		}
	}

	c.bumpIdleDeadline()
	return nil
}

func (c *connection) ReadWireMessage(ctx context.Context) (wiremessage.WireMessage, error) {
	if c.dead {
		return nil, Error{
			ConnectionID: c.id,
			message:      "connection is dead",
		}
	}

	select {
	case <-ctx.Done():
		// We close the connection because we don't know if there
		// is an unread message on the wire.
		c.Close()
		return nil, Error{
			ConnectionID: c.id,
			Wrapped:      ctx.Err(),
			message:      "failed to read",
		}
	default:
	}

	deadline := time.Time{}
	if c.readTimeout != 0 {
		deadline = time.Now().Add(c.readTimeout)
	}

	if ctxDL, ok := ctx.Deadline(); ok && (deadline.IsZero() || ctxDL.Before(deadline)) {
		deadline = ctxDL
	}

	if err := c.conn.SetReadDeadline(deadline); err != nil {
		return nil, Error{
			ConnectionID: c.id,
			Wrapped:      ctx.Err(),
			message:      "failed to set read deadline",
		}
	}

	var sizeBuf [4]byte
	_, err := io.ReadFull(c.conn, sizeBuf[:])
	if err != nil {
		defer c.Close()
		return nil, Error{
			ConnectionID: c.id,
			Wrapped:      err,
			message:      "unable to decode message length",
		}
	}

	size := readInt32(sizeBuf[:], 0)

	// Isn't the best reuse, but resizing a []byte to be larger
	// is difficult.
	if len(c.readBuf) > int(size) {
		c.readBuf = c.readBuf[:size]
	} else {
		c.readBuf = make([]byte, size)
	}

	c.readBuf[0], c.readBuf[1], c.readBuf[2], c.readBuf[3] = sizeBuf[0], sizeBuf[1], sizeBuf[2], sizeBuf[3]

	_, err = io.ReadFull(c.conn, c.readBuf[4:])
	if err != nil {
		defer c.Close()
		return nil, Error{
			ConnectionID: c.id,
			Wrapped:      err,
			message:      "unable to read full message",
		}
	}

	hdr, err := wiremessage.ReadHeader(c.readBuf, 0)
	if err != nil {
		defer c.Close()
		return nil, Error{
			ConnectionID: c.id,
			Wrapped:      err,
			message:      "unable to decode header",
		}
	}

	var wm wiremessage.WireMessage
	switch hdr.OpCode {
	case wiremessage.OpReply:
		var r wiremessage.Reply
		err := r.UnmarshalWireMessage(c.readBuf)
		if err != nil {
			defer c.Close()
			return nil, Error{
				ConnectionID: c.id,
				Wrapped:      err,
				message:      "unable to decode OP_REPLY",
			}
		}
		wm = r
	default:
		defer c.Close()
		return nil, Error{
			ConnectionID: c.id,
			message:      fmt.Sprintf("opcode %s not implemented", hdr.OpCode),
		}
	}

	c.bumpIdleDeadline()
	return wm, nil
}

func (c *connection) bumpIdleDeadline() {
	if c.idleTimeout > 0 {
		c.idleDeadline = time.Now().Add(c.idleTimeout)
	}
}

func (c *connection) Close() error {
	c.dead = true
	err := c.conn.Close()
	if err != nil {
		return Error{
			ConnectionID: c.id,
			Wrapped:      err,
			message:      "failed to close net.Conn",
		}
	}

	return nil
}

func (c *connection) ID() string {
	return c.id
}

func (c *connection) initialize(ctx context.Context, appName string) error {
	return nil
}

func clientDoc(app string) *bson.Document {
	doc := bson.NewDocument(
		bson.EC.SubDocumentFromElements(
			"driver",
			bson.EC.String("name", "mongo-go-driver"),
			bson.EC.String("version", version.Driver),
		),
		bson.EC.SubDocumentFromElements(
			"os",
			bson.EC.String("type", runtime.GOOS),
			bson.EC.String("architecture", runtime.GOARCH),
		),
		bson.EC.String("platform", runtime.Version()))

	if app != "" {
		doc.Append(bson.EC.SubDocumentFromElements(
			"application",
			bson.EC.String("name", app),
		))
	}

	return doc
}

func readInt32(b []byte, pos int32) int32 {
	return (int32(b[pos+0])) | (int32(b[pos+1]) << 8) | (int32(b[pos+2]) << 16) | (int32(b[pos+3]) << 24)
}
