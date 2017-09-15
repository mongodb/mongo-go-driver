package cluster

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/10gen/mongo-go-driver/yamgo/model"
	"github.com/stretchr/testify/require"
)

func selectFirst(_ *model.Cluster, servers []*model.Server) ([]*model.Server, error) {
	if len(servers) == 0 {
		return []*model.Server{}, nil
	}
	return servers[0:1], nil
}

func selectNone(_ *model.Cluster, _ []*model.Server) ([]*model.Server, error) {
	return []*model.Server{}, nil
}

func selectError(_ *model.Cluster, _ []*model.Server) ([]*model.Server, error) {
	return nil, errors.New("encountered an error in the selector")
}

type fakeMonitor struct {
	updates chan *model.Cluster
}

func newFakeMonitor(endpoints ...string) *fakeMonitor {
	updates := make(chan *model.Cluster, 1)
	monitor := &fakeMonitor{
		updates: updates,
	}
	monitor.updateEndpoints(endpoints...)
	return monitor
}

func (f *fakeMonitor) Subscribe() (<-chan *model.Cluster, func(), error) {
	return f.updates, func() {}, nil
}

func (f *fakeMonitor) RequestImmediateCheck() {
	// no-op
}

func (f *fakeMonitor) updateEndpoints(endpoints ...string) {
	servers := make([]*model.Server, len(endpoints))
	for i, end := range endpoints {
		addr := model.Addr(end)
		server := &model.Server{
			Addr: addr,
			Kind: model.Standalone,
		}
		servers[i] = server
	}
	cluster := &model.Cluster{
		Servers: servers,
	}

	select {
	case <-f.updates:
	default:
	}
	f.updates <- cluster
}

func TestSelectServer_Success(t *testing.T) {
	m := newFakeMonitor("one", "two", "three")
	srvs, err := selectServers(context.Background(), m, selectFirst)
	require.NoError(t, err)
	require.Len(t, srvs, 1)
	require.Equal(t, "one", string(srvs[0].Addr))
}

func TestSelectServer_Updated(t *testing.T) {
	m := newFakeMonitor()
	errCh := make(chan error)
	srvCh := make(chan []*model.Server)

	go func() {
		srvs, err := selectServers(context.Background(), m, selectFirst)
		errCh <- err
		srvCh <- srvs
	}()

	time.Sleep(1 * time.Second)
	m.updateEndpoints("four", "five")
	err := <-errCh
	srvs := <-srvCh

	require.NoError(t, err)
	require.Len(t, srvs, 1)
	require.Equal(t, "four", string(srvs[0].Addr))
}

func TestSelectServer_Cancel(t *testing.T) {
	m := newFakeMonitor("one", "two", "three")
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)

	go func() {
		_, err := selectServers(ctx, m, selectNone)
		errCh <- err
	}()

	time.Sleep(1 * time.Second)
	select {
	case <-errCh:
		t.Fatal("selectServers returned before it was cancelled")
	default:
		// this is what we expect
	}

	cancel()

	select {
	case <-time.After(1 * time.Second):
		t.Fatal("selectServers failed to return when cancelled")
	case err := <-errCh:
		// this is what we expect
		require.Equal(t, "server selection failed: context canceled", err.Error())
	}
}

func TestSelectServer_Timeout(t *testing.T) {
	m := newFakeMonitor("one", "two", "three")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	errCh := make(chan error)

	go func() {
		_, err := selectServers(ctx, m, selectNone)
		errCh <- err
	}()

	time.Sleep(1 * time.Second)
	select {
	case <-errCh:
		t.Fatal("selectServers returned before it was cancelled")
	default:
		// this is what we expect
	}

	select {
	case <-time.After(2 * time.Second):
		t.Fatal("selectServers failed to return when cancelled")
	case err := <-errCh:
		// this is what we expect
		require.Equal(t, "server selection failed: context deadline exceeded", err.Error())
	}
}

func TestSelectServer_Error(t *testing.T) {
	m := newFakeMonitor("one", "two", "three")
	errCh := make(chan error)

	go func() {
		_, err := selectServers(context.Background(), m, selectError)
		errCh <- err
	}()

	select {
	case <-time.After(1 * time.Second):
		t.Fatal("selectServers failed to return when selector returned error")
	case err := <-errCh:
		// this is what we expect
		require.Equal(t, "encountered an error in the selector", err.Error())
	}
}
