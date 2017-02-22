package cluster

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal/clustertest"
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
	if len(servers) == 0 {
		return []*server.Desc{}, nil
	}
	return servers[0:1], nil
}

func selectError(_ *Desc, _ []*server.Desc) ([]*server.Desc, error) {
	return nil, errors.New("encountered an error in the selector")
}

func newTestCluster(endpoints ...string) *Cluster {
	cluster := &Cluster{
		cfg:     newConfig(WithServerSelectionTimeout(3 * time.Second)),
		waiters: make(map[int64]chan struct{}),
		rand:    rand.New(rand.NewSource(time.Now().UnixNano())),
		monitor: &Monitor{},
	}
	cluster.setClusterEndpoints(endpoints...)
	return cluster
}

func (c *Cluster) setClusterEndpoints(endpoints ...string) {
	stateServers := make(map[conn.Endpoint]serverCloser)
	servers := make([]*server.Desc, len(endpoints))
	for i, end := range endpoints {
		endpoint := conn.Endpoint(end)
		server := &server.Desc{
			Endpoint: endpoint,
			Type:     server.Standalone,
		}
		servers[i] = server
		stateServers[endpoint] = clustertest.NewFakeServer(endpoint)
	}
	clusterDesc := &Desc{
		Servers: servers,
	}
	c.stateServers = stateServers
	c.stateDesc = clusterDesc
}

func endpointName(srv Server) string {
	return string(srv.Desc().Endpoint)
}

func TestSelectServer_Success(t *testing.T) {
	c := newTestCluster("one", "two", "three")
	srv, err := c.SelectServer(context.Background(), selectFirst)
	require.Nil(t, err)
	require.Equal(t, "one", endpointName(srv))
}

// this test should only fail every (10*iters) runs.
// mostly just a sanity check that we don't return the
// same endpoint every time from the suitable choices
func TestSelectServer_Random(t *testing.T) {
	c := newTestCluster(
		"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten",
	)
	iters := 100
	success := false
	lastValue := ""
	for i := 0; i < iters; i++ {
		srv, err := c.SelectServer(context.Background(), selectAll)
		require.Nil(t, err)
		endpointName := endpointName(srv)
		if endpointName != lastValue {
			success = true
			break
		}
	}
	require.True(t, success)
}

func TestSelectServer_Updated(t *testing.T) {
	c := newTestCluster()
	errCh := make(chan error)
	srvCh := make(chan Server)
	go func() {
		srv, err := c.SelectServer(context.Background(), selectFirst)
		errCh <- err
		srvCh <- srv
	}()
	time.Sleep(1 * time.Second)
	c.setClusterEndpoints("one", "two")
	c.waiterLock.Lock()
	for _, ch := range c.waiters {
		ch <- struct{}{}
	}
	c.waiterLock.Unlock()
	require.Nil(t, <-errCh)
	require.Equal(t, "one", endpointName(<-srvCh))
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
