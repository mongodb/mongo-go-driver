package conn

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/10gen/mongo-go-driver/internal"
	"github.com/10gen/mongo-go-driver/model"
	"github.com/10gen/mongo-go-driver/msg"
	"github.com/10gen/mongo-go-driver/msg/compress"

	"github.com/10gen/mongo-go-driver/bson"
)

var globalClientConnectionID int32

func nextClientConnectionID() int32 {
	return atomic.AddInt32(&globalClientConnectionID, 1)
}

// Opener opens a connection.
type Opener func(context.Context, model.Addr, ...Option) (Connection, error)

// New opens a connection to a server.
func New(ctx context.Context, addr model.Addr, opts ...Option) (Connection, error) {

	cfg, err := newConfig(opts...)
	if err != nil {
		return nil, err
	}

	dialer := net.Dialer{
		Timeout:   cfg.connectTimeout,
		KeepAlive: cfg.keepAlive,
	}

	netConn, err := cfg.dialer(ctx, &dialer, addr.Network(), addr.String())
	if err != nil {
		return nil, err
	}

	var lifetimeDeadline time.Time
	if cfg.lifeTimeout > 0 {
		lifetimeDeadline = time.Now().Add(cfg.lifeTimeout)
	}

	id := fmt.Sprintf("%s[-%d]", addr, nextClientConnectionID())

	c := &connImpl{
		id:    id,
		cfg:   cfg,
		codec: compress.NewCodec(cfg.codec, cfg.compressors...),
		model: &model.Conn{
			ID: id,
			Server: model.Server{
				Addr: addr,
			},
		},
		addr:             addr,
		rw:               netConn,
		lifetimeDeadline: lifetimeDeadline,
	}

	c.bumpIdleDeadline()

	err = c.initialize(ctx, cfg.appName)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Connection is responsible for reading and writing messages.
type Connection interface {
	// Alive indicates if the connection is still alive.
	Alive() bool
	// Close closes the connection.
	Close() error
	// MarkDead forces a connection to close.
	MarkDead()
	// Model gets a description of the connection.
	Model() *model.Conn
	// Expired indicates if the connection has expired.
	Expired() bool
	// Read reads a message from the connection.
	Read(context.Context, int32) (msg.Response, error)
	// Write writes a number of messages to the connection.
	Write(context.Context, ...msg.Request) error
}

// Error represents an error that in the connection package.
type Error struct {
	ConnectionID string

	message string
	inner   error
}

// Message gets the basic error message.
func (e *Error) Message() string {
	return e.message
}

// Error gets a rolled-up error message.
func (e *Error) Error() string {
	return internal.RolledUpErrorMessage(e)
}

// Inner gets the inner error if one exists.
func (e *Error) Inner() error {
	return e.inner
}

type connImpl struct {
	// if id is negative, it's the client identifier; otherwise it's the same
	// as the id the server is using.
	id               string
	cfg              *config
	codec            msg.Codec
	model            *model.Conn
	addr             model.Addr
	rw               net.Conn
	dead             bool
	idleDeadline     time.Time
	lifetimeDeadline time.Time
}

func (c *connImpl) Alive() bool {
	return !c.dead
}

func (c *connImpl) Close() error {
	c.dead = true
	err := c.rw.Close()
	if err != nil {
		return c.wrapError(err, "failed closing")
	}

	return nil
}

func (c *connImpl) MarkDead() {
	c.dead = true
}

func (c *connImpl) Model() *model.Conn {
	return c.model
}

func (c *connImpl) Expired() bool {
	now := time.Now()
	if !c.idleDeadline.IsZero() && now.After(c.idleDeadline) {
		return true
	}
	if !c.lifetimeDeadline.IsZero() && now.After(c.lifetimeDeadline) {
		return true
	}

	return c.dead
}

func (c *connImpl) Read(ctx context.Context, responseTo int32) (msg.Response, error) {
	if c.dead {
		return nil, &Error{
			ConnectionID: c.id,
			message:      "connection is dead",
		}
	}

	select {
	case <-ctx.Done():
		// we need to close here because we don't
		// know if there is a message sitting on the wire
		// unread.
		c.Close()
		return nil, c.wrapError(ctx.Err(), "failed to read")
	default:
	}

	// first set deadline based on the read timeout.
	deadline := time.Time{}
	if c.cfg.readTimeout != 0 {
		deadline = time.Now().Add(c.cfg.readTimeout)
	}

	// second, if the ctxDeadline is before the read timeout's deadline, then use it instead.
	if ctxDeadline, ok := ctx.Deadline(); ok && (deadline.IsZero() || ctxDeadline.Before(deadline)) {
		deadline = ctxDeadline
	}

	if err := c.rw.SetReadDeadline(deadline); err != nil {
		return nil, c.wrapError(err, "failed to set read deadline")
	}

	message, err := c.codec.Decode(c.rw)
	if err != nil {
		c.Close()
		return nil, c.wrapError(err, "failed reading")
	}

	resp, ok := message.(msg.Response)
	if !ok {
		return nil, c.wrapError(err, "failed reading: invalid message type received")
	}

	if resp.ResponseTo() != responseTo {
		return nil, &Error{
			ConnectionID: c.id,
			message:      fmt.Sprintf("received out of order response: expected %d, but got %d", responseTo, resp.ResponseTo()),
		}
	}

	c.bumpIdleDeadline()
	return resp, nil
}

func (c *connImpl) String() string {
	return c.id
}

func (c *connImpl) Write(ctx context.Context, requests ...msg.Request) error {
	if c.dead {
		return &Error{
			ConnectionID: c.id,
			message:      "connection is dead",
		}
	}

	select {
	case <-ctx.Done():
		return c.wrapError(ctx.Err(), "failed to write")
	default:
	}

	// first set deadline based on the write timeout.
	deadline := time.Time{}
	if c.cfg.writeTimeout != 0 {
		deadline = time.Now().Add(c.cfg.writeTimeout)
	}

	// second, if the ctxDeadline is before the read timeout's deadline, then use it instead.
	if ctxDeadline, ok := ctx.Deadline(); ok && (deadline.IsZero() || ctxDeadline.Before(deadline)) {
		deadline = ctxDeadline
	}

	if err := c.rw.SetWriteDeadline(deadline); err != nil {
		return c.wrapError(err, "failed to set write deadline")
	}

	var messages []msg.Message
	for _, message := range requests {
		messages = append(messages, message)
	}

	err := c.codec.Encode(c.rw, messages...)
	if err != nil {
		c.Close()
		return c.wrapError(err, "failed writing")
	}

	c.bumpIdleDeadline()
	return nil
}

func (c *connImpl) bumpIdleDeadline() {
	if c.cfg.idleTimeout > 0 {
		c.idleDeadline = time.Now().Add(c.cfg.idleTimeout)
	}
}

func (c *connImpl) describeServer(ctx context.Context, clientDoc bson.M) (*internal.IsMasterResult, *internal.BuildInfoResult, error) {

	isMasterCmd := bson.D{
		{Name: "ismaster", Value: 1},
	}
	if len(c.cfg.compressors) > 0 {
		var names []string
		for _, compressor := range c.cfg.compressors {
			names = append(names, compressor.Name())
		}
		isMasterCmd = append(isMasterCmd, bson.DocElem{
			Name:  "compression",
			Value: names,
		})
	}
	if clientDoc != nil {
		isMasterCmd = append(isMasterCmd, bson.DocElem{
			Name:  "client",
			Value: clientDoc,
		})
	}

	isMasterReq := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		true,
		isMasterCmd,
	)
	buildInfoReq := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		true,
		bson.D{{Name: "buildInfo", Value: 1}},
	)

	var isMasterResult internal.IsMasterResult
	var buildInfoResult internal.BuildInfoResult
	err := ExecuteCommands(ctx, c, []msg.Request{isMasterReq, buildInfoReq}, []interface{}{&isMasterResult, &buildInfoResult})
	if err != nil {
		return nil, nil, err
	}

	if len(isMasterResult.Compression) > 0 {
		compCodec := c.codec.(*compress.Codec)
		compCodec.SetCompressors(isMasterResult.Compression)
	} else {
		c.codec = c.cfg.codec
	}

	return &isMasterResult, &buildInfoResult, nil
}

