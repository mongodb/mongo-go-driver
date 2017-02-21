package cluster

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/server"
	"github.com/stretchr/testify/require"
)

func selectFirst(_ *Desc, servers []*server.Desc) ([]*server.Desc, error) {
	if len(servers) == 0 {
		return []*server.Desc{}, nil
	}
	return servers[0:1], nil
}

func selectNone(_ *Desc, _ []*server.Desc) ([]*server.Desc, error) {
	return []*server.Desc{}, nil
}

func selectError(_ *Desc, _ []*server.Desc) ([]*server.Desc, error) {
	return nil, errors.New("encountered an error in the selector")
}

type fakeMonitor struct {
	updates chan *Desc
}

func newFakeMonitor(endpoints ...string) *fakeMonitor {
	updates := make(chan *Desc, 1)
	monitor := &fakeMonitor{
		updates: updates,
	}
	monitor.updateEndpoints(endpoints...)
	return monitor
}

func (f *fakeMonitor) Subscribe() (<-chan *Desc, func(), error) {
	return f.updates, func() {}, nil
}

func (f *fakeMonitor) RequestImmediateCheck() {
	// no-op
}

func (f *fakeMonitor) updateEndpoints(endpoints ...string) {
	servers := make([]*server.Desc, len(endpoints))
	for i, end := range endpoints {
		endpoint := conn.Endpoint(end)
		server := &server.Desc{
			Endpoint: endpoint,
			Type:     server.Standalone,
		}
		servers[i] = server
	}
	clusterDesc := &Desc{
		Servers: servers,
	}

	select {
	case <-f.updates:
	default:
	}
	f.updates <- clusterDesc
}

func TestSelectServer_Success(t *testing.T) {
	m := newFakeMonitor("one", "two", "three")
	srvs, err := selectServers(context.Background(), m, selectFirst)
	require.NoError(t, err)
	require.Len(t, srvs, 1)
	require.Equal(t, "one", string(srvs[0].Endpoint))
}

func TestSelectServer_Updated(t *testing.T) {
	m := newFakeMonitor()
	errCh := make(chan error)
	srvCh := make(chan []*server.Desc)

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
	require.Equal(t, "four", string(srvs[0].Endpoint))
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
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
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
