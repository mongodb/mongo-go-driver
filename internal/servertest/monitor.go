package servertest

import (
	"context"
	"fmt"
	"sync"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal"
	"github.com/10gen/mongo-go-driver/internal/msgtest"
	"github.com/10gen/mongo-go-driver/msg"
	"github.com/10gen/mongo-go-driver/server"
	"gopkg.in/mgo.v2/bson"
)

// NewFakeMonitor creates a fake monitor.
func NewFakeMonitor(serverType server.Type, endpoint conn.Endpoint, opts ...server.Option) *FakeMonitor {
	fm := &FakeMonitor{
		serverType: serverType,
	}
	opts = append(opts, server.WithConnectionDialer(fm.dial))
	m, _ := server.StartMonitor(endpoint, opts...)
	fm.Monitor = m
	return fm
}

// FakeMonitor is a fake monitor.
type FakeMonitor struct {
	*server.Monitor

	connLock   sync.Mutex
	conn       *fakeMonitorConn
	serverType server.Type
}

// SetServerType sets the server type for the monitor and sets
// the connection to respond with the correct server type.
func (m *FakeMonitor) SetServerType(serverType server.Type) {
	if m.serverType != serverType {
		m.serverType = serverType
		m.connLock.Lock()
		if m.conn != nil {
			m.conn.Close()
		}
		m.connLock.Unlock()
	}
}

func (m *FakeMonitor) dial(ctx context.Context, endpoint conn.Endpoint, opts ...conn.Option) (conn.Connection, error) {
	if m.serverType == server.Unknown {
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

func (c *fakeMonitorConn) Desc() *conn.Desc {
	return &conn.Desc{}
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