func (c *connImpl) initialize(ctx context.Context, appName string) error {

	isMasterResult, buildInfoResult, err := c.describeServer(ctx, createClientDoc(appName))
	if err != nil {
		return err
	}

	getLastErrorReq := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		true,
		bson.D{{Name: "getLastError", Value: 1}},
	)

	c.model = &model.Conn{
		ID:     c.id,
		Server: *model.BuildServer(c.addr, isMasterResult, buildInfoResult),
	}

	var getLastErrorResult internal.GetLastErrorResult
	err = ExecuteCommand(ctx, c, getLastErrorReq, &getLastErrorResult)
	// NOTE: we don't care about this result. If it fails, it doesn't
	// harm us in any way other than not being able to correlate
	// our logs with the server's logs.
	if err == nil {
		c.id = fmt.Sprintf("%s[%d]", c.addr, getLastErrorResult.ConnectionID)
		c.model.ID = c.id
	}

	return nil
}

func (c *connImpl) wrapError(inner error, message string) error {
	return &Error{
		c.id,
		fmt.Sprintf("connection(%s) error: %s", c.id, message),
		inner,
	}
}

func createClientDoc(appName string) bson.M {
	clientDoc := bson.M{
		"driver": bson.M{
			"name":    "mongo-go-driver",
			"version": internal.Version,
		},
		"os": bson.M{
			"type":         runtime.GOOS,
			"architecture": runtime.GOARCH,
		},
		"platform": runtime.Version(),
	}
	if appName != "" {
		clientDoc["application"] = bson.M{"name": appName}
	}

	return clientDoc
}
