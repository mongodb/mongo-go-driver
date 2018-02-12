// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package conn_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/mongodb/mongo-go-driver/mongo/internal/conntest"
	"github.com/mongodb/mongo-go-driver/mongo/internal/testutil/helpers"
	. "github.com/mongodb/mongo-go-driver/mongo/private/conn"
	"github.com/stretchr/testify/require"
)

func TestPool_caches_connections(t *testing.T) {
	t.Parallel()

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

	testhelpers.RequireNoErrorOnClose(t, c3)

	c4, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)

	testhelpers.RequireNoErrorOnClose(t, c4)
	testhelpers.RequireNoErrorOnClose(t, c2)

	c5, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)
	c6, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)

	testhelpers.RequireNoErrorOnClose(t, c6)
	testhelpers.RequireNoErrorOnClose(t, c5)
	testhelpers.RequireNoErrorOnClose(t, c1)

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
	t.Parallel()

	var created []*conntest.MockConnection
	factory := func(_ context.Context) (Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	p := NewPool(2, factory)
	c1, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)
	testhelpers.RequireNoErrorOnClose(t, c1)

	created[0].Dead = true

	_, err = p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 2)
}

func TestPool_Get_when_context_is_done(t *testing.T) {
	t.Parallel()

	var created []*conntest.MockConnection
	factory := func(ctx context.Context) (Connection, error) {
		if len(created) == 2 {
			<-ctx.Done()
			return nil, ctx.Err()
		}
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
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		_, err = p.Get(ctx)
		require.Error(t, err)
		require.Len(t, created, 2)
		wg.Done()
	}()

	cancel()
	wg.Wait()
}

func TestPool_Get_returns_an_error_when_unable_to_create_a_connection(t *testing.T) {
	t.Parallel()

	factory := func(_ context.Context) (Connection, error) {
		return nil, fmt.Errorf("AGAHAA")
	}

	p := NewPool(2, factory)

	_, err := p.Get(context.Background())
	require.Error(t, err)
}

func TestPool_Get_returns_an_error_after_pool_is_closed(t *testing.T) {
	t.Parallel()

	factory := func(_ context.Context) (Connection, error) {
		return &conntest.MockConnection{}, nil
	}

	p := NewPool(2, factory)

	testhelpers.RequireNoErrorOnClose(t, p)

	_, err := p.Get(context.Background())
	require.Error(t, err)
}

func TestPool_Connection_Close_does_not_error_after_pool_is_closed(t *testing.T) {
	t.Parallel()

	var created []*conntest.MockConnection
	factory := func(_ context.Context) (Connection, error) {
		created = append(created, &conntest.MockConnection{})
		return created[len(created)-1], nil
	}

	p := NewPool(2, factory)

	c1, err := p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)

	testhelpers.RequireNoErrorOnClose(t, p)
	err = c1.Close()
	require.NoError(t, err)
	require.False(t, created[0].Alive())
}

func TestPool_Connection_Close_does_not_close_underlying_connection_when_it_is_not_expired(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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

	testhelpers.RequireNoErrorOnClose(t, c1)
	require.True(t, created[0].Alive())
	testhelpers.RequireNoErrorOnClose(t, c2)
	require.True(t, created[1].Alive())
	testhelpers.RequireNoErrorOnClose(t, c3)
	require.False(t, created[2].Alive())
}

func TestPool_Clear_expires_existing_checked_out_connections(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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

	testhelpers.RequireNoErrorOnClose(t, c1)
	require.True(t, created[0].Alive())
	testhelpers.RequireNoErrorOnClose(t, c2)
	require.True(t, created[1].Alive())

	p.Clear()

	_, err = p.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 3)

	require.False(t, created[0].Alive())
	require.False(t, created[1].Alive())
}

func TestPool_Close_can_be_called_multiple_times(t *testing.T) {
	t.Parallel()

	factory := func(_ context.Context) (Connection, error) {
		return nil, nil
	}

	p := NewPool(2, factory)

	testhelpers.RequireNoErrorOnClose(t, p)
	testhelpers.RequireNoErrorOnClose(t, p)
}

func TestPool_Close_closes_all_connections_in_the_pool(t *testing.T) {
	t.Parallel()

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

	testhelpers.RequireNoErrorOnClose(t, c1)
	require.True(t, created[0].Alive())
	testhelpers.RequireNoErrorOnClose(t, c2)
	require.True(t, created[1].Alive())

	testhelpers.RequireNoErrorOnClose(t, p)

	require.False(t, created[0].Alive())
	require.False(t, created[1].Alive())
}
