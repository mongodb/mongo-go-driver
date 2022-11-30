// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
)

const testTimeout = 2 * time.Second

func noerr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}
}

func compareErrors(err1, err2 error) bool {
	if err1 == nil && err2 == nil {
		return true
	}

	if err1 == nil || err2 == nil {
		return false
	}

	if err1.Error() != err2.Error() {
		return false
	}

	return true
}

func TestServerSelection(t *testing.T) {
	var selectFirst description.ServerSelectorFunc = func(_ description.Topology, candidates []description.Server) ([]description.Server, error) {
		if len(candidates) == 0 {
			return []description.Server{}, nil
		}
		return candidates[0:1], nil
	}
	var selectNone description.ServerSelectorFunc = func(description.Topology, []description.Server) ([]description.Server, error) {
		return []description.Server{}, nil
	}
	var errSelectionError = errors.New("encountered an error in the selector")
	var selectError description.ServerSelectorFunc = func(description.Topology, []description.Server) ([]description.Server, error) {
		return nil, errSelectionError
	}

	t.Run("Success", func(t *testing.T) {
		topo, err := New(nil)
		noerr(t, err)
		desc := description.Topology{
			Servers: []description.Server{
				{Addr: address.Address("one"), Kind: description.Standalone},
				{Addr: address.Address("two"), Kind: description.Standalone},
				{Addr: address.Address("three"), Kind: description.Standalone},
			},
		}
		subCh := make(chan description.Topology, 1)
		subCh <- desc

		state := newServerSelectionState(selectFirst, nil)
		srvs, err := topo.selectServerFromSubscription(context.Background(), subCh, state)
		noerr(t, err)
		if len(srvs) != 1 {
			t.Errorf("Incorrect number of descriptions returned. got %d; want %d", len(srvs), 1)
		}
		if srvs[0].Addr != desc.Servers[0].Addr {
			t.Errorf("Incorrect sever selected. got %s; want %s", srvs[0].Addr, desc.Servers[0].Addr)
		}
	})
	t.Run("Compatibility Error Min Version Too High", func(t *testing.T) {
		topo, err := New(nil)
		noerr(t, err)
		desc := description.Topology{
			Kind: description.Single,
			Servers: []description.Server{
				{Addr: address.Address("one:27017"), Kind: description.Standalone, WireVersion: &description.VersionRange{Max: 11, Min: 11}},
				{Addr: address.Address("two:27017"), Kind: description.Standalone, WireVersion: &description.VersionRange{Max: 9, Min: 2}},
				{Addr: address.Address("three:27017"), Kind: description.Standalone, WireVersion: &description.VersionRange{Max: 9, Min: 2}},
			},
		}
		want := fmt.Errorf(
			"server at %s requires wire version %d, but this version of the Go driver only supports up to %d",
			desc.Servers[0].Addr.String(),
			desc.Servers[0].WireVersion.Min,
			SupportedWireVersions.Max,
		)
		desc.CompatibilityErr = want
		atomic.StoreInt64(&topo.state, topologyConnected)
		topo.desc.Store(desc)
		_, err = topo.SelectServer(context.Background(), selectFirst)
		assert.Equal(t, err, want, "expected %v, got %v", want, err)
	})
	t.Run("Compatibility Error Max Version Too Low", func(t *testing.T) {
		topo, err := New(nil)
		noerr(t, err)
		desc := description.Topology{
			Kind: description.Single,
			Servers: []description.Server{
				{Addr: address.Address("one:27017"), Kind: description.Standalone, WireVersion: &description.VersionRange{Max: 1, Min: 1}},
				{Addr: address.Address("two:27017"), Kind: description.Standalone, WireVersion: &description.VersionRange{Max: 9, Min: 2}},
				{Addr: address.Address("three:27017"), Kind: description.Standalone, WireVersion: &description.VersionRange{Max: 9, Min: 2}},
			},
		}
		want := fmt.Errorf(
			"server at %s reports wire version %d, but this version of the Go driver requires "+
				"at least 6 (MongoDB 3.6)",
			desc.Servers[0].Addr.String(),
			desc.Servers[0].WireVersion.Max,
		)
		desc.CompatibilityErr = want
		atomic.StoreInt64(&topo.state, topologyConnected)
		topo.desc.Store(desc)
		_, err = topo.SelectServer(context.Background(), selectFirst)
		assert.Equal(t, err, want, "expected %v, got %v", want, err)
	})
	t.Run("Updated", func(t *testing.T) {
		topo, err := New(nil)
		noerr(t, err)
		desc := description.Topology{Servers: []description.Server{}}
		subCh := make(chan description.Topology, 1)
		subCh <- desc

		resp := make(chan []description.Server)
		go func() {
			state := newServerSelectionState(selectFirst, nil)
			srvs, err := topo.selectServerFromSubscription(context.Background(), subCh, state)
			noerr(t, err)
			resp <- srvs
		}()

		desc = description.Topology{
			Servers: []description.Server{
				{Addr: address.Address("one"), Kind: description.Standalone},
				{Addr: address.Address("two"), Kind: description.Standalone},
				{Addr: address.Address("three"), Kind: description.Standalone},
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
				{Addr: address.Address("one"), Kind: description.Standalone},
				{Addr: address.Address("two"), Kind: description.Standalone},
				{Addr: address.Address("three"), Kind: description.Standalone},
			},
		}
		topo, err := New(nil)
		noerr(t, err)
		subCh := make(chan description.Topology, 1)
		subCh <- desc
		resp := make(chan error)
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			state := newServerSelectionState(selectNone, nil)
			_, err := topo.selectServerFromSubscription(ctx, subCh, state)
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

		want := ServerSelectionError{Wrapped: context.Canceled, Desc: desc}
		assert.Equal(t, err, want, "Incorrect error received. got %v; want %v", err, want)
	})
	t.Run("Timeout", func(t *testing.T) {
		desc := description.Topology{
			Servers: []description.Server{
				{Addr: address.Address("one"), Kind: description.Standalone},
				{Addr: address.Address("two"), Kind: description.Standalone},
				{Addr: address.Address("three"), Kind: description.Standalone},
			},
		}
		topo, err := New(nil)
		noerr(t, err)
		subCh := make(chan description.Topology, 1)
		subCh <- desc
		resp := make(chan error)
		timeout := make(chan time.Time)
		go func() {
			state := newServerSelectionState(selectNone, timeout)
			_, err := topo.selectServerFromSubscription(context.Background(), subCh, state)
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

		if err == nil {
			t.Fatalf("did not receive error from server selection")
		}
	})
	t.Run("Error", func(t *testing.T) {
		desc := description.Topology{
			Servers: []description.Server{
				{Addr: address.Address("one"), Kind: description.Standalone},
				{Addr: address.Address("two"), Kind: description.Standalone},
				{Addr: address.Address("three"), Kind: description.Standalone},
			},
		}
		topo, err := New(nil)
		noerr(t, err)
		subCh := make(chan description.Topology, 1)
		subCh <- desc
		resp := make(chan error)
		timeout := make(chan time.Time)
		go func() {
			state := newServerSelectionState(selectError, timeout)
			_, err := topo.selectServerFromSubscription(context.Background(), subCh, state)
			resp <- err
		}()

		select {
		case err = <-resp:
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Timed out while trying to retrieve selected servers")
		}

		if err == nil {
			t.Fatalf("did not receive error from server selection")
		}
	})
	t.Run("findServer returns topology kind", func(t *testing.T) {
		topo, err := New(nil)
		noerr(t, err)
		atomic.StoreInt64(&topo.state, topologyConnected)
		srvr, err := ConnectServer(address.Address("one"), topo.updateCallback, topo.id)
		noerr(t, err)
		topo.servers[address.Address("one")] = srvr
		desc := topo.desc.Load().(description.Topology)
		desc.Kind = description.Single
		topo.desc.Store(desc)

		selected := description.Server{Addr: address.Address("one")}

		ss, err := topo.FindServer(selected)
		noerr(t, err)
		if ss.Kind != description.Single {
			t.Errorf("findServer does not properly set the topology description kind. got %v; want %v", ss.Kind, description.Single)
		}
	})
	t.Run("Update on not primary error", func(t *testing.T) {
		topo, err := New(nil)
		noerr(t, err)
		atomic.StoreInt64(&topo.state, topologyConnected)

		addr1 := address.Address("one")
		addr2 := address.Address("two")
		addr3 := address.Address("three")
		desc := description.Topology{
			Servers: []description.Server{
				{Addr: addr1, Kind: description.RSPrimary},
				{Addr: addr2, Kind: description.RSSecondary},
				{Addr: addr3, Kind: description.RSSecondary},
			},
		}

		// manually add the servers to the topology
		for _, srv := range desc.Servers {
			s, err := ConnectServer(srv.Addr, topo.updateCallback, topo.id)
			noerr(t, err)
			topo.servers[srv.Addr] = s
		}

		// Send updated description
		desc = description.Topology{
			Servers: []description.Server{
				{Addr: addr1, Kind: description.RSSecondary},
				{Addr: addr2, Kind: description.RSPrimary},
				{Addr: addr3, Kind: description.RSSecondary},
			},
		}

		subCh := make(chan description.Topology, 1)
		subCh <- desc

		// send a not primary error to the server forcing an update
		serv, err := topo.FindServer(desc.Servers[0])
		noerr(t, err)
		atomic.StoreInt64(&serv.state, serverConnected)
		_ = serv.ProcessError(driver.Error{Message: internal.LegacyNotPrimary}, initConnection{})

		resp := make(chan []description.Server)

		go func() {
			// server selection should discover the new topology
			state := newServerSelectionState(description.WriteSelector(), nil)
			srvs, err := topo.selectServerFromSubscription(context.Background(), subCh, state)
			noerr(t, err)
			resp <- srvs
		}()

		var srvs []description.Server
		select {
		case srvs = <-resp:
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Timed out while trying to retrieve selected servers")
		}

		if len(srvs) != 1 {
			t.Errorf("Incorrect number of descriptions returned. got %d; want %d", len(srvs), 1)
		}
		if srvs[0].Addr != desc.Servers[1].Addr {
			t.Errorf("Incorrect sever selected. got %s; want %s", srvs[0].Addr, desc.Servers[1].Addr)
		}
	})
	t.Run("fast path does not subscribe or check timeouts", func(t *testing.T) {
		// Assert that the server selection fast path does not create a Subscription or check for timeout errors.
		topo, err := New(nil)
		noerr(t, err)
		atomic.StoreInt64(&topo.state, topologyConnected)

		primaryAddr := address.Address("one")
		desc := description.Topology{
			Servers: []description.Server{
				{Addr: primaryAddr, Kind: description.RSPrimary},
			},
		}
		topo.desc.Store(desc)
		for _, srv := range desc.Servers {
			s, err := ConnectServer(srv.Addr, topo.updateCallback, topo.id)
			noerr(t, err)
			topo.servers[srv.Addr] = s
		}

		// Manually close subscriptions so calls to Subscribe will error and pass in a cancelled context to ensure the
		// fast path ignores timeout errors.
		topo.subscriptionsClosed = true
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		selectedServer, err := topo.SelectServer(ctx, description.WriteSelector())
		noerr(t, err)
		selectedAddr := selectedServer.(*SelectedServer).address
		assert.Equal(t, primaryAddr, selectedAddr, "expected address %v, got %v", primaryAddr, selectedAddr)
	})
	t.Run("default to selecting from subscription if fast path fails", func(t *testing.T) {
		topo, err := New(nil)
		noerr(t, err)

		atomic.StoreInt64(&topo.state, topologyConnected)
		desc := description.Topology{
			Servers: []description.Server{},
		}
		topo.desc.Store(desc)

		topo.subscriptionsClosed = true
		_, err = topo.SelectServer(context.Background(), description.WriteSelector())
		assert.Equal(t, ErrSubscribeAfterClosed, err, "expected error %v, got %v", ErrSubscribeAfterClosed, err)
	})
}

func TestSessionTimeout(t *testing.T) {
	t.Run("UpdateSessionTimeout", func(t *testing.T) {
		topo, err := New(nil)
		noerr(t, err)
		topo.servers["foo"] = nil
		topo.fsm.Servers = []description.Server{
			{Addr: address.Address("foo").Canonicalize(), Kind: description.RSPrimary, SessionTimeoutMinutes: 60},
		}

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		desc := description.Server{
			Addr:                  "foo",
			Kind:                  description.RSPrimary,
			SessionTimeoutMinutes: 30,
		}
		topo.apply(ctx, desc)

		currDesc := topo.desc.Load().(description.Topology)
		if currDesc.SessionTimeoutMinutes != 30 {
			t.Errorf("session timeout minutes mismatch. got: %d. expected: 30", currDesc.SessionTimeoutMinutes)
		}
	})
	t.Run("MultipleUpdates", func(t *testing.T) {
		topo, err := New(nil)
		noerr(t, err)
		topo.fsm.Kind = description.ReplicaSetWithPrimary
		topo.servers["foo"] = nil
		topo.servers["bar"] = nil
		topo.fsm.Servers = []description.Server{
			{Addr: address.Address("foo").Canonicalize(), Kind: description.RSPrimary, SessionTimeoutMinutes: 60},
			{Addr: address.Address("bar").Canonicalize(), Kind: description.RSSecondary, SessionTimeoutMinutes: 60},
		}

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		desc1 := description.Server{
			Addr:                  "foo",
			Kind:                  description.RSPrimary,
			SessionTimeoutMinutes: 30,
			Members:               []address.Address{address.Address("foo").Canonicalize(), address.Address("bar").Canonicalize()},
		}
		// should update because new timeout is lower
		desc2 := description.Server{
			Addr:                  "bar",
			Kind:                  description.RSPrimary,
			SessionTimeoutMinutes: 20,
			Members:               []address.Address{address.Address("foo").Canonicalize(), address.Address("bar").Canonicalize()},
		}
		topo.apply(ctx, desc1)
		topo.apply(ctx, desc2)

		currDesc := topo.Description()
		if currDesc.SessionTimeoutMinutes != 20 {
			t.Errorf("session timeout minutes mismatch. got: %d. expected: 20", currDesc.SessionTimeoutMinutes)
		}
	})
	t.Run("NoUpdate", func(t *testing.T) {
		topo, err := New(nil)
		noerr(t, err)
		topo.servers["foo"] = nil
		topo.servers["bar"] = nil
		topo.fsm.Servers = []description.Server{
			{Addr: address.Address("foo").Canonicalize(), Kind: description.RSPrimary, SessionTimeoutMinutes: 60},
			{Addr: address.Address("bar").Canonicalize(), Kind: description.RSSecondary, SessionTimeoutMinutes: 60},
		}

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		desc1 := description.Server{
			Addr:                  "foo",
			Kind:                  description.RSPrimary,
			SessionTimeoutMinutes: 20,
			Members:               []address.Address{address.Address("foo").Canonicalize(), address.Address("bar").Canonicalize()},
		}
		// should not update because new timeout is higher
		desc2 := description.Server{
			Addr:                  "bar",
			Kind:                  description.RSPrimary,
			SessionTimeoutMinutes: 30,
			Members:               []address.Address{address.Address("foo").Canonicalize(), address.Address("bar").Canonicalize()},
		}
		topo.apply(ctx, desc1)
		topo.apply(ctx, desc2)

		currDesc := topo.desc.Load().(description.Topology)
		if currDesc.SessionTimeoutMinutes != 20 {
			t.Errorf("session timeout minutes mismatch. got: %d. expected: 20", currDesc.SessionTimeoutMinutes)
		}
	})
	t.Run("TimeoutDataBearing", func(t *testing.T) {
		topo, err := New(nil)
		noerr(t, err)
		topo.servers["foo"] = nil
		topo.servers["bar"] = nil
		topo.fsm.Servers = []description.Server{
			{Addr: address.Address("foo").Canonicalize(), Kind: description.RSPrimary, SessionTimeoutMinutes: 60},
			{Addr: address.Address("bar").Canonicalize(), Kind: description.RSSecondary, SessionTimeoutMinutes: 60},
		}

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		desc1 := description.Server{
			Addr:                  "foo",
			Kind:                  description.RSPrimary,
			SessionTimeoutMinutes: 20,
			Members:               []address.Address{address.Address("foo").Canonicalize(), address.Address("bar").Canonicalize()},
		}
		// should not update because not a data bearing server
		desc2 := description.Server{
			Addr:                  "bar",
			Kind:                  description.Unknown,
			SessionTimeoutMinutes: 10,
			Members:               []address.Address{address.Address("foo").Canonicalize(), address.Address("bar").Canonicalize()},
		}
		topo.apply(ctx, desc1)
		topo.apply(ctx, desc2)

		currDesc := topo.desc.Load().(description.Topology)
		if currDesc.SessionTimeoutMinutes != 20 {
			t.Errorf("session timeout minutes mismatch. got: %d. expected: 20", currDesc.SessionTimeoutMinutes)
		}
	})
	t.Run("MixedSessionSupport", func(t *testing.T) {
		topo, err := New(nil)
		noerr(t, err)
		topo.fsm.Kind = description.ReplicaSetWithPrimary
		topo.servers["one"] = nil
		topo.servers["two"] = nil
		topo.servers["three"] = nil
		topo.fsm.Servers = []description.Server{
			{Addr: address.Address("one").Canonicalize(), Kind: description.RSPrimary, SessionTimeoutMinutes: 20},
			{Addr: address.Address("two").Canonicalize(), Kind: description.RSSecondary}, // does not support sessions
			{Addr: address.Address("three").Canonicalize(), Kind: description.RSPrimary, SessionTimeoutMinutes: 60},
		}

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		desc := description.Server{
			Addr: address.Address("three"), Kind: description.RSSecondary, SessionTimeoutMinutes: 30}
		topo.apply(ctx, desc)

		currDesc := topo.desc.Load().(description.Topology)
		if currDesc.SessionTimeoutMinutes != 0 {
			t.Errorf("session timeout minutes mismatch. got: %d. expected: 0", currDesc.SessionTimeoutMinutes)
		}
	})
}

func TestMinPoolSize(t *testing.T) {
	cfg, err := NewConfig(options.Client().SetHosts([]string{"localhost:27017"}).SetMinPoolSize(10), nil)
	if err != nil {
		t.Errorf("error constructing topology config: %v", err)
	}

	topo, err := New(cfg)
	if err != nil {
		t.Errorf("topology.New shouldn't error. got: %v", err)
	}
	err = topo.Connect()
	if err != nil {
		t.Errorf("topology.Connect shouldn't error. got: %v", err)
	}
}

func TestTopology_String_Race(t *testing.T) {
	ch := make(chan bool)
	topo := &Topology{
		servers: make(map[address.Address]*Server),
	}

	go func() {
		topo.serversLock.Lock()
		srv := &Server{}
		srv.desc.Store(description.Server{})
		topo.servers[address.Address("127.0.0.1:27017")] = srv
		topo.serversLock.Unlock()
		ch <- true
	}()

	go func() {
		_ = topo.String()
		ch <- true
	}()

	<-ch
	<-ch
}

func TestTopologyConstruction(t *testing.T) {
	t.Run("construct with URI", func(t *testing.T) {
		testCases := []struct {
			name            string
			uri             string
			pollingRequired bool
		}{
			{"normal", "mongodb://localhost:27017", false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cfg, err := NewConfig(options.Client().ApplyURI(tc.uri), nil)
				assert.Nil(t, err, "error constructing topology config: %v", err)

				topo, err := New(cfg)
				assert.Nil(t, err, "topology.New error: %v", err)

				assert.Equal(t, tc.uri, topo.cfg.URI, "expected topology URI to be %v, got %v", tc.uri, topo.cfg.URI)
				assert.Equal(t, tc.pollingRequired, topo.pollingRequired,
					"expected topo.pollingRequired to be %v, got %v", tc.pollingRequired, topo.pollingRequired)
			})
		}
	})
}

