// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topologyx

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"strings"

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/x/mongo/driverx"
	"go.mongodb.org/mongo-driver/x/network/address"
	"go.mongodb.org/mongo-driver/x/network/command"
	"go.mongodb.org/mongo-driver/x/network/compressor"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
)

var notMasterCodes = []int32{10107, 13435}
var recoveringCodes = []int32{11600, 11602, 13436, 189, 91}
var globalClientConnectionID uint64

func nextClientConnectionID() uint64 {
	return atomic.AddUint64(&globalClientConnectionID, 1)
}

// Dialer is used to make network connections.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// DialerFunc is a type implemented by functions that can be used as a Dialer.
type DialerFunc func(ctx context.Context, network, address string) (net.Conn, error)

// DialContext implements the Dialer interface.
func (df DialerFunc) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return df(ctx, network, address)
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
	Handshake(context.Context, address.Address, driverx.Connection) (description.Server, error)
}

// HandshakerFunc is an adapter to allow the use of ordinary functions as
// connection handshakers.
type HandshakerFunc func(context.Context, address.Address, driverx.Connection) (description.Server, error)

// Handshake implements the Handshaker interface.
func (hf HandshakerFunc) Handshake(ctx context.Context, addr address.Address, conn driverx.Connection) (description.Server, error) {
	return hf(ctx, addr, conn)
}

// ConnectionError represents a connection error.
type ConnectionError struct {
	ConnectionID string
	Wrapped      error

	message string
}

// Error implements the error interface.
func (e ConnectionError) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("connection(%s) %s: %s", e.ConnectionID, e.message, e.Wrapped.Error())
	}
	return fmt.Sprintf("connection(%s) %s", e.ConnectionID, e.message)
}

// NetworkError represents an error that occurred while reading from or writing
// to a network socket.
type NetworkError struct {
	ConnectionID string
	Wrapped      error
}

func (ne NetworkError) Error() string {
	return fmt.Sprintf("connection(%s): %s", ne.ConnectionID, ne.Wrapped.Error())
}

type connection struct {
	nc          net.Conn
	desc        description.Server
	addr        address.Address
	id          string
	compressBuf []byte                // buffer to compress messages
	compressor  compressor.Compressor // use for compressing messages
	// server can compress response with any compressor supported by driver
	compressorMap    map[wiremessage.CompressorID]compressor.Compressor
	commandMap       map[int64]*commandMetadata // map for monitoring commands sent to server
	dead             bool
	idleTimeout      time.Duration
	idleDeadline     time.Time
	lifetimeDeadline time.Time
	cmdMonitor       *event.CommandMonitor
	readTimeout      time.Duration
	uncompressBuf    []byte // buffer to uncompress messages
	writeTimeout     time.Duration
	readBuf          []byte
	writeBuf         []byte
	wireMessageBuf   []byte // buffer to store uncompressed wire message before compressing

	p          *pool
	poolID     uint64 // The ID of this connection in the pool
	generation uint64
	closed     int32
}

