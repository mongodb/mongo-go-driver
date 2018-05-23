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

	"github.com/mongodb/mongo-go-driver/core/address"
	"github.com/mongodb/mongo-go-driver/core/description"
)

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

		if err != ErrServerSelectionTimeout {
			t.Errorf("Incorrect error received. got %v; want %v", err, ErrServerSelectionTimeout)
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

		if err != errSelectionError {
			t.Errorf("Incorrect error received. got %v; want %v", err, errSelectionError)
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

		ss, err := topo.findServer(selected)
		noerr(t, err)
		if ss.Kind != description.Single {
			t.Errorf("findServer does not properly set the topology description kind. got %v; want %v", ss.Kind, description.Single)
		}
	})
}
