package topology

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/addr"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
)

func TestServerSelection(t *testing.T) {
	var selectFirst ServerSelectorFunc = func(_ description.Topology, candidates []description.Server) ([]description.Server, error) {
		if len(candidates) == 0 {
			return []description.Server{}, nil
		}
		return candidates[0:1], nil
	}
	var selectNone ServerSelectorFunc = func(description.Topology, []description.Server) ([]description.Server, error) {
		return []description.Server{}, nil
	}
	var errSelectionError = errors.New("encountered an error in the selector")
	var selectError ServerSelectorFunc = func(description.Topology, []description.Server) ([]description.Server, error) {
		return nil, errSelectionError
	}
	noerr := func(t *testing.T, err error) {
		if err != nil {
			t.Errorf("Unepexted error: %v", err)
			t.FailNow()
		}
	}

	t.Run("Success", func(t *testing.T) {
		topo, err := New()
		noerr(t, err)
		desc := description.Topology{
			Servers: []description.Server{
				{Addr: addr.Addr("one"), Kind: description.Standalone},
				{Addr: addr.Addr("two"), Kind: description.Standalone},
				{Addr: addr.Addr("three"), Kind: description.Standalone},
			},
		}
		subCh := make(chan description.Topology, 1)
		subCh <- desc
		srvs, err := topo.selectServer(context.Background(), subCh, selectFirst, nil)
		noerr(t, err)
		if len(srvs) != 1 {
			t.Errorf("Incorrect number of descriptions returned. got %d; want %d", len(srvs), 1)
		}
		if srvs[0].Addr != desc.Servers[0].Addr {
			t.Errorf("Incorrect sever selected. got %s; want %s", srvs[0].Addr, desc.Servers[0].Addr)
		}
	})
	t.Run("Updated", func(t *testing.T) {
		topo, err := New()
		noerr(t, err)
		desc := description.Topology{Servers: []description.Server{}}
		subCh := make(chan description.Topology, 1)
		subCh <- desc

		resp := make(chan []description.Server)
		go func() {
			srvs, err := topo.selectServer(context.Background(), subCh, selectFirst, nil)
			noerr(t, err)
			resp <- srvs
		}()

		desc = description.Topology{
			Servers: []description.Server{
				{Addr: addr.Addr("one"), Kind: description.Standalone},
				{Addr: addr.Addr("two"), Kind: description.Standalone},
				{Addr: addr.Addr("three"), Kind: description.Standalone},
			},
		}
		select {
		case subCh <- desc:
		case <-time.After(100 * time.Millisecond):
			t.Error("Timed out while trying to send topology description")
		}

		var srvs []description.Server
		select {
		case srvs = <-resp:
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Timed out while trying to retrieve selected servers")
		}

		if len(srvs) != 1 {
			t.Errorf("Incorrect number of descriptions returned. got %d; want %d", len(srvs), 1)
		}
		if srvs[0].Addr != desc.Servers[0].Addr {
			t.Errorf("Incorrect sever selected. got %s; want %s", srvs[0].Addr, desc.Servers[0].Addr)
		}
	})
	t.Run("Cancel", func(t *testing.T) {
		desc := description.Topology{
			Servers: []description.Server{
				{Addr: addr.Addr("one"), Kind: description.Standalone},
				{Addr: addr.Addr("two"), Kind: description.Standalone},
				{Addr: addr.Addr("three"), Kind: description.Standalone},
			},
		}
		topo, err := New()
		noerr(t, err)
		subCh := make(chan description.Topology, 1)
		subCh <- desc
		resp := make(chan error)
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			_, err := topo.selectServer(ctx, subCh, selectNone, nil)
			resp <- err
		}()

		select {
		case err := <-resp:
			t.Errorf("Received error from server selection too soon: %v", err)
		case <-time.After(100 * time.Millisecond):
		}

		cancel()

		select {
		case err = <-resp:
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Timed out while trying to retrieve selected servers")
		}

		if err != context.Canceled {
			t.Errorf("Incorrect error received. got %v; want %v", err, context.Canceled)
		}
	})
	t.Run("Timeout", func(t *testing.T) {
		desc := description.Topology{
			Servers: []description.Server{
				{Addr: addr.Addr("one"), Kind: description.Standalone},
				{Addr: addr.Addr("two"), Kind: description.Standalone},
				{Addr: addr.Addr("three"), Kind: description.Standalone},
			},
		}
		topo, err := New()
		noerr(t, err)
		subCh := make(chan description.Topology, 1)
		subCh <- desc
		resp := make(chan error)
		timeout := make(chan time.Time)
		go func() {
			_, err := topo.selectServer(context.Background(), subCh, selectNone, timeout)
			resp <- err
		}()

		select {
		case err := <-resp:
			t.Errorf("Received error from server selection too soon: %v", err)
		case timeout <- time.Now():
		}

		select {
		case err = <-resp:
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Timed out while trying to retrieve selected servers")
		}

		if err != ErrServerSelectionTimeout {
			t.Errorf("Incorrect error received. got %v; want %v", err, ErrServerSelectionTimeout)
		}
	})
	t.Run("Error", func(t *testing.T) {
		desc := description.Topology{
			Servers: []description.Server{
				{Addr: addr.Addr("one"), Kind: description.Standalone},
				{Addr: addr.Addr("two"), Kind: description.Standalone},
				{Addr: addr.Addr("three"), Kind: description.Standalone},
			},
		}
		topo, err := New()
		noerr(t, err)
		subCh := make(chan description.Topology, 1)
		subCh <- desc
		resp := make(chan error)
		timeout := make(chan time.Time)
		go func() {
			_, err := topo.selectServer(context.Background(), subCh, selectError, timeout)
			resp <- err
		}()

		select {
		case err = <-resp:
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Timed out while trying to retrieve selected servers")
		}

		if err != errSelectionError {
			t.Errorf("Incorrect error received. got %v; want %v", err, errSelectionError)
		}
	})
	t.Run("WriteSelector", func(t *testing.T) {
		testCases := []struct {
			name  string
			desc  description.Topology
			start int
			end   int
		}{
			{
				name: "ReplicaSetWithPrimary",
				desc: description.Topology{
					Kind: description.ReplicaSetWithPrimary,
					Servers: []description.Server{
						{Addr: addr.Addr("localhost:27017"), Kind: description.RSPrimary},
						{Addr: addr.Addr("localhost:27018"), Kind: description.RSSecondary},
						{Addr: addr.Addr("localhost:27019"), Kind: description.RSSecondary},
					},
				},
				start: 0,
				end:   1,
			},
			{
				name: "ReplicaSetNoPrimary",
				desc: description.Topology{
					Kind: description.ReplicaSetNoPrimary,
					Servers: []description.Server{
						{Addr: addr.Addr("localhost:27018"), Kind: description.RSSecondary},
						{Addr: addr.Addr("localhost:27019"), Kind: description.RSSecondary},
					},
				},
				start: 0,
				end:   0,
			},
			{
				name: "Sharded",
				desc: description.Topology{
					Kind: description.Sharded,
					Servers: []description.Server{
						{Addr: addr.Addr("localhost:27018"), Kind: description.Mongos},
						{Addr: addr.Addr("localhost:27019"), Kind: description.Mongos},
					},
				},
				start: 0,
				end:   2,
			},
			{
				name: "Single",
				desc: description.Topology{
					Kind: description.Single,
					Servers: []description.Server{
						{Addr: addr.Addr("localhost:27018"), Kind: description.Standalone},
					},
				},
				start: 0,
				end:   1,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := WriteSelector().SelectServer(tc.desc, tc.desc.Servers)
				noerr(t, err)
				if len(result) != tc.end-tc.start {
					t.Errorf("Incorrect number of servers selected. got %d; want %d", len(result), tc.end-tc.start)
				}
				if diff := cmp.Diff(result, tc.desc.Servers[tc.start:tc.end]); diff != "" {
					t.Errorf("Incorrect servers selected (-got +want):\n%s", diff)
				}
			})
		}
	})
	t.Run("LatencySelector", func(t *testing.T) {
		testCases := []struct {
			name  string
			desc  description.Topology
			start int
			end   int
		}{
			{
				name: "NoRTTSet",
				desc: description.Topology{
					Servers: []description.Server{
						{Addr: addr.Addr("localhost:27017")},
						{Addr: addr.Addr("localhost:27018")},
						{Addr: addr.Addr("localhost:27019")},
					},
				},
				start: 0,
				end:   3,
			},
			{
				name: "MultipleServers PartialNoRTTSet",
				desc: description.Topology{
					Servers: []description.Server{
						{Addr: addr.Addr("localhost:27017"), AverageRTT: 5 * time.Second, AverageRTTSet: true},
						{Addr: addr.Addr("localhost:27018"), AverageRTT: 10 * time.Second, AverageRTTSet: true},
						{Addr: addr.Addr("localhost:27019")},
					},
				},
				start: 0,
				end:   2,
			},
			{
				name: "MultipleServers",
				desc: description.Topology{
					Servers: []description.Server{
						{Addr: addr.Addr("localhost:27017"), AverageRTT: 5 * time.Second, AverageRTTSet: true},
						{Addr: addr.Addr("localhost:27018"), AverageRTT: 10 * time.Second, AverageRTTSet: true},
						{Addr: addr.Addr("localhost:27019"), AverageRTT: 26 * time.Second, AverageRTTSet: true},
					},
				},
				start: 0,
				end:   2,
			},
			{
				name:  "No Servers",
				desc:  description.Topology{Servers: []description.Server{}},
				start: 0,
				end:   0,
			},
			{
				name: "1 Server",
				desc: description.Topology{
					Servers: []description.Server{
						{Addr: addr.Addr("localhost:27017"), AverageRTT: 26 * time.Second, AverageRTTSet: true},
					},
				},
				start: 0,
				end:   1,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := LatencySelector(20*time.Second).SelectServer(tc.desc, tc.desc.Servers)
				noerr(t, err)
				if len(result) != tc.end-tc.start {
					t.Errorf("Incorrect number of servers selected. got %d; want %d", len(result), tc.end-tc.start)
				}
				if diff := cmp.Diff(result, tc.desc.Servers[tc.start:tc.end]); diff != "" {
					t.Errorf("Incorrect servers selected (-got +want):\n%s", diff)
				}
			})
		}
	})
}
