package conn

import (
	"context"

	"github.com/10gen/mongo-go-driver/yamgo/internal"
	"github.com/10gen/mongo-go-driver/yamgo/model"
)

// Provider gets a connection.
type Provider func(context.Context) (Connection, error)

// OpeningProvider returns a Factory that uses a dialer.
func OpeningProvider(opener Opener, addr model.Addr, opts ...Option) Provider {
	return func(ctx context.Context) (Connection, error) {
		return opener(ctx, addr, opts...)
	}
}

// CappedProvider returns a Provider that is constrained by a resource
// limit.
func CappedProvider(max uint64, provider Provider) Provider {
	permits := internal.NewSemaphore(max)
	return func(ctx context.Context) (Connection, error) {

		err := permits.Wait(ctx)
		if err != nil {
			return nil, err
		}

		c, err := provider(ctx)
		if err != nil {
			permits.Release()
			return nil, err
		}
		return &cappedProviderConn{c, permits}, nil
	}
}

type cappedProviderConn struct {
	Connection
	permits *internal.Semaphore
}

func (c *cappedProviderConn) Close() error {
	c.permits.Release()
	return c.Connection.Close()
}