// newConnection opens a connection to a given Addr.
//
// TODO(GODRIVER-617): Should this be a method on pool instead of having pool as a parameter?
func newConnection(ctx context.Context, p *pool, addr address.Address, opts ...ConnectionOption) (*connection, error) {
	cfg, err := newConnectionConfig(opts...)
	if err != nil {
		return nil, err
	}

	nc, err := cfg.dialer.DialContext(ctx, addr.Network(), addr.String())
	if err != nil {
		return nil, err
	}

	if cfg.tlsConfig != nil {
		tlsConfig := cfg.tlsConfig.Clone()
		nc, err = configureTLS(ctx, nc, addr, tlsConfig)
		if err != nil {
			return nil, err
		}
	}

	var lifetimeDeadline time.Time
	if cfg.lifeTimeout > 0 {
		lifetimeDeadline = time.Now().Add(cfg.lifeTimeout)
	}

	id := fmt.Sprintf("%s[-%d]", addr, nextClientConnectionID())
	compressorMap := make(map[wiremessage.CompressorID]compressor.Compressor)

	for _, comp := range cfg.compressors {
		compressorMap[comp.CompressorID()] = comp
	}

	c := &connection{
		id:               id,
		nc:               nc,
		compressBuf:      make([]byte, 256),
		compressorMap:    compressorMap,
		commandMap:       make(map[int64]*commandMetadata),
		addr:             addr,
		idleTimeout:      cfg.idleTimeout,
		lifetimeDeadline: lifetimeDeadline,
		readTimeout:      cfg.readTimeout,
		writeTimeout:     cfg.writeTimeout,
		readBuf:          make([]byte, 256),
		uncompressBuf:    make([]byte, 256),
		writeBuf:         make([]byte, 0, 256),
		wireMessageBuf:   make([]byte, 256),

		// pool specific stuff
		p:          p,
		generation: atomic.LoadUint64(&p.generation),
		poolID:     atomic.AddUint64(&p.nextid, 1),
	}

	c.bumpIdleDeadline()

	if cfg.handshaker != nil {
		c.desc, err = cfg.handshaker.Handshake(ctx, c.addr, c)
		if err != nil {
			return nil, err
		}

		if len(c.desc.Compression) > 0 {
		clientMethodLoop:
			for _, comp := range cfg.compressors {
				method := comp.Name()

				for _, serverMethod := range c.desc.Compression {
					if method != serverMethod {
						continue
					}

					c.compressor = comp // found matching compressor
					break clientMethodLoop
				}
			}

		}
	}

	c.cmdMonitor = cfg.cmdMonitor // attach the command monitor later to avoid monitoring auth
	return c, nil
}

