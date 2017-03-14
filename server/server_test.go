package server_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal/conntest"
	"github.com/10gen/mongo-go-driver/internal/servertest"
	"github.com/10gen/mongo-go-driver/model"
	"github.com/10gen/mongo-go-driver/msg"
	. "github.com/10gen/mongo-go-driver/server"
	"github.com/stretchr/testify/require"
)

func TestServer_Close_should_not_return_new_connections(t *testing.T) {
	t.Parallel()

	var created []*conntest.MockConnection
	dialer := func(ctx context.Context, addr model.Addr, opts ...conn.Option) (conn.Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	fake := servertest.NewFakeMonitor(model.Standalone, model.Addr("localhost:27017"))
	s := NewWithMonitor(
		fake.Monitor,
		WithConnectionDialer(dialer),
		WithMaxConnections(2),
	)

	s.Close()

	_, err := s.Connection(context.Background())
	require.Error(t, err)
	require.Len(t, created, 0)
}

func TestServer_Connection_should_provide_up_to_maxConn_connections(t *testing.T) {
	t.Parallel()

	var created []*conntest.MockConnection
	dialer := func(ctx context.Context, addr model.Addr, opts ...conn.Option) (conn.Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	fake := servertest.NewFakeMonitor(model.Standalone, model.Addr("localhost:27017"))
	s := NewWithMonitor(
		fake.Monitor,
		WithConnectionDialer(dialer),
		WithMaxConnections(2),
	)

	_, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)

	_, err = s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	go func() {
		_, err = s.Connection(ctx)
		require.Error(t, err)
		require.Len(t, created, 2)
	}()

	cancel()
}

func TestServer_Connection_should_pool_connections(t *testing.T) {
	t.Parallel()

	var created []*conntest.MockConnection
	dialer := func(ctx context.Context, addr model.Addr, opts ...conn.Option) (conn.Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	fake := servertest.NewFakeMonitor(model.Standalone, model.Addr("localhost:27017"))
	s := NewWithMonitor(
		fake.Monitor,
		WithConnectionDialer(dialer),
		WithMaxConnections(2),
	)

	c1, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)

	c2, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)

	c1.Close()
	c2.Close()

	_, err = s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)

	_, err = s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)
}

func TestServer_Connection_should_clear_pool_when_monitor_fails(t *testing.T) {
	t.Parallel()

	var created []*conntest.MockConnection
	dialer := func(ctx context.Context, addr model.Addr, opts ...conn.Option) (conn.Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	fake := servertest.NewFakeMonitor(model.Standalone, model.Addr("localhost:27017"), WithHeartbeatInterval(1*time.Second))
	s := NewWithMonitor(
		fake.Monitor,
		WithConnectionDialer(dialer),
		WithMaxConnections(2),
	)

	c1, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)

	c2, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)

	c1.Close()
	c2.Close()

	fake.SetKind(model.Unknown)
	time.Sleep(1 * time.Second)

	_, err = s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)

	_, err = s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 4)
}

func TestServer_Connection_Read_failure_should_cause_immediate_monitor_check(t *testing.T) {
	t.Parallel()

	var created []*conntest.MockConnection
	dialer := func(ctx context.Context, addr model.Addr, opts ...conn.Option) (conn.Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	fake := servertest.NewFakeMonitor(model.Standalone, model.Addr("localhost:27017"), WithHeartbeatInterval(100*time.Second))
	s := NewWithMonitor(
		fake.Monitor,
		WithConnectionDialer(dialer),
		WithMaxConnections(2),
	)

	c1, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)

	c2, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)

	fake.SetKind(model.Unknown)
	c1.Write(context.Background(), &msg.Query{})
	c1.Read(context.Background(), 0)

	c1.Close()
	c2.Close()

	time.Sleep(1 * time.Second)

	_, err = s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)

	_, err = s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 4)
}

func TestServer_Connection_Write_failure_should_cause_immediate_monitor_check(t *testing.T) {
	t.Parallel()

	var created []*conntest.MockConnection
	dialer := func(ctx context.Context, addr model.Addr, opts ...conn.Option) (conn.Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	fake := servertest.NewFakeMonitor(model.Standalone, model.Addr("localhost:27017"), WithHeartbeatInterval(100*time.Second))
	s := NewWithMonitor(
		fake.Monitor,
		WithConnectionDialer(dialer),
		WithMaxConnections(2),
	)

	c1, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)

	c2, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)

	fake.SetKind(model.Unknown)
	created[0].WriteErr = fmt.Errorf("forced write error")
	c1.Write(context.Background(), &msg.Query{})

	c1.Close()
	c2.Close()

	time.Sleep(1 * time.Second)

	_, err = s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)

	_, err = s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 4)
}
