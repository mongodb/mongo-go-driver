package internal_test

import (
	"context"
	"testing"
	"time"

	. "github.com/10gen/mongo-go-driver/yamgo/internal"
	"github.com/stretchr/testify/require"
)

func TestSemaphore_Wait(t *testing.T) {
	s := NewSemaphore(3)
	err := s.Wait(context.Background())
	require.NoError(t, err)
	err = s.Wait(context.Background())
	require.NoError(t, err)
	err = s.Wait(context.Background())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err = s.Wait(ctx)
		require.Error(t, err)
	}()

	time.Sleep(1 * time.Second)
	cancel()
}

func TestSemaphore_Release(t *testing.T) {
	s := NewSemaphore(3)
	err := s.Wait(context.Background())
	err = s.Wait(context.Background())
	err = s.Wait(context.Background())

	go func() {
		err = s.Wait(context.Background())
		require.NoError(t, err)
	}()

	time.Sleep(1 * time.Second)
	s.Release()
}
