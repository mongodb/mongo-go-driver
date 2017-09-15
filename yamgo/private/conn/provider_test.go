package conn_test

import (
	"context"
	"testing"

	"time"

	. "github.com/10gen/mongo-go-driver/yamgo/private/conn"
	"github.com/10gen/mongo-go-driver/yamgo/internal/conntest"
	"github.com/stretchr/testify/require"
)

func TestCappedProvider_only_allows_max_number_of_connections(t *testing.T) {
	t.Parallel()

	factory := func(_ context.Context) (Connection, error) {
		return &conntest.MockConnection{}, nil
	}

	cappedProvider := CappedProvider(2, factory)

	cappedProvider(context.Background())
	cappedProvider(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(1 * time.Second)
		cancel()
	}()
	_, err := cappedProvider(ctx)
	require.Error(t, err)
}

func TestCappedProvider_closing_a_connection_releases_a_resource(t *testing.T) {
	t.Parallel()

	factory := func(_ context.Context) (Connection, error) {
		return &conntest.MockConnection{}, nil
	}

	cappedProvider := CappedProvider(2, factory)

	c1, _ := cappedProvider(context.Background())
	cappedProvider(context.Background())

	go func() {
		time.Sleep(1 * time.Second)
		c1.Close()
	}()
	_, err := cappedProvider(context.Background())
	require.NoError(t, err)
}
