package servertest

import (
	"context"
	"fmt"
	"sync"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/private/conn"
	"github.com/10gen/mongo-go-driver/yamgo/internal"
	"github.com/10gen/mongo-go-driver/yamgo/internal/msgtest"
	"github.com/10gen/mongo-go-driver/yamgo/model"
	"github.com/10gen/mongo-go-driver/yamgo/private/msg"
	"github.com/10gen/mongo-go-driver/yamgo/private/server"
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
func (m *FakeMonitor) SetKind(kind model.ServerKind) {
	if m.kind != kind {
		m.kind = kind
		m.connLock.Lock()
		if m.conn != nil {
			m.conn.Close()
		}
		m.connLock.Unlock()
	}
}

func (m *FakeMonitor) open(ctx context.Context, addr model.Addr, opts ...conn.Option) (conn.Connection, error) {
	if m.kind == model.Unknown {
		return nil, fmt.Errorf("server type is unknown")
	}

	m.connLock.Lock()
	defer m.connLock.Unlock()
	m.conn = &fakeMonitorConn{}
	return m.conn, nil
}

type fakeMonitorConn struct {
	Dead      bool
	responses []msg.Response
}

func (c *fakeMonitorConn) Alive() bool {
	return !c.Dead
}

func (c *fakeMonitorConn) Close() error {
	c.Dead = true
	return nil
}

func (c *fakeMonitorConn) MarkDead() {
	c.Dead = true
}

func (c *fakeMonitorConn) Model() *model.Conn {
	return &model.Conn{}
}

func (c *fakeMonitorConn) Expired() bool {
	return c.Dead
}

func (c *fakeMonitorConn) Read(_ context.Context, _ int32) (msg.Response, error) {
	if c.Dead {
		return nil, fmt.Errorf("conn is closed")
	}
	r := c.responses[0]
	c.responses = c.responses[1:]
	return r, nil
}

func (c *fakeMonitorConn) Write(_ context.Context, msgs ...msg.Request) error {
	if c.Dead {
		return fmt.Errorf("conn is closed")
	}
	for _, m := range msgs {
		var reply *msg.Reply
		switch typedM := m.(type) {
		case *msg.Query:
			doc := typedM.Query.(bson.D)
			switch doc[0].Name {
			case "ismaster":
				reply = msgtest.CreateCommandReply(
					internal.IsMasterResult{
						OK:       true,
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
				return fmt.Errorf("unknown response to %s", doc[0].Name)
			}
		}
		reply.RespTo = m.RequestID()
		c.responses = append(c.responses, reply)
	}

	return nil
}
