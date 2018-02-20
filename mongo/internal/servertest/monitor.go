// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package servertest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/internal"
	"github.com/mongodb/mongo-go-driver/mongo/internal/msgtest"
	"github.com/mongodb/mongo-go-driver/mongo/model"
	"github.com/mongodb/mongo-go-driver/mongo/private/conn"
	"github.com/mongodb/mongo-go-driver/mongo/private/msg"
	"github.com/mongodb/mongo-go-driver/mongo/private/server"
)

// NewFakeMonitor creates a fake monitor.
func NewFakeMonitor(kind model.ServerKind, addr model.Addr, opts ...server.Option) *FakeMonitor {
	fm := &FakeMonitor{
		kind: kind,
	}
	opts = append(opts, server.WithConnectionOpener(fm.open))
	m, _ := server.StartMonitor(addr, opts...)
	fm.Monitor = m
	return fm
}

// FakeMonitor is a fake monitor.
type FakeMonitor struct {
	*server.Monitor

	connLock sync.Mutex
	conn     *fakeMonitorConn
	kind     model.ServerKind
}

// SetKind sets the server kind for the monitor.
func (m *FakeMonitor) SetKind(kind model.ServerKind) error {
	m.connLock.Lock()
	defer m.connLock.Unlock()

	if m.kind != kind {

		m.kind = kind
		if m.conn != nil {
			return m.conn.Close()
		}
	}

	return nil
}

func (m *FakeMonitor) open(ctx context.Context, addr model.Addr, opts ...conn.Option) (conn.Connection, error) {
	m.connLock.Lock()
	defer m.connLock.Unlock()

	if m.kind == model.Unknown {
		return nil, fmt.Errorf("server type is unknown")
	}

	m.conn = &fakeMonitorConn{}
	return m.conn, nil
}

type fakeMonitorConn struct {
	sync.Mutex
	Dead      bool
	responses []msg.Response
}

func (c *fakeMonitorConn) Alive() bool {
	c.Lock()
	defer c.Unlock()

	return !c.Dead
}

func (c *fakeMonitorConn) Close() error {
	c.MarkDead()
	return nil
}

func (c *fakeMonitorConn) CloseIgnoreError() {
	_ = c.Close()
}

func (c *fakeMonitorConn) LocalAddr() net.Addr {
	return nil
}

func (c *fakeMonitorConn) MarkDead() {
	c.Lock()
	defer c.Unlock()

	c.Dead = true
}

func (c *fakeMonitorConn) Model() *model.Conn {
	return &model.Conn{}
}

func (c *fakeMonitorConn) Expired() bool {
	c.Lock()
	defer c.Unlock()

	return c.Dead
}

func (c *fakeMonitorConn) Read(_ context.Context, _ int32) (msg.Response, error) {
	if c.Expired() {
		return nil, fmt.Errorf("conn is closed")
	}
	r := c.responses[0]
	c.responses = c.responses[1:]
	return r, nil
}

func (c *fakeMonitorConn) Write(_ context.Context, msgs ...msg.Request) error {
	if c.Expired() {
		return fmt.Errorf("conn is closed")
	}

	for _, m := range msgs {
		var reply *msg.Reply
		switch typedM := m.(type) {
		case *msg.Query:
			doc := typedM.Query.(*bson.Document)
			elem, ok := doc.ElementAtOK(0)
			if !ok {
				return errors.New("empty response received")
			}
			switch elem.Key() {
			case "ismaster":
				reply = msgtest.CreateCommandReply(
					internal.IsMasterResult{
						OK:       1,
						IsMaster: true,
					},
				)
			case "buildInfo":
				reply = msgtest.CreateCommandReply(
					internal.BuildInfoResult{
						OK:           true,
						VersionArray: []uint8{3, 4, 0},
					},
				)
			default:
				return fmt.Errorf("unknown response to %s", elem.Key())
			}
		}
		reply.RespTo = m.RequestID()
		c.responses = append(c.responses, reply)
	}

	return nil
}