type inWindowServer struct {
	Address  string `json:"address"`
	Type     string `json:"type"`
	AvgRTTMS int64  `json:"avg_rtt_ms"`
}

type inWindowTopology struct {
	Type    string           `json:"type"`
	Servers []inWindowServer `json:"servers"`
}

type inWindowOutcome struct {
	Tolerance           float64            `json:"tolerance"`
	ExpectedFrequencies map[string]float64 `json:"expected_frequencies"`
}

type inWindowTopologyState struct {
	Address        string `json:"address"`
	OperationCount int64  `json:"operation_count"`
}

type inWindowTestCase struct {
	TopologyDescription inWindowTopology        `json:"topology_description"`
	MockedTopologyState []inWindowTopologyState `json:"mocked_topology_state"`
	Iterations          int                     `json:"iterations"`
	Outcome             inWindowOutcome         `json:"outcome"`
}

// TestServerSelectionSpecInWindow runs the "in_window" server selection spec tests. This test is
// in the "topology" package instead of the "description" package (where the rest of the server
// selection spec tests are) because it primarily tests load-based server selection. Load-based
// server selection is implemented in Topology.SelectServer() because it requires knowledge of the
// current "operation count" (the number of currently running operations) for each server, so it
// can't be effectively accomplished just with server descriptions like most other server selection
// algorithms.
func TestServerSelectionSpecInWindow(t *testing.T) {
	const testsDir = "../../../../testdata/server-selection/in_window"

	files := helpers.FindJSONFilesInDir(t, testsDir)

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			runInWindowTest(t, testsDir, file)
		})
	}
}

