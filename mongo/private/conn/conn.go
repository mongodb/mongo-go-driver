// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package conn

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo/internal"
	"github.com/mongodb/mongo-go-driver/mongo/model"
	"github.com/mongodb/mongo-go-driver/mongo/private/msg"

	"github.com/mongodb/mongo-go-driver/bson"
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

	dialer := net.Dialer{Timeout: cfg.connectTimeout}

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
		codec: cfg.codec,
		model: &model.Conn{
			ID: id,
			Server: model.Server{
				Addr: addr,
			},
		},
		addr:             addr,
		rw:               netConn,
		lifetimeDeadline: lifetimeDeadline,
		idleTimeout:      cfg.idleTimeout,
		readTimeout:      cfg.readTimeout,
		writeTimeout:     cfg.writeTimeout,
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
	// CloseIgnoreError closes the connection and ignores any error that occurs.
	CloseIgnoreError()
	// MarkDead forces a connection to close.
	MarkDead()
	// Model gets a description of the connection.
	Model() *model.Conn
	// Expired indicates if the connection has expired.
	Expired() bool
	// LocalAddr returns the local address of the connection.
	LocalAddr() net.Addr
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
	codec            msg.Codec
	model            *model.Conn
	addr             model.Addr
	rw               net.Conn
	dead             bool
	idleTimeout      time.Duration
	idleDeadline     time.Time
	lifetimeDeadline time.Time
	readTimeout      time.Duration
	writeTimeout     time.Duration
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

func (c *connImpl) CloseIgnoreError() {
	_ = c.Close()
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

// LocalAddr returns the local address of a connection.
func (c *connImpl) LocalAddr() net.Addr {
	return c.rw.LocalAddr()
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
		c.CloseIgnoreError()
		c.dead = true
		return nil, c.wrapError(ctx.Err(), "failed to read")
	default:
	}

	// first set deadline based on the read timeout.
	deadline := time.Time{}
	if c.readTimeout != 0 {
		deadline = time.Now().Add(c.readTimeout)
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
		c.CloseIgnoreError()
		c.dead = true
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
	if c.writeTimeout != 0 {
		deadline = time.Now().Add(c.writeTimeout)
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
		// Ignore any error that occurs since we're already returning a different one.
		_ = c.Close()
		c.dead = true
		return c.wrapError(err, "failed writing")
	}

	c.bumpIdleDeadline()
	return nil
}

func (c *connImpl) bumpIdleDeadline() {
	if c.idleTimeout > 0 {
		c.idleDeadline = time.Now().Add(c.idleTimeout)
	}
}

func (c *connImpl) describeServer(ctx context.Context, clientDoc *bson.Document) (*internal.IsMasterResult, *internal.BuildInfoResult, error) {
	isMasterCmd := bson.NewDocument(bson.EC.Int32("isMaster", 1))
	if clientDoc != nil {
		isMasterCmd.Append(bson.EC.SubDocument("client", clientDoc))
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
		bson.NewDocument(bson.EC.Int32("buildInfo", 1)),
	)

	var isMasterResult internal.IsMasterResult
	var buildInfoResult internal.BuildInfoResult
	rdrs, err := ExecuteCommands(ctx, c, []msg.Request{isMasterReq, buildInfoReq})
	if err != nil {
		return nil, nil, err
	}

	err = bson.NewDecoder(bytes.NewReader(rdrs[0])).Decode(&isMasterResult)
	if err != nil {
		return nil, nil, err
	}
	err = bson.NewDecoder(bytes.NewReader(rdrs[1])).Decode(&buildInfoResult)
	if err != nil {
		return nil, nil, err
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
		bson.NewDocument(bson.EC.Int32("getLastError", 1)),
	)

	c.model = &model.Conn{
		ID:     c.id,
		Server: *model.BuildServer(c.addr, isMasterResult, buildInfoResult),
	}

	// NOTE: we don't care about this result. If it fails, it doesn't
	// harm us in any way other than not being able to correlate
	// our logs with the server's logs.
	var getLastErrorResult internal.GetLastErrorResult
	rdr, err := ExecuteCommand(ctx, c, getLastErrorReq)
	if err != nil {
		return nil
	}

	err = bson.NewDecoder(bytes.NewReader(rdr)).Decode(&getLastErrorResult)
	if err != nil {
		return nil
	}

	c.id = fmt.Sprintf("%s[%d]", c.addr, getLastErrorResult.ConnectionID)
	c.model.ID = c.id

	return nil
}

func (c *connImpl) wrapError(inner error, message string) error {
	return &Error{
		c.id,
		fmt.Sprintf("connection(%s) error: %s", c.id, message),
		inner,
	}
}

func createClientDoc(appName string) *bson.Document {
	doc := bson.NewDocument(
		bson.EC.SubDocumentFromElements(
			"driver",
			bson.EC.String("name", "mongo-go-driver"),
			bson.EC.String("version", internal.Version),
		),
		bson.EC.SubDocumentFromElements(
			"os",
			bson.EC.String("type", runtime.GOOS),
			bson.EC.String("architecture", runtime.GOARCH),
		),
		bson.EC.String("platform", runtime.Version()))

	if appName != "" {
		doc.Append(bson.EC.SubDocumentFromElements(
			"application",
			bson.EC.String("name", appName),
		))
	}

	return doc
}
