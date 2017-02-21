package cluster

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/server"
	"github.com/stretchr/testify/require"
)

func selectAll(_ *Desc, servers []*server.Desc) ([]*server.Desc, error) {
	return servers, nil
}

func selectNone(_ *Desc, _ []*server.Desc) ([]*server.Desc, error) {
	return []*server.Desc{}, nil
}

func selectFirst(_ *Desc, servers []*server.Desc) ([]*server.Desc, error) {
	return servers[0:1], nil
}

func selectError(_ *Desc, _ []*server.Desc) ([]*server.Desc, error) {
	return nil, errors.New("encountered an error in the selector")
}

func newTestCluster() *Cluster {
	return &Cluster{
		cfg:          newConfig(WithServerSelectionTimeout(3 * time.Second)),
		stateDesc:    &Desc{},
		stateServers: make(map[conn.Endpoint]*server.Server),
		waiters:      make(map[int64]chan struct{}),
		rand:         rand.New(rand.NewSource(time.Now().UnixNano())),
		monitor:      &Monitor{},
	}
}

func TestSelectServer_Timeout(t *testing.T) {
	c := newTestCluster()
	done := make(chan error)
	go func() {
		_, err := c.SelectServer(context.Background(), selectNone)
		done <- err
	}()

	select {
	case <-time.After(2 * time.Second):
		// this is what we expect
	case <-done:
		t.Fatal("SelectServer returned before ServerSelectionTimeout")
	}

	select {
	case <-time.After(2 * time.Second):
		t.Fatal("ServerSelectionTimeout exceeded, but SelectServer has not returned")
	case err := <-done:
		// this is what we expect
		require.Equal(t, "server selection timed out", err.Error())
	}
}

func TestSelectServer_Cancel(t *testing.T) {
	c := newTestCluster()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error)
	go func() {
		_, err := c.SelectServer(ctx, selectNone)
		done <- err
	}()

	select {
	case <-time.After(2 * time.Second):
		// this is what we expect
	case <-done:
		t.Fatal("SelectServer returned before ServerSelectionTimeout")
	}

	cancel()

	select {
	case <-time.After(2 * time.Second):
		t.Fatal("ServerSelectionTimeout exceeded, but SelectServer has not returned")
	case err := <-done:
		// this is what we expect
		require.Equal(t, "server selection failed: context canceled", err.Error())
	}
}

func TestSelectServer_Error(t *testing.T) {
	c := newTestCluster()
	done := make(chan error)
	go func() {
		_, err := c.SelectServer(context.Background(), selectError)
		done <- err
	}()

	select {
	case <-time.After(2 * time.Second):
		t.Fatal("ServerSelectionTimeout exceeded, but SelectServer has not returned")
	case err := <-done:
		// this is what we expect
		require.Equal(t, "encountered an error in the selector", err.Error())
	}
}
