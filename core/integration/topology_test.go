package integration

import (
	"context"
	"net"
	"testing"

	"github.com/mongodb/mongo-go-driver/core/connection"
	"github.com/mongodb/mongo-go-driver/core/connstring"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/topology"
)

func TestTopologyTopology(t *testing.T) {
	t.Run("Disconnect", func(t *testing.T) {
		t.Run("cannot disconnect before connecting", func(t *testing.T) {
			topo, err := topology.New(topology.WithConnString(func(connstring.ConnString) connstring.ConnString { return connectionString }))
			noerr(t, err)
			err = topo.Disconnect(context.TODO())
			if err != topology.ErrTopologyClosed {
				t.Errorf("Expected a topology disconnected error. got %v; want %v", err, topology.ErrTopologyClosed)
			}
		})
		t.Run("cannot disconnect twice", func(t *testing.T) {
			topo, err := topology.New(topology.WithConnString(func(connstring.ConnString) connstring.ConnString { return connectionString }))
			noerr(t, err)
			err = topo.Connect(context.TODO())
			noerr(t, err)
			err = topo.Disconnect(context.TODO())
			noerr(t, err)
			err = topo.Disconnect(context.TODO())
			if err != topology.ErrTopologyClosed {
				t.Errorf("Expected a topology disconnected error. got %v; want %v", err, topology.ErrTopologyClosed)
			}
		})
		t.Run("all open sockets should be closed after disconnect", func(t *testing.T) {
			d := newdialer(&net.Dialer{})
			topo, err := topology.New(
				topology.WithConnString(
					func(connstring.ConnString) connstring.ConnString { return connectionString },
				),
				topology.WithServerOptions(func(opts ...topology.ServerOption) []topology.ServerOption {
					return append(
						opts,
						topology.WithConnectionOptions(func(opts ...connection.Option) []connection.Option {
							return append(
								opts,
								connection.WithDialer(func(connection.Dialer) connection.Dialer { return d }),
							)
						}),
					)
				}),
			)
			noerr(t, err)
			err = topo.Connect(context.TODO())
			noerr(t, err)
			ss, err := topo.SelectServer(context.TODO(), description.WriteSelector())
			noerr(t, err)

			conns := [3]connection.Connection{}
			for idx := range [3]struct{}{} {
				conns[idx], err = ss.Connection(context.TODO())
				noerr(t, err)
			}
			for idx := range [2]struct{}{} {
				err = conns[idx].Close()
				noerr(t, err)
			}
			if d.lenopened() < 3 {
				t.Errorf("Should have opened at least 3 connections, but didn't. got %d; want >%d", d.lenopened(), 3)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err = topo.Disconnect(ctx)
			noerr(t, err)
			if d.lenclosed() != d.lenopened() {
				t.Errorf(
					"Should have closed the same number of connections as opened. closed %d; opened %d",
					d.lenclosed(), d.lenopened())
			}
		})
	})
	t.Run("Connect", func(t *testing.T) {
		t.Run("can reconnect a disconnected topology", func(t *testing.T) {
			topo, err := topology.New(
				topology.WithConnString(
					func(connstring.ConnString) connstring.ConnString { return connectionString },
				),
			)
			noerr(t, err)
			err = topo.Connect(context.TODO())
			noerr(t, err)
			err = topo.Disconnect(context.TODO())
			noerr(t, err)
			err = topo.Connect(context.TODO())
			noerr(t, err)
		})
		t.Run("cannot connect multiple times without disconnect", func(t *testing.T) {
			topo, err := topology.New(
				topology.WithConnString(
					func(connstring.ConnString) connstring.ConnString { return connectionString },
				),
			)
			noerr(t, err)
			err = topo.Connect(context.TODO())
			noerr(t, err)
			err = topo.Disconnect(context.TODO())
			noerr(t, err)
			err = topo.Connect(context.TODO())
			noerr(t, err)
			err = topo.Connect(context.TODO())
			if err != topology.ErrTopologyConnected {
				t.Errorf("Expected a topology connected error. got %v; want %v", err, topology.ErrTopologyConnected)
			}
		})
		t.Run("can disconnect and reconnect multiple times", func(t *testing.T) {
			topo, err := topology.New(
				topology.WithConnString(
					func(connstring.ConnString) connstring.ConnString { return connectionString },
				),
			)
			noerr(t, err)
			err = topo.Connect(context.TODO())
			noerr(t, err)

			err = topo.Disconnect(context.TODO())
			noerr(t, err)
			err = topo.Connect(context.TODO())
			noerr(t, err)

			err = topo.Disconnect(context.TODO())
			noerr(t, err)
			err = topo.Connect(context.TODO())
			noerr(t, err)

			err = topo.Disconnect(context.TODO())
			noerr(t, err)
			err = topo.Connect(context.TODO())
			noerr(t, err)
		})
	})
}
