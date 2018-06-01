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

type pool struct {
	connectionError bool
	drainCalled     bool
}

func (p *pool) Get(ctx context.Context) (connection.Connection, *description.Server, error) {
	if p.connectionError {
		return nil, nil, &auth.Error{}
	}
	return nil, nil, nil
}

func (p *pool) Connect(ctx context.Context) error {
	return nil
}

func (p *pool) Disconnect(ctx context.Context) error {
	return nil
}

func (p *pool) Drain() error {
	p.drainCalled = true
	return nil
}

func NewPool(connectionError bool) (connection.Pool, error) {
	p := &pool{
		connectionError: connectionError,
	}
	return p, nil
}

func TestSever(t *testing.T) {
	var serverTestTable = []struct {
		name            string
		connectionError bool
	}{
		{"auth_error", true},
		{"auth_no_error", false},
	}

	for _, tt := range serverTestTable {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewServer(address.Address("localhost"))
			require.NoError(t, err)

			s.pool, err = NewPool(tt.connectionError)
			s.connectionstate = connected

			_, err = s.Connection(context.Background())

			if tt.connectionError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, s.pool.(*pool).drainCalled, tt.connectionError)
		})
	}
}