func runInWindowTest(t *testing.T, directory string, filename string) {
	filepath := path.Join(directory, filename)
	content, err := ioutil.ReadFile(filepath)
	require.NoError(t, err)

	var test inWindowTestCase
	require.NoError(t, json.Unmarshal(content, &test))

	// For each server described in the test's "topology_description", create both a *Server and
	// description.Server, which are both required to run Topology.SelectServer().
	servers := make(map[string]*Server, len(test.TopologyDescription.Servers))
	descriptions := make([]description.Server, 0, len(test.TopologyDescription.Servers))
	for _, testDesc := range test.TopologyDescription.Servers {
		server := NewServer(
			address.Address(testDesc.Address),
			primitive.NilObjectID,
			withMonitoringDisabled(func(bool) bool { return true }))
		servers[testDesc.Address] = server

		desc := description.Server{
			Kind:          serverKindFromString(t, testDesc.Type),
			Addr:          address.Address(testDesc.Address),
			AverageRTT:    time.Duration(testDesc.AvgRTTMS) * time.Millisecond,
			AverageRTTSet: true,
		}

		if testDesc.AvgRTTMS > 0 {
			desc.AverageRTT = time.Duration(testDesc.AvgRTTMS) * time.Millisecond
			desc.AverageRTTSet = true
		}

		descriptions = append(descriptions, desc)
	}

	// For each server state in the test's "mocked_topology_state", set the connection pool's
	// in-use connections count to the test operation count value.
	for _, state := range test.MockedTopologyState {
		servers[state.Address].operationCount = state.OperationCount
	}

	// Create a new Topology, set the state to "connected", store a topology description
	// containing all server descriptions created from the test server descriptions, and copy
	// all *Server instances to the Topology's servers list.
	topology, err := New(nil)
	require.NoError(t, err, "error creating new Topology")
	topology.state = topologyConnected
	topology.desc.Store(description.Topology{
		Kind:    topologyKindFromString(t, test.TopologyDescription.Type),
		Servers: descriptions,
	})
	for addr, server := range servers {
		topology.servers[address.Address(addr)] = server
	}

	// Run server selection the required number of times and record how many times each server
	// address was selected.
	counts := make(map[string]int, len(test.TopologyDescription.Servers))
	for i := 0; i < test.Iterations; i++ {
		selected, err := topology.SelectServer(
			context.Background(),
			description.ReadPrefSelector(readpref.Nearest()))
		require.NoError(t, err, "error selecting server")
		counts[string(selected.(*SelectedServer).address)]++
	}

	// Convert the server selection counts to selection frequencies by dividing the counts by
	// the total number of server selection attempts.
	frequencies := make(map[string]float64, len(counts))
	for addr, count := range counts {
		frequencies[addr] = float64(count) / float64(test.Iterations)
	}

	// Assert that the observed server selection frequency for each server address matches the
	// expected server selection frequency.
	for addr, expected := range test.Outcome.ExpectedFrequencies {
		actual := frequencies[addr]

		// If the expected frequency for a given server is 1 or 0, then the observed frequency
		// MUST be exactly equal to the expected one.
		if expected == 1 || expected == 0 {
			assert.Equal(
				t,
				expected,
				actual,
				"expected frequency of %q to be equal to %f, but is %f",
				addr, expected, actual)
			continue
		}

		// Otherwise, check if the expected frequency is within the given tolerance range.
		// TODO(GODRIVER-2179): Use assert.Deltaf() when we migrate all test code to the "testify/assert" or an
		// TODO API-compatible library for assertions.
		low := expected - test.Outcome.Tolerance
		high := expected + test.Outcome.Tolerance
		assert.True(
			t,
			actual >= low && actual <= high,
			"expected frequency of %q to be in range [%f, %f], but is %f",
			addr, low, high, actual)
	}
}

