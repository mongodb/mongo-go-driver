package conn_test

import (
	"context"
	"fmt"
	"testing"

	. "github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal/conntest"
	"github.com/stretchr/testify/require"
)

func TestPool_Get(t *testing.T) {
	var created []*conntest.MockConnection
	factory := func(_ context.Context) (Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	p := NewPool(2, factory)

	c1, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)
	c2, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)
	c3, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)

	c3.Close()

	c4, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)

	c4.Close()
	c2.Close()

	c5, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)
	c6, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)

	c6.Close()
	c5.Close()
	c1.Close()

	_, err = p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)
	_, err = p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)
	_, err = p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 4)
}

func TestPool_Get_a_connection_which_expired_in_the_pool(t *testing.T) {
	var created []*conntest.MockConnection
	factory := func(_ context.Context) (Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	p := NewPool(2, factory)
	c1, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)
	c1.Close()

	created[0].Dead = true

	_, err = p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)
}

func TestPool_Get_when_context_is_done(t *testing.T) {
	var created []*conntest.MockConnection
	factory := func(_ context.Context) (Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	p := NewPool(2, factory)
	_, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)
	_, err = p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_, err = p.Get(ctx)
		require.Error(t, err)
		require.Len(t, created, 2)
	}()

	cancel()
}

func TestPool_Get_returns_an_error_when_unable_to_create_a_connection(t *testing.T) {
	factory := func(_ context.Context) (Connection, error) {
		return nil, fmt.Errorf("AGAHAA")
	}

	p := NewPool(2, factory)

	_, err := p.Get(context.Background())
	require.Error(t, err)
}

func TestPool_Get_returns_an_error_after_pool_is_closed(t *testing.T) {
	factory := func(_ context.Context) (Connection, error) {
		return &conntest.MockConnection{}, nil
	}

	p := NewPool(2, factory)

	p.Close()

	_, err := p.Get(context.Background())
	require.Error(t, err)
}

func TestPool_Connection_Close_does_not_error_after_pool_is_closed(t *testing.T) {
	var created []*conntest.MockConnection
	factory := func(_ context.Context) (Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	p := NewPool(2, factory)

	c1, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)

	p.Close()
	err = c1.Close()
	require.NoError(t, err)
	require.False(t, created[0].Alive())
}

func TestPool_Connection_Close_does_not_close_underlying_connection_when_it_is_not_expired(t *testing.T) {
	var created []*conntest.MockConnection
	factory := func(_ context.Context) (Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	p := NewPool(1, factory)

	actualC, err := p.Get(context.Background())
	require.NoError(t, err)

	require.Len(t, created, 1)

	err = actualC.Close()
	require.NoError(t, err)

	require.True(t, created[0].Alive())
}

func TestPool_Connection_Close_closes_underlying_connection_when_it_is_expired(t *testing.T) {
	var created []*conntest.MockConnection
	factory := func(_ context.Context) (Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	p := NewPool(1, factory)

	c1, err := p.Get(context.Background())
	require.NoError(t, err)

	require.True(t, created[0].Alive())

	p.Clear()

	err = c1.Close()
	require.NoError(t, err)
	require.False(t, created[0].Alive())
}

func TestPool_Connection_Close_closes_underlying_connection_when_pool_is_full(t *testing.T) {
	var created []*conntest.MockConnection
	factory := func(_ context.Context) (Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	p := NewPool(2, factory)

	c1, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)
	c2, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)
	c3, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)

	c1.Close()
	require.True(t, created[0].Alive())
	c2.Close()
	require.True(t, created[1].Alive())
	c3.Close()
	require.False(t, created[2].Alive())
}

func TestPool_Clear_expires_existing_checked_out_connections(t *testing.T) {
	var created []*conntest.MockConnection
	factory := func(_ context.Context) (Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	p := NewPool(1, factory)

	c1, err := p.Get(context.Background())
	require.NoError(t, err)

	require.False(t, c1.Expired())

	p.Clear()

	require.True(t, c1.Expired())
}

func TestPool_Clear_expires_idle_connections(t *testing.T) {
	var created []*conntest.MockConnection
	factory := func(_ context.Context) (Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	p := NewPool(2, factory)

	c1, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)
	c2, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)

	c1.Close()
	require.True(t, created[0].Alive())
	c2.Close()
	require.True(t, created[1].Alive())

	p.Clear()

	_, err = p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)

	require.False(t, created[0].Alive())
	require.False(t, created[1].Alive())
}

func TestPool_Close_can_be_called_multiple_times(t *testing.T) {
	factory := func(_ context.Context) (Connection, error) {
		return nil, nil
	}

	p := NewPool(2, factory)

	p.Close()
	p.Close()
}

func TestPool_Close_closes_all_connections_in_the_pool(t *testing.T) {
	var created []*conntest.MockConnection
	factory := func(_ context.Context) (Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	p := NewPool(2, factory)

	c1, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)
	c2, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)

	c1.Close()
	require.True(t, created[0].Alive())
	c2.Close()
	require.True(t, created[1].Alive())

	p.Close()

	require.False(t, created[0].Alive())
	require.False(t, created[1].Alive())
}
