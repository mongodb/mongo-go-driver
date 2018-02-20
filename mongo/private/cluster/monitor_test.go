// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package cluster_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"

	"github.com/mongodb/mongo-go-driver/mongo/private/cluster"
	"github.com/mongodb/mongo-go-driver/mongo/private/server"
	"github.com/stretchr/testify/require"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/internal"
	"github.com/mongodb/mongo-go-driver/mongo/internal/msgtest"
	"github.com/mongodb/mongo-go-driver/mongo/model"
	"github.com/mongodb/mongo-go-driver/mongo/private/conn"
	"github.com/mongodb/mongo-go-driver/mongo/private/msg"
)

func TestStop(t *testing.T) {
	h := &monitorHelper{}

	m, err := cluster.StartMonitor(
		cluster.WithSeedList("localhost"),
		cluster.WithServerOptions(
			server.WithConnectionOpener(h.open),
		))

	require.NoError(t, err)

	m.Stop()
}

type monitorHelper struct {
	connLock sync.Mutex
	conn     *fakeMonitorConn
	kind     model.ServerKind
}

func (m *monitorHelper) open(ctx context.Context, addr model.Addr, opts ...conn.Option) (conn.Connection, error) {
	m.connLock.Lock()
	defer m.connLock.Unlock()
	m.conn = &fakeMonitorConn{}
	return m.conn, nil
}

type fakeMonitorConn struct {
	dead      bool
	responses []msg.Response
}

func (c *fakeMonitorConn) Alive() bool {
	return !c.dead
}

func (c *fakeMonitorConn) Close() error {
	c.dead = true
	return nil
}

func (c *fakeMonitorConn) CloseIgnoreError() {
	_ = c.Close()
}

func (c *fakeMonitorConn) MarkDead() {
	c.dead = true
}

func (c *fakeMonitorConn) Model() *model.Conn {
	return &model.Conn{}
}

func (c *fakeMonitorConn) LocalAddr() net.Addr {
	return nil
}

func (c *fakeMonitorConn) Expired() bool {
	return c.dead
}

func (c *fakeMonitorConn) Read(_ context.Context, _ int32) (msg.Response, error) {
	if c.dead {
		return nil, fmt.Errorf("conn is closed")
	}
	r := c.responses[0]
	c.responses = c.responses[1:]
	return r, nil
}

func (c *fakeMonitorConn) Write(_ context.Context, msgs ...msg.Request) error {
	if c.dead {
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