func topologyKindFromString(t *testing.T, s string) description.TopologyKind {
	t.Helper()

	switch s {
	case "Single":
		return description.Single
	case "ReplicaSet":
		return description.ReplicaSet
	case "ReplicaSetNoPrimary":
		return description.ReplicaSetNoPrimary
	case "ReplicaSetWithPrimary":
		return description.ReplicaSetWithPrimary
	case "Sharded":
		return description.Sharded
	case "LoadBalanced":
		return description.LoadBalanced
	case "Unknown":
		return description.Unknown
	default:
		t.Fatalf("unrecognized topology kind: %q", s)
	}

	return description.Unknown
}

func serverKindFromString(t *testing.T, s string) description.ServerKind {
	t.Helper()

	switch s {
	case "Standalone":
		return description.Standalone
	case "RSOther":
		return description.RSMember
	case "RSPrimary":
		return description.RSPrimary
	case "RSSecondary":
		return description.RSSecondary
	case "RSArbiter":
		return description.RSArbiter
	case "RSGhost":
		return description.RSGhost
	case "Mongos":
		return description.Mongos
	case "LoadBalancer":
		return description.LoadBalancer
	case "PossiblePrimary", "Unknown":
		// Go does not have a PossiblePrimary server type and per the SDAM spec, this type is synonymous with Unknown.
		return description.Unknown
	default:
		t.Fatalf("unrecognized server kind: %q", s)
	}

	return description.Unknown
}

