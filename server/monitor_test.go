package server_test

import (
	"testing"
	"time"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal/servertest"
	. "github.com/10gen/mongo-go-driver/server"
	"github.com/stretchr/testify/require"
)

func TestMonitor_Close_should_close_all_update_channels(t *testing.T) {
	t.Parallel()

	fm := servertest.NewFakeMonitor(Standalone, conn.Endpoint("localhost:27017"))

	updates1, _, _ := fm.Subscribe()
	done1 := false
	go func() {
		for range updates1 {
		}
		done1 = true
	}()
	updates2, _, _ := fm.Subscribe()
	done2 := false
	go func() {
		for range updates2 {
		}
		done2 = true
	}()

	fm.Stop()

	time.Sleep(1 * time.Second)

	require.True(t, done1)
	require.True(t, done2)
}

func TestMonitor_Subscribe_after_close_should_return_an_error(t *testing.T) {
	t.Parallel()

	fm := servertest.NewFakeMonitor(Standalone, conn.Endpoint("localhost:27017"))

	fm.Stop()

	time.Sleep(1 * time.Second)

	_, _, err := fm.Subscribe()
	require.Error(t, err)
}
