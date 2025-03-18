package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestCompileCheck(t *testing.T) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:      "alpine",
		Cmd:        []string{"echo", "hello world"},
		WaitingFor: wait.ForLog("hello world"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	require.NoError(t, err)

	defer func() {
		err := container.Terminate(ctx)
		require.NoError(t, err)
	}()
}