func BenchmarkSelectServerFromDescription(b *testing.B) {
	for _, bcase := range []struct {
		name        string
		serversHook func(servers []description.Server)
	}{
		{
			name:        "AllFit",
			serversHook: func(servers []description.Server) {},
		},
		{
			name: "AllButOneFit",
			serversHook: func(servers []description.Server) {
				servers[0].Kind = description.Unknown
			},
		},
		{
			name: "HalfFit",
			serversHook: func(servers []description.Server) {
				for i := 0; i < len(servers); i += 2 {
					servers[i].Kind = description.Unknown
				}
			},
		},
		{
			name: "OneFit",
			serversHook: func(servers []description.Server) {
				for i := 1; i < len(servers); i++ {
					servers[i].Kind = description.Unknown
				}
			},
		},
	} {
		bcase := bcase

		b.Run(bcase.name, func(b *testing.B) {
			s := description.Server{
				Addr:              address.Address("localhost:27017"),
				HeartbeatInterval: time.Duration(10) * time.Second,
				LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
				LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
				Kind:              description.Mongos,
				WireVersion:       &description.VersionRange{Min: 0, Max: 5},
			}
			servers := make([]description.Server, 100)
			for i := 0; i < len(servers); i++ {
				servers[i] = s
			}
			bcase.serversHook(servers)
			desc := description.Topology{
				Servers: servers,
			}

			timeout := make(chan time.Time)
			b.ResetTimer()
			b.RunParallel(func(p *testing.PB) {
				b.ReportAllocs()
				for p.Next() {
					var c Topology
					_, _ = c.selectServerFromDescription(desc, newServerSelectionState(selectNone, timeout))
				}
			})
		})
	}
}
