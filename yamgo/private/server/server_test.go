package server_test

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/10gen/mongo-go-driver/yamgo/internal/conntest"
	"github.com/10gen/mongo-go-driver/yamgo/internal/servertest"
	"github.com/10gen/mongo-go-driver/yamgo/internal/testutil"
	"github.com/10gen/mongo-go-driver/yamgo/model"
	"github.com/10gen/mongo-go-driver/yamgo/private/conn"
	"github.com/10gen/mongo-go-driver/yamgo/private/msg"
	. "github.com/10gen/mongo-go-driver/yamgo/private/server"
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
	s, _ := NewWithMonitor(
		fake.Monitor,
		WithConnectionOpener(dialer),
		WithMaxConnections(2),
	)

	require.NoError(t, s.Close())

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
	s, _ := NewWithMonitor(
		fake.Monitor,
		WithConnectionOpener(dialer),
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
	s, _ := NewWithMonitor(
		fake.Monitor,
		WithConnectionOpener(dialer),
		WithMaxConnections(2),
	)

	c1, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)

	c2, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)

	testutil.RequireNoErrorOnClose(t, c1)
	testutil.RequireNoErrorOnClose(t, c2)

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
	s, _ := NewWithMonitor(
		fake.Monitor,
		WithConnectionOpener(dialer),
		WithMaxConnections(2),
	)

	c1, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)

	c2, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)

	testutil.RequireNoErrorOnClose(t, c1)
	testutil.RequireNoErrorOnClose(t, c2)

	require.NoError(t, fake.SetKind(model.Unknown))
	time.Sleep(1 * time.Second)

	_, err = s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)

	_, err = s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 4)
}

func TestServer_Connection_Read_non_specific_error_should_clear_the_pool(t *testing.T) {
	t.Parallel()

	var created []*conntest.MockConnection
	dialer := func(ctx context.Context, addr model.Addr, opts ...conn.Option) (conn.Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	fake := servertest.NewFakeMonitor(model.Standalone, model.Addr("localhost:27017"), WithHeartbeatInterval(100*time.Second))
	s, _ := NewWithMonitor(
		fake.Monitor,
		WithConnectionOpener(dialer),
		WithMaxConnections(2),
	)

	c1, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)

	c2, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)

	require.NoError(t, c1.Write(context.Background(), &msg.Query{}))
	// Don't check for errors here since no response is queued.
	_, _ = c1.Read(context.Background(), 0)

	c1.MarkDead()

	testutil.RequireNoErrorOnClose(t, c1)
	testutil.RequireNoErrorOnClose(t, c2)

	time.Sleep(1 * time.Second)

	_, err = s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)

	_, err = s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 4)
}

var ignoredErrs []error = []error{
	&net.DNSError{
		IsTimeout: true,
	},
	&net.DNSError{
		IsTemporary: true,
	},
	&url.Error{
		Err: &net.DNSError{
			IsTimeout: true,
		},
	},
	context.DeadlineExceeded,
	context.Canceled,
}

func TestServer_Connection_Read_specific_error_should_not_clear_the_pool(t *testing.T) {

	for _, ignoredErr := range ignoredErrs {
		t.Run(ignoredErr.Error(), func(t *testing.T) {
			t.Parallel()

			var created []*conntest.MockConnection
			dialer := func(ctx context.Context, addr model.Addr, opts ...conn.Option) (conn.Connection, error) {
				created = append(created, &conntest.MockConnection{})
				return created[len(created)-1], nil
			}

			fake := servertest.NewFakeMonitor(model.Standalone, model.Addr("localhost:27017"), WithHeartbeatInterval(100*time.Second))
			s, _ := NewWithMonitor(
				fake.Monitor,
				WithConnectionOpener(dialer),
				WithMaxConnections(2),
			)

			c1, err := s.Connection(context.Background())
			require.NoError(t, err)
			require.Len(t, created, 1)

			c2, err := s.Connection(context.Background())
			require.NoError(t, err)
			require.Len(t, created, 2)

			created[0].ReadErr = ignoredErr

			require.NoError(t, c1.Write(context.Background(), &msg.Query{}))
			_, err = c1.Read(context.Background(), 0)
			require.EqualError(t, err, ignoredErr.Error())

			c1.MarkDead()

			testutil.RequireNoErrorOnClose(t, c1)
			testutil.RequireNoErrorOnClose(t, c2)

			time.Sleep(1 * time.Second)

			_, err = s.Connection(context.Background())
			require.NoError(t, err)
			require.Len(t, created, 2)

			_, err = s.Connection(context.Background())
			require.NoError(t, err)
			require.Len(t, created, 3)
		})
	}
}

func TestServer_Connection_Write_non_specific_error_should_clear_the_pool(t *testing.T) {
	t.Parallel()

	var created []*conntest.MockConnection
	dialer := func(ctx context.Context, addr model.Addr, opts ...conn.Option) (conn.Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	fake := servertest.NewFakeMonitor(model.Standalone, model.Addr("localhost:27017"), WithHeartbeatInterval(100*time.Second))
	s, _ := NewWithMonitor(
		fake.Monitor,
		WithConnectionOpener(dialer),
		WithMaxConnections(2),
	)

	c1, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)

	c2, err := s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)

	nonSpecificError := fmt.Errorf("forced write error")
	created[0].WriteErr = nonSpecificError
	require.EqualError(t, c1.Write(context.Background(), &msg.Query{}), nonSpecificError.Error())
	c1.MarkDead()

	testutil.RequireNoErrorOnClose(t, c1)
	testutil.RequireNoErrorOnClose(t, c2)

	time.Sleep(1 * time.Second)

	_, err = s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)

	_, err = s.Connection(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 4)
}

func TestServer_Connection_Write_specific_error_should_not_clear_the_pool(t *testing.T) {
	for _, ignoredErr := range ignoredErrs {
		t.Run(ignoredErr.Error(), func(t *testing.T) {
			t.Parallel()

			var created []*conntest.MockConnection
			dialer := func(ctx context.Context, addr model.Addr, opts ...conn.Option) (conn.Connection, error) {
				created = append(created, &conntest.MockConnection{})
				return created[len(created)-1], nil
			}

			fake := servertest.NewFakeMonitor(model.Standalone, model.Addr("localhost:27017"), WithHeartbeatInterval(100*time.Second))
			s, _ := NewWithMonitor(
				fake.Monitor,
				WithConnectionOpener(dialer),
				WithMaxConnections(2),
			)

			c1, err := s.Connection(context.Background())
			require.NoError(t, err)
			require.Len(t, created, 1)

			c2, err := s.Connection(context.Background())
			require.NoError(t, err)
			require.Len(t, created, 2)

			created[0].WriteErr = ignoredErr
			require.EqualError(t, c1.Write(context.Background(), &msg.Query{}), ignoredErr.Error())
			c1.MarkDead()

			testutil.RequireNoErrorOnClose(t, c1)
			testutil.RequireNoErrorOnClose(t, c2)

			time.Sleep(1 * time.Second)

			_, err = s.Connection(context.Background())
			require.NoError(t, err)
			require.Len(t, created, 2)

			_, err = s.Connection(context.Background())
			require.NoError(t, err)
			require.Len(t, created, 3)
		})
	}
}