func (c *connection) WriteWireMessage(ctx context.Context, wm []byte) error {
	var err error
	if c.dead {
		return ConnectionError{
			ConnectionID: c.id,
			message:      "connection is dead",
		}
	}

	select {
	case <-ctx.Done():
		return ConnectionError{
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

	if err := c.nc.SetWriteDeadline(deadline); err != nil {
		return ConnectionError{
			ConnectionID: c.id,
			Wrapped:      err,
			message:      "failed to set write deadline",
		}
	}

	messageToWrite := wm
	// Compress if possible
	//
	// TODO(GODRIVER-617): Do we still want to do compression here? Should compression be the
	// responsibility of the caller?
	if c.compressor != nil {
		compressed, err := c.compressMessage(wm)
		if err != nil {
			return ConnectionError{
				ConnectionID: c.id,
				Wrapped:      err,
				message:      "unable to encode wire message",
			}
		}
		messageToWrite = compressed
	}

	_, err = c.nc.Write(messageToWrite)
	if err != nil {
		c.p.closeConnection(c)
		c.dead = true
		// TODO(GODRIVER-617): This should probably be a NetworkError.
		return ConnectionError{
			ConnectionID: c.id,
			Wrapped:      err,
			message:      "unable to write wire message to network",
		}
	}

	c.bumpIdleDeadline()
	err = c.commandStartedEvent(ctx, wm)
	if err != nil {
		return err
	}
	return nil
}

func (c *connection) ReadWireMessage(ctx context.Context, dst []byte) ([]byte, error) {
	if c.dead {
		return nil, ConnectionError{
			ConnectionID: c.id,
			message:      "connection is dead",
		}
	}

	select {
	case <-ctx.Done():
		// We close the connection because we don't know if there
		// is an unread message on the wire.
		c.p.closeConnection(c)
		c.dead = true
		return nil, ConnectionError{
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

	if err := c.nc.SetReadDeadline(deadline); err != nil {
		return nil, ConnectionError{
			ConnectionID: c.id,
			Wrapped:      ctx.Err(),
			message:      "failed to set read deadline",
		}
	}

	var sizeBuf [4]byte
	_, err := io.ReadFull(c.nc, sizeBuf[:])
	if err != nil {
		c.p.closeConnection(c)
		c.dead = true
		return nil, ConnectionError{
			ConnectionID: c.id,
			Wrapped:      err,
			message:      "unable to decode message length",
		}
	}

	size := readInt32(sizeBuf[:], 0)

	// TODO(GODRIVER-617): We should be writing directly to dst here.
	if cap(c.readBuf) > int(size) {
		c.readBuf = c.readBuf[:size]
	} else {
		c.readBuf = make([]byte, size)
	}

	c.readBuf[0], c.readBuf[1], c.readBuf[2], c.readBuf[3] = sizeBuf[0], sizeBuf[1], sizeBuf[2], sizeBuf[3]

	_, err = io.ReadFull(c.nc, c.readBuf[4:])
	if err != nil {
		c.p.closeConnection(c)
		c.dead = true
		return nil, ConnectionError{
			ConnectionID: c.id,
			Wrapped:      err,
			message:      "unable to read full message",
		}
	}

	wm := append(dst, c.readBuf...)

	// hdr, err := wiremessage.ReadHeader(c.readBuf, 0)
	// if err != nil {
	// 	c.Close()
	// 	return nil, ConnectionError{
	// 		ConnectionID: c.id,
	// 		Wrapped:      err,
	// 		message:      "unable to decode header",
	// 	}
	// }
	//
	// messageToDecode := c.readBuf
	// opcodeToCheck := hdr.OpCode
	//
	// if hdr.OpCode == wiremessage.OpCompressed {
	// 	var compressed wiremessage.Compressed
	// 	err := compressed.UnmarshalWireMessage(c.readBuf)
	// 	if err != nil {
	// 		defer c.Close()
	// 		return nil, Error{
	// 			ConnectionID: c.id,
	// 			Wrapped:      err,
	// 			message:      "unable to decode OP_COMPRESSED",
	// 		}
	// 	}
	//
	// 	uncompressed, origOpcode, err := c.uncompressMessage(compressed)
	// 	if err != nil {
	// 		defer c.Close()
	// 		return nil, Error{
	// 			ConnectionID: c.id,
	// 			Wrapped:      err,
	// 			message:      "unable to uncompress message",
	// 		}
	// 	}
	// 	messageToDecode = uncompressed
	// 	opcodeToCheck = origOpcode
	// }
	//
	// var wm wiremessage.WireMessage
	// switch opcodeToCheck {
	// case wiremessage.OpReply:
	// 	var r wiremessage.Reply
	// 	err := r.UnmarshalWireMessage(messageToDecode)
	// 	if err != nil {
	// 		c.Close()
	// 		return nil, Error{
	// 			ConnectionID: c.id,
	// 			Wrapped:      err,
	// 			message:      "unable to decode OP_REPLY",
	// 		}
	// 	}
	// 	wm = r
	// case wiremessage.OpMsg:
	// 	var reply wiremessage.Msg
	// 	err := reply.UnmarshalWireMessage(messageToDecode)
	// 	if err != nil {
	// 		c.Close()
	// 		return nil, Error{
	// 			ConnectionID: c.id,
	// 			Wrapped:      err,
	// 			message:      "unable to decode OP_MSG",
	// 		}
	// 	}
	// 	wm = reply
	// default:
	// 	c.Close()
	// 	return nil, Error{
	// 		ConnectionID: c.id,
	// 		message:      fmt.Sprintf("opcode %s not implemented", hdr.OpCode),
	// 	}
	// }

	c.bumpIdleDeadline()
	err = c.commandFinishedEvent(ctx, wm)
	if err != nil {
		return nil, err // TODO: do we care if monitoring fails?
	}

	return wm, nil
}

func (c *connection) Description() description.Server {
	return c.desc
}

func (c *connection) ID() string {
	return c.id
}

func (c *connection) Close() error {
	return c.p.returnConnection(c)
}

func (c *connection) Expired() bool {
	return c.dead || c.p.isExpired(c.generation)
	// return pc.Connection.Expired() || pc.p.isExpired(pc.generation)
}

func (c *connection) bumpIdleDeadline() {
	if c.idleTimeout > 0 {
		c.idleDeadline = time.Now().Add(c.idleTimeout)
	}
}

func (c *connection) compressMessage(wm []byte) ([]byte, error) {
	return wm, nil
	// var requestID int32
	// var responseTo int32
	// var origOpcode wiremessage.OpCode
	//
	// switch converted := wm.(type) {
	// case wiremessage.Query:
	// 	firstElem, err := converted.Query.IndexErr(0)
	// 	if err != nil {
	// 		return wiremessage.Compressed{}, err
	// 	}
	//
	// 	key := firstElem.Key()
	// 	if !canCompress(key) {
	// 		return wm, nil // return original message because this command can't be compressed
	// 	}
	// 	requestID = converted.MsgHeader.RequestID
	// 	origOpcode = wiremessage.OpQuery
	// 	responseTo = converted.MsgHeader.ResponseTo
	// case wiremessage.Msg:
	// 	firstElem, err := converted.Sections[0].(wiremessage.SectionBody).Document.IndexErr(0)
	// 	if err != nil {
	// 		return wiremessage.Compressed{}, err
	// 	}
	//
	// 	key := firstElem.Key()
	// 	if !canCompress(key) {
	// 		return wm, nil
	// 	}
	//
	// 	requestID = converted.MsgHeader.RequestID
	// 	origOpcode = wiremessage.OpMsg
	// 	responseTo = converted.MsgHeader.ResponseTo
	// }
	//
	// // can compress
	// c.wireMessageBuf = c.wireMessageBuf[:0] // truncate
	// var err error
	// c.wireMessageBuf, err = wm.AppendWireMessage(c.wireMessageBuf)
	// if err != nil {
	// 	return wiremessage.Compressed{}, err
	// }
	//
	// c.wireMessageBuf = c.wireMessageBuf[16:] // strip header
	// c.compressBuf = c.compressBuf[:0]
	// compressedBytes, err := c.compressor.CompressBytes(c.wireMessageBuf, c.compressBuf)
	// if err != nil {
	// 	return wiremessage.Compressed{}, err
	// }
	//
	// compressedMessage := wiremessage.Compressed{
	// 	MsgHeader: wiremessage.Header{
	// 		// MessageLength and OpCode will be set when marshalling wire message by SetDefaults()
	// 		RequestID:  requestID,
	// 		ResponseTo: responseTo,
	// 	},
	// 	OriginalOpCode:    origOpcode,
	// 	UncompressedSize:  int32(len(c.wireMessageBuf)), // length of uncompressed message excluding MsgHeader
	// 	CompressorID:      wiremessage.CompressorID(c.compressor.CompressorID()),
	// 	CompressedMessage: compressedBytes,
	// }
	//
	// return compressedMessage, nil
}

func (c *connection) commandStartedEvent(ctx context.Context, wm []byte) error {
	return nil
	// if c.cmdMonitor == nil || c.cmdMonitor.Started == nil {
	// 	return nil
	// }
	//
	// startedEvent := &event.CommandStartedEvent{
	// 	ConnectionID: c.id,
	// }
	//
	// var cmd bsonx.Doc
	// var err error
	// var legacy bool
	// var fullCollName string
	//
	// var acknowledged bool
	// switch converted := wm.(type) {
	// case wiremessage.Query:
	// 	cmd, err = converted.CommandDocument()
	// 	if err != nil {
	// 		return err
	// 	}
	//
	// 	acknowledged = converted.AcknowledgedWrite()
	// 	startedEvent.DatabaseName = converted.DatabaseName()
	// 	startedEvent.RequestID = int64(converted.MsgHeader.RequestID)
	// 	legacy = converted.Legacy()
	// 	fullCollName = converted.FullCollectionName
	// case wiremessage.Msg:
	// 	cmd, err = converted.GetMainDocument()
	// 	if err != nil {
	// 		return err
	// 	}
	//
	// 	acknowledged = converted.AcknowledgedWrite()
	// 	arr, identifier, err := converted.GetSequenceArray()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if arr != nil {
	// 		cmd = cmd.Copy() // make copy to avoid changing original command
	// 		cmd = append(cmd, bsonx.Elem{identifier, bsonx.Array(arr)})
	// 	}
	//
	// 	dbVal, err := cmd.LookupErr("$db")
	// 	if err != nil {
	// 		return err
	// 	}
	//
	// 	startedEvent.DatabaseName = dbVal.StringValue()
	// 	startedEvent.RequestID = int64(converted.MsgHeader.RequestID)
	// case wiremessage.GetMore:
	// 	cmd = converted.CommandDocument()
	// 	startedEvent.DatabaseName = converted.DatabaseName()
	// 	startedEvent.RequestID = int64(converted.MsgHeader.RequestID)
	// 	acknowledged = true
	// 	legacy = true
	// 	fullCollName = converted.FullCollectionName
	// case wiremessage.KillCursors:
	// 	cmd = converted.CommandDocument()
	// 	startedEvent.DatabaseName = converted.DatabaseName
	// 	startedEvent.RequestID = int64(converted.MsgHeader.RequestID)
	// 	legacy = true
	// }
	//
	// rawcmd, _ := cmd.MarshalBSON()
	// startedEvent.Command = rawcmd
	// startedEvent.CommandName = cmd[0].Key
	// if !canMonitor(startedEvent.CommandName) {
	// 	startedEvent.Command = emptyDoc
	// }
	//
	// c.cmdMonitor.Started(ctx, startedEvent)
	//
	// if !acknowledged {
	// 	if c.cmdMonitor.Succeeded == nil {
	// 		return nil
	// 	}
	//
	// 	// unack writes must provide a CommandSucceededEvent with an { ok: 1 } reply
	// 	finishedEvent := event.CommandFinishedEvent{
	// 		DurationNanos: 0,
	// 		CommandName:   startedEvent.CommandName,
	// 		RequestID:     startedEvent.RequestID,
	// 		ConnectionID:  c.id,
	// 	}
	//
	// 	c.cmdMonitor.Succeeded(ctx, &event.CommandSucceededEvent{
	// 		CommandFinishedEvent: finishedEvent,
	// 		Reply:                bsoncore.BuildDocument(nil, bsoncore.AppendInt32Element(nil, "ok", 1)),
	// 	})
	//
	// 	return nil
	// }
	//
	// c.commandMap[startedEvent.RequestID] = createMetadata(startedEvent.CommandName, legacy, fullCollName)
	// return nil
}

func (c *connection) commandFinishedEvent(ctx context.Context, wm []byte) error {
	return nil
	// if c.cmdMonitor == nil {
	// 	return nil
	// }
	//
	// var reply bsonx.Doc
	// var requestID int64
	// var err error
	//
	// switch converted := wm.(type) {
	// case wiremessage.Reply:
	// 	requestID = int64(converted.MsgHeader.ResponseTo)
	// case wiremessage.Msg:
	// 	requestID = int64(converted.MsgHeader.ResponseTo)
	// }
	// cmdMetadata := c.commandMap[requestID]
	// delete(c.commandMap, requestID)
	//
	// switch converted := wm.(type) {
	// case wiremessage.Reply:
	// 	if cmdMetadata.Legacy {
	// 		reply, err = converted.GetMainLegacyDocument(cmdMetadata.FullCollectionName)
	// 	} else {
	// 		reply, err = converted.GetMainDocument()
	// 	}
	// case wiremessage.Msg:
	// 	reply, err = converted.GetMainDocument()
	// }
	// if err != nil {
	// 	return err
	// }
	//
	// success, errmsg := processReply(reply)
	//
	// if (success && c.cmdMonitor.Succeeded == nil) || (!success && c.cmdMonitor.Failed == nil) {
	// 	return nil
	// }
	//
	// finishedEvent := event.CommandFinishedEvent{
	// 	DurationNanos: cmdMetadata.TimeDifference(),
	// 	CommandName:   cmdMetadata.Name,
	// 	RequestID:     requestID,
	// 	ConnectionID:  c.id,
	// }
	//
	// if success {
	// 	if !canMonitor(finishedEvent.CommandName) {
	// 		successEvent := &event.CommandSucceededEvent{
	// 			Reply:                emptyDoc,
	// 			CommandFinishedEvent: finishedEvent,
	// 		}
	// 		c.cmdMonitor.Succeeded(ctx, successEvent)
	// 		return nil
	// 	}
	//
	// 	// if response has type 1 document sequence, the sequence must be included as a BSON array in the event's reply.
	// 	if opmsg, ok := wm.(wiremessage.Msg); ok {
	// 		arr, identifier, err := opmsg.GetSequenceArray()
	// 		if err != nil {
	// 			return err
	// 		}
	// 		if arr != nil {
	// 			reply = reply.Copy() // make copy to avoid changing original command
	// 			reply = append(reply, bsonx.Elem{identifier, bsonx.Array(arr)})
	// 		}
	// 	}
	//
	// 	replyraw, _ := reply.MarshalBSON()
	// 	successEvent := &event.CommandSucceededEvent{
	// 		Reply:                replyraw,
	// 		CommandFinishedEvent: finishedEvent,
	// 	}
	//
	// 	c.cmdMonitor.Succeeded(ctx, successEvent)
	// 	return nil
	// }
	//
	// failureEvent := &event.CommandFailedEvent{
	// 	Failure:              errmsg,
	// 	CommandFinishedEvent: finishedEvent,
	// }
	//
	// c.cmdMonitor.Failed(ctx, failureEvent)
	// return nil
}

// Connection is a wrapper around a connection.Connection. This type is returned by
// a Server so that it can track network errors and when a non-timeout network
// error is returned, the pool on the server can be cleared.
type Connection struct {
	pc *connection
	s  *Server

	mu sync.RWMutex
}

func (sc *Connection) ReadWireMessage(ctx context.Context, dst []byte) ([]byte, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	wm, err := sc.pc.ReadWireMessage(ctx, dst)
	if err != nil {
		sc.processErr(err)
	} else {
		// e := command.DecodeError(wm)
		// sc.processErr(e)
	}
	return wm, err
}

func (sc *Connection) WriteWireMessage(ctx context.Context, wm []byte) error {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	err := sc.pc.WriteWireMessage(ctx, wm)
	sc.processErr(err)
	return err
}

func (sc *Connection) Description() description.Server {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	if sc.pc == nil {
		return description.Server{}
	}
	return sc.pc.desc
}

func (sc *Connection) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if sc.pc == nil {
		return nil
	}
	err := sc.pc.Close()
	sc.s.sem.Release(1)
	sc.pc = nil
	return err
}

func (sc *Connection) ID() string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return ""
}

func (sc *Connection) processErr(err error) {
	// TODO(GODRIVER-524) handle the rest of sdam error handling
	// Invalidate server description if not master or node recovering error occurs
	if cerr, ok := err.(command.Error); ok && (isRecoveringError(cerr) || isNotMasterError(cerr)) {
		desc := sc.Description()
		desc.Kind = description.Unknown
		desc.LastError = err
		// updates description to unknown
		sc.s.updateDescription(desc, false)
	}

	ne, ok := err.(NetworkError)
	if !ok {
		return
	}

	if netErr, ok := ne.Wrapped.(net.Error); ok && netErr.Timeout() {
		return
	}
	if ne.Wrapped == context.Canceled || ne.Wrapped == context.DeadlineExceeded {
		return
	}

	desc := sc.Description()
	desc.Kind = description.Unknown
	desc.LastError = err
	// updates description to unknown
	sc.s.updateDescription(desc, false)
}

func configureTLS(ctx context.Context, nc net.Conn, addr address.Address, config *TLSConfig) (net.Conn, error) {
	if !config.InsecureSkipVerify {
		hostname := addr.String()
		colonPos := strings.LastIndex(hostname, ":")
		if colonPos == -1 {
			colonPos = len(hostname)
		}

		hostname = hostname[:colonPos]
		config.ServerName = hostname
	}

	client := tls.Client(nc, config.Config)

	errChan := make(chan error, 1)
	go func() {
		errChan <- client.Handshake()
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return nil, err
		}
	case <-ctx.Done():
		return nil, errors.New("server connection cancelled/timeout during TLS handshake")
	}
	return client, nil
}

func isRecoveringError(err command.Error) bool {
	for _, c := range recoveringCodes {
		if c == err.Code {
			return true
		}
	}
	return strings.Contains(err.Error(), "node is recovering")
}

func isNotMasterError(err command.Error) bool {
	for _, c := range notMasterCodes {
		if c == err.Code {
			return true
		}
	}
	return strings.Contains(err.Error(), "not master")
}

func readInt32(b []byte, pos int32) int32 {
	return (int32(b[pos+0])) | (int32(b[pos+1]) << 8) | (int32(b[pos+2]) << 16) | (int32(b[pos+3]) << 24)
}
