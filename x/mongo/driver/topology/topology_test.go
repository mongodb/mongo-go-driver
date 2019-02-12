// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/x/network/address"
	"go.mongodb.org/mongo-driver/x/network/command"
	"go.mongodb.org/mongo-driver/x/network/description"
)

const testTimeout = 2 * time.Second

func noerr(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}
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
		topo, err := New()
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
				{Addr: address.Address("one"), Kind: description.Standalone},
				{Addr: address.Address("two"), Kind: description.Standalone},
				{Addr: address.Address("three"), Kind: description.Standalone},
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

		if err == nil {
			t.Fatalf("did not receive error from server selection")
		}
	})
	t.Run("findServer returns topology kind", func(t *testing.T) {
		topo, err := New()
		noerr(t, err)
		atomic.StoreInt32(&topo.connectionstate, connected)
		srvr, err := NewServer(address.Address("one"))
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
	t.Run("Update on not master error", func(t *testing.T) {
		topo, err := New()
		noerr(t, err)
		topo.cfg.cs.HeartbeatInterval = time.Minute
		atomic.StoreInt32(&topo.connectionstate, connected)

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
			s, err := NewServer(srv.Addr)
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

		// send a not master error to the server forcing an update
		serv, err := topo.FindServer(desc.Servers[0])
		noerr(t, err)
		err = serv.pool.Connect(context.Background())
		noerr(t, err)
		atomic.StoreInt32(&serv.connectionstate, connected)
		sc := &sconn{s: serv.Server}
		sc.processErr(command.Error{Message: "not master"})

		resp := make(chan []description.Server)

		go func() {
			// server selection should discover the new topology
			srvs, err := topo.selectServer(context.Background(), subCh, description.WriteSelector(), nil)
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
}

func TestSessionTimeout(t *testing.T) {
	t.Run("UpdateSessionTimeout", func(t *testing.T) {
		topo, err := New()
		noerr(t, err)
		doneCh := make(chan struct{}, 1)

		// update topology and signal when done
		go func() {
			topo.changeswg.Add(1)
			topo.update()
			doneCh <- struct{}{}
		}()

		timeoutChan := time.After(testTimeout)
		topo.changes <- description.Server{
			Kind:                  description.RSPrimary,
			SessionTimeoutMinutes: 30,
		}
		topo.done <- struct{}{}

		select {
		case <-doneCh:
			currDesc := topo.desc.Load().(description.Topology)
			if currDesc.SessionTimeoutMinutes != 30 {
				t.Errorf("session timeout minutes mismatch. got: %d. expected: 30", currDesc.SessionTimeoutMinutes)
			}
		case <-timeoutChan:
			t.Errorf("test case timed out")
		}
	})
	t.Run("MultipleUpdates", func(t *testing.T) {
		topo, err := New()
		noerr(t, err)
		doneCh := make(chan struct{}, 2)

		// update topology and signal when done
		go func() {
			topo.changeswg.Add(1)
			topo.update()
			doneCh <- struct{}{}
		}()

		timeoutChan := time.After(testTimeout)
		topo.changes <- description.Server{
			Kind:                  description.RSPrimary,
			SessionTimeoutMinutes: 30,
		}
		// should update because new timeout is lower
		topo.changes <- description.Server{
			Kind:                  description.RSPrimary,
			SessionTimeoutMinutes: 20,
		}
		topo.done <- struct{}{}

		select {
		case <-doneCh:
			currDesc := topo.Description()
			if currDesc.SessionTimeoutMinutes != 20 {
				t.Errorf("session timeout minutes mismatch. got: %d. expected: 20", currDesc.SessionTimeoutMinutes)
			}
		case <-timeoutChan:
			t.Errorf("test case timed out")
		}
	})
	t.Run("NoUpdate", func(t *testing.T) {
		topo, err := New()
		noerr(t, err)
		doneCh := make(chan struct{}, 2)

		// update topology and signal when done
		go func() {
			topo.changeswg.Add(1)
			topo.update()
			doneCh <- struct{}{}
		}()

		timeoutChan := time.After(testTimeout)
		topo.changes <- description.Server{
			Kind:                  description.RSPrimary,
			SessionTimeoutMinutes: 20,
		}
		// should not update because new timeout is higher
		topo.changes <- description.Server{
			Kind:                  description.RSPrimary,
			SessionTimeoutMinutes: 30,
		}
		topo.done <- struct{}{}

		select {
		case <-doneCh:
			currDesc := topo.desc.Load().(description.Topology)
			if currDesc.SessionTimeoutMinutes != 20 {
				t.Errorf("session timeout minutes mismatch. got: %d. expected: 20", currDesc.SessionTimeoutMinutes)
			}
		case <-timeoutChan:
			t.Errorf("test case timed out")
		}
	})
	t.Run("TimeoutDataBearing", func(t *testing.T) {
		topo, err := New()
		noerr(t, err)
		doneCh := make(chan struct{}, 2)

		// update topology and signal when done
		go func() {
			topo.changeswg.Add(1)
			topo.update()
			doneCh <- struct{}{}
		}()

		timeoutChan := time.After(testTimeout)
		topo.changes <- description.Server{
			Kind:                  description.RSPrimary,
			SessionTimeoutMinutes: 20,
		}
		// should not update because not a data bearing server
		topo.changes <- description.Server{
			Kind:                  description.Unknown,
			SessionTimeoutMinutes: 10,
		}
		topo.done <- struct{}{}

		select {
		case <-doneCh:
			currDesc := topo.desc.Load().(description.Topology)
			if currDesc.SessionTimeoutMinutes != 20 {
				t.Errorf("session timeout minutes mismatch. got: %d. expected: 20", currDesc.SessionTimeoutMinutes)
			}
		case <-timeoutChan:
			t.Errorf("test case timed out")
		}
	})
	t.Run("MixedSessionSupport", func(t *testing.T) {
		topo, err := New()
		noerr(t, err)
		doneCh := make(chan struct{}, 2)

		topo.fsm.Kind = description.ReplicaSetWithPrimary
		topo.fsm.Servers = []description.Server{
			{Addr: address.Address("one"), Kind: description.RSPrimary, SessionTimeoutMinutes: 20},
			{Addr: address.Address("two"), Kind: description.RSSecondary}, // does not support sessions
			{Addr: address.Address("three"), Kind: description.RSSecondary, SessionTimeoutMinutes: 30},
		}

		// update topology and signal when done
		go func() {
			topo.changeswg.Add(1)
			topo.update()
			doneCh <- struct{}{}
		}()

		topo.done <- struct{}{}

		<-doneCh
		currDesc := topo.desc.Load().(description.Topology)
		if currDesc.SessionTimeoutMinutes != 0 {
			t.Errorf("session timeout minutes mismatch. got: %d. expected: 0", currDesc.SessionTimeoutMinutes)
		}
	})
}
