package topology

import (
	"context"
	"testing"

	"github.com/mongodb/mongo-go-driver/core/address"
	"github.com/mongodb/mongo-go-driver/core/auth"
	"github.com/mongodb/mongo-go-driver/core/connection"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/stretchr/testify/require"
)

type errorPool struct {
	address     address.Address
	drainCalled bool
}

type okPool struct {
	address     address.Address
	drainCalled bool
}

func (p *errorPool) Get(ctx context.Context) (connection.Connection, *description.Server, error) {
	return nil, nil, &auth.Error{}
}

func (p *errorPool) Connect(ctx context.Context) error {
	return nil
}

func (p *errorPool) Disconnect(ctx context.Context) error {
	return nil
}

func (p *errorPool) Drain() error {
	p.drainCalled = true
	return nil
}

func (p *okPool) Get(ctx context.Context) (connection.Connection, *description.Server, error) {
	return nil, nil, nil
}

func (p *okPool) Connect(ctx context.Context) error {
	return nil
}

func (p *okPool) Disconnect(ctx context.Context) error {
	return nil
}

func (p *okPool) Drain() error {
	p.drainCalled = true
	return nil
}

func NewErrorPool(addr address.Address) (connection.Pool, error) {
	p := &errorPool{
		address: addr,
	}
	return p, nil
}

func NewOkPool(addr address.Address) (connection.Pool, error) {
	p := &okPool{
		address: addr,
	}
	return p, nil
}

func TestServerAuthError(t *testing.T) {
	s, err := NewServer(address.Address("localhost"))
	require.NoError(t, err)

	s.pool, err = NewErrorPool(address.Address("localhost"))
	require.NoError(t, err)

	s.connectionstate = connected

	_, err = s.Connection(context.Background())
	require.Error(t, err)
	require.Equal(t, s.pool.(*errorPool).drainCalled, true)
}

func TestServerAuthNoError(t *testing.T) {
	s, err := NewServer(address.Address("localhost"))
	require.NoError(t, err)

	s.pool, err = NewOkPool(address.Address("localhost"))
	require.NoError(t, err)

	s.connectionstate = connected

	_, err = s.Connection(context.Background())
	require.NoError(t, err)
	require.Equal(t, s.pool.(*okPool).drainCalled, false)
}
