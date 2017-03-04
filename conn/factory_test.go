package conn_test

import (
	"context"
	"testing"

	"time"

	. "github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal/conntest"
	"github.com/stretchr/testify/require"
)

func TestLimitedFactory_only_allows_max_number_of_connections(t *testing.T) {
	t.Parallel()

	factory := func(_ context.Context) (Connection, error) {
		return &conntest.MockConnection{}, nil
	}

	limitedFactory := LimitedFactory(2, factory)

	limitedFactory(context.Background())
	limitedFactory(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(1 * time.Second)
		cancel()
	}()
	_, err := limitedFactory(ctx)
	require.Error(t, err)
}

func TestLimitedFactory_closing_a_connection_releases_a_resource(t *testing.T) {
	t.Parallel()

	factory := func(_ context.Context) (Connection, error) {
		return &conntest.MockConnection{}, nil
	}

	limitedFactory := LimitedFactory(2, factory)

	c1, _ := limitedFactory(context.Background())
	limitedFactory(context.Background())

	go func() {
		time.Sleep(1 * time.Second)
		c1.Close()
	}()
	_, err := limitedFactory(context.Background())
	require.NoError(t, err)
}
