// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mtest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

// ProxyMessage represents a sent/received pair of parsed wire messages.
type ProxyMessage struct {
	ServerAddress string
	CommandName   string
	Sent          *SentMessage
	Received      *ReceivedMessage
}

// RawProxyMessage represents a sent/received pair of raw wire messages.
type RawProxyMessage struct {
	ServerAddress string
	Sent          wiremessage.WireMessage
	Received      wiremessage.WireMessage
}

// proxyDialer is a ContextDialer implementation that wraps a net.Dialer and records the messages sent and received
// using connections created through it.
type proxyDialer struct {
	*net.Dialer
	sync.Mutex
	// sentMap temporarily stores the message sent to the server using the requestID so it can map requests to their
	// responses.
	sentMap         map[int32]*SentMessage
	messages        []*ProxyMessage
	rawMessages     []*RawProxyMessage
	dialedAddresses map[string]struct{}

	rawMessagesOnly bool
}

var _ options.ContextDialer = (*proxyDialer)(nil)

func newProxyDialer() *proxyDialer {
	return &proxyDialer{
		Dialer:          &net.Dialer{Timeout: 30 * time.Second},
		sentMap:         make(map[int32]*SentMessage),
		dialedAddresses: make(map[string]struct{}),
	}
}

func newRawProxyDialer() *proxyDialer {
	pd := newProxyDialer()
	pd.rawMessagesOnly = true
	return pd
}

func newProxyError(err error) error {
	return fmt.Errorf("proxy error: %v", err)
}

func newProxyErrorWithWireMsg(wm []byte, err error) error {
	return fmt.Errorf("proxy error for wiremessage %v: %v", wm, err)
}

// DialContext creates a new proxyConnection.
func (p *proxyDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	p.dialedAddresses[address] = struct{}{}
	netConn, err := p.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return netConn, err
	}

	proxy := &proxyConn{
		Conn:   netConn,
		dialer: p,
	}
	return proxy, nil
}

func (p *proxyDialer) storeSentMessage(wm []byte) error {
	p.Lock()
	defer p.Unlock()

	// Create a copy of the wire message so it can be parsed/stored and will not be affected if the wm slice is
	// changed by the driver.
	wmCopy := copyBytes(wm)
	parsed, err := parseSentMessage(wmCopy, p.rawMessagesOnly)
	if err != nil {
		return err
	}
	p.sentMap[parsed.RequestID] = parsed
	return nil
}

// getDialedAddress gets the address that was originally dialed corresponding to the given address.
func (p *proxyDialer) getDialedAddress(addr string) (string, error) {
	if _, ok := p.dialedAddresses[addr]; ok {
		return addr, nil
	}

	// If localhost:<port> was dialed, the internally reported remote address will be 127.0.0.1:<port>. In this case,
	// check the dialedAddresses map again with the localhost:<port>.
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("net.SplitHostPort error: %v", err)
	}

	if host == "127.0.0.1" {
		host = "localhost"
	}
	addr = net.JoinHostPort(host, port)
	if _, ok := p.dialedAddresses[addr]; ok {
		return addr, nil
	}

	return "", errors.New("address not found")
}

func (p *proxyDialer) storeReceivedMessage(wm []byte, addr string) error {
	p.Lock()
	defer p.Unlock()

	serverAddress, err := p.getDialedAddress(addr)
	if err != nil {
		return fmt.Errorf("error getting known address for %s: %v", addr, err)
	}

	// Create a copy of the wire message so it can be parsed/stored and will not be affected if the wm slice is
	// changed by the driver. Parse the incoming message and get the corresponding outgoing message.
	wmCopy := copyBytes(wm)
	parsed, err := parseReceivedMessage(wmCopy, p.rawMessagesOnly)
	if err != nil {
		return err
	}
	sent, ok := p.sentMap[parsed.ResponseTo]
	if !ok {
		return errors.New("no sent message found")
	}
	delete(p.sentMap, parsed.ResponseTo)

	// Store the raw message pair.
	rawMsgPair := &RawProxyMessage{
		ServerAddress: serverAddress,
		Sent:          sent.RawMessage,
		Received:      parsed.RawMessage,
	}
	p.rawMessages = append(p.rawMessages, rawMsgPair)
	if p.rawMessagesOnly {
		return nil
	}

	// Store the parsed message pair.
	msgPair := &ProxyMessage{
		// The command name is always the first key in the command document.
		CommandName:   sent.Command.Index(0).Key(),
		ServerAddress: serverAddress,
		Sent:          sent,
		Received:      parsed,
	}
	p.messages = append(p.messages, msgPair)
	return nil
}

// proxyConn is a net.Conn that wraps a network connection. All messages sent/received through a proxyConn are stored
// in the associated proxyDialer and are forwarded over the wrapped connection. Errors encountered when parsing and
// storing wire messages are wrapped to add context, while errors returned from the underlying network connection are
// forwarded without wrapping.
type proxyConn struct {
	net.Conn
	dialer *proxyDialer
}

// Write stores the given message in the proxyDialer associated with this connection and forwards the message to the
// server.
func (pc *proxyConn) Write(wm []byte) (n int, err error) {
	if err := pc.dialer.storeSentMessage(wm); err != nil {
		wrapped := fmt.Errorf("error storing sent message: %v", err)
		return 0, newProxyErrorWithWireMsg(wm, wrapped)
	}

	return pc.Conn.Write(wm)
}

// Read reads the message from the server into the given buffer and stores the read message in the proxyDialer
// associated with this connection.
func (pc *proxyConn) Read(buffer []byte) (int, error) {
	n, err := pc.Conn.Read(buffer)
	if err != nil {
		return n, err
	}

	// The driver reads wire messages in two phases: a four-byte read to get the length of the incoming wire message
	// and a (length-4) byte read to get the message itself. There's nothing to be stored during the initial four-byte
	// read because we can calculate the length from the rest of the message.
	if len(buffer) == 4 {
		return 4, nil
	}

	// The buffer contains the entire wire message except for the length bytes. Re-create the full message by appending
	// buffer to the end of a four-byte slice and using UpdateLength to set the length bytes.
	idx, wm := bsoncore.ReserveLength(nil)
	wm = append(wm, buffer...)
	wm = bsoncore.UpdateLength(wm, idx, int32(len(wm[idx:])))

	if err := pc.dialer.storeReceivedMessage(wm, pc.RemoteAddr().String()); err != nil {
		wrapped := fmt.Errorf("error storing received message: %v", err)
		return 0, newProxyErrorWithWireMsg(wm, wrapped)
	}

	return n, nil
}
