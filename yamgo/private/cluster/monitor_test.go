package cluster_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/10gen/mongo-go-driver/yamgo/private/cluster"
	"github.com/10gen/mongo-go-driver/yamgo/private/server"
	"github.com/stretchr/testify/require"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/private/conn"
	"github.com/10gen/mongo-go-driver/yamgo/internal"
	"github.com/10gen/mongo-go-driver/yamgo/internal/msgtest"
	"github.com/10gen/mongo-go-driver/yamgo/model"
	"github.com/10gen/mongo-go-driver/yamgo/private/msg"
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

func (c *fakeMonitorConn) MarkDead() {
	c.dead = true
}

func (c *fakeMonitorConn) Model() *model.Conn {
	return &model.Conn{}
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
