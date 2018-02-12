// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package cluster

import (
	"errors"
	"sync"

	"github.com/mongodb/mongo-go-driver/mongo/model"
	"github.com/mongodb/mongo-go-driver/mongo/private/server"
)

type monitor interface {
	Subscribe() (<-chan *model.Cluster, func(), error)
	RequestImmediateCheck()
}

// StartMonitor begins monitoring a cluster.
func StartMonitor(opts ...Option) (*Monitor, error) {
	cfg, err := newConfig(opts...)
	if err != nil {
		return nil, err
	}

	m := &Monitor{
		cfg:         cfg,
		subscribers: make(map[int64]chan *model.Cluster),
		changes:     make(chan *model.Server),
		current:     &model.Cluster{},
		fsm:         model.NewFSM(),
		servers:     make(map[model.Addr]*server.Monitor),
	}

	if cfg.replicaSetName != "" {
		m.fsm.SetName = cfg.replicaSetName
		m.fsm.Kind = model.ReplicaSetNoPrimary
	}
	if cfg.mode == SingleMode {
		m.fsm.Kind = model.Single
	}

	for _, address := range cfg.seedList {
		addr := model.Addr(address).Canonicalize()
		m.fsm.Servers = append(m.fsm.Servers, &model.Server{Addr: addr})
		m.startMonitoringServer(addr)
	}

	go func() {
		for change := range m.changes {
			// apply the change
			current, err := m.apply(change)
			if err != nil {
				continue
			}

			m.currentLock.Lock()
			m.current = current
			m.currentLock.Unlock()

			// send the change to all subscribers
			m.subscriberLock.Lock()
			for _, ch := range m.subscribers {
				select {
				case <-ch:
					// drain channel if not empty
				default:
					// do nothing if chan already empty
				}
				ch <- current
			}
			m.subscriberLock.Unlock()
		}
		m.subscriberLock.Lock()
		for id, ch := range m.subscribers {
			close(ch)
			delete(m.subscribers, id)
		}
		m.subscriptionsClosed = true
		m.subscriberLock.Unlock()
	}()

	return m, nil
}

// MonitorMode indicates the mode with which to run the monitor.
type MonitorMode uint8

// MonitorMode constants.
const (
	AutomaticMode MonitorMode = iota
	SingleMode
)

// Monitor continuously monitors the cluster for changes
// and reacts accordingly, adding or removing servers as necessary.
type Monitor struct {
	cfg *config

	current     *model.Cluster
	currentLock sync.Mutex

	changes chan *model.Server
	fsm     *model.FSM

	subscribers         map[int64]chan *model.Cluster
	lastSubscriberID    int64
	subscriptionsClosed bool
	subscriberLock      sync.Mutex

	serversLock   sync.Mutex
	serversClosed bool
	servers       map[model.Addr]*server.Monitor
	closingWG     sync.WaitGroup
}

// ServerMonitor gets the server monitor for the specified endpoint. It
// is imperative that this monitor not be stopped.
func (m *Monitor) ServerMonitor(addr model.Addr) (*server.Monitor, bool) {
	m.serversLock.Lock()
	server, ok := m.servers[addr]
	m.serversLock.Unlock()
	return server, ok
}

// Stop turns the monitor off.
func (m *Monitor) Stop() {
	m.serversLock.Lock()
	m.serversClosed = true
	for addr, server := range m.servers {
		m.stopMonitoringServer(addr, server)
	}
	m.serversLock.Unlock()

	m.closingWG.Wait()
	close(m.changes)
}

// Subscribe returns a channel on which all updated ClusterDescs
// will be sent. The channel will have a buffer size of one, and
// will be pre-populated with the current ClusterDesc.
// Subscribe also returns a function that, when called, will close
// the subscription channel and remove it from the list of subscriptions.
func (m *Monitor) Subscribe() (<-chan *model.Cluster, func(), error) {
	// create channel and populate with current state
	ch := make(chan *model.Cluster, 1)
	m.currentLock.Lock()
	ch <- m.current
	m.currentLock.Unlock()

	// add channel to subscribers
	m.subscriberLock.Lock()
	defer m.subscriberLock.Unlock()
	if m.subscriptionsClosed {
		close(ch)
		return nil, nil, errors.New("cannot subscribe to monitor after stopping it")
	}
	m.lastSubscriberID++
	id := m.lastSubscriberID
	m.subscribers[id] = ch

	unsubscribe := func() {
		m.subscriberLock.Lock()
		defer m.subscriberLock.Unlock()
		if !m.subscriptionsClosed {
			close(ch)
			delete(m.subscribers, id)
		}
	}

	return ch, unsubscribe, nil
}

// RequestImmediateCheck will send heartbeats to all the servers in the
// cluster right away, instead of waiting for the heartbeat timeout.
func (m *Monitor) RequestImmediateCheck() {
	m.serversLock.Lock()
	for _, server := range m.servers {
		server.RequestImmediateCheck()
	}
	m.serversLock.Unlock()
}

func (m *Monitor) startMonitoringServer(addr model.Addr) {
	if _, ok := m.servers[addr]; ok {
		return
	}

	monitor, _ := server.StartMonitor(addr, m.cfg.serverOpts...)

	m.servers[addr] = monitor

	ch, _, _ := monitor.Subscribe()

	m.closingWG.Add(1)
	go func() {
		for c := range ch {
			m.changes <- c
		}
		m.closingWG.Done()
	}()
}

func (m *Monitor) stopMonitoringServer(addr model.Addr, server *server.Monitor) {
	server.Stop()
	delete(m.servers, addr)
}

func (m *Monitor) apply(s *model.Server) (*model.Cluster, error) {
	old := m.fsm.Cluster

	err := m.fsm.Apply(s)
	if err != nil {
		return nil, err
	}

	new := m.fsm.Cluster

	diff := model.DiffCluster(&old, &new)
	m.serversLock.Lock()
	if m.serversClosed {
		m.serversLock.Unlock()
		return &model.Cluster{}, nil
	}
	for _, oldServer := range diff.RemovedServers {
		if sm, ok := m.servers[oldServer.Addr]; ok {
			m.stopMonitoringServer(oldServer.Addr, sm)
		}
	}
	for _, newServer := range diff.AddedServers {
		if _, ok := m.servers[newServer.Addr]; !ok {
			m.startMonitoringServer(newServer.Addr)
		}
	}
	m.serversLock.Unlock()
	return &new, nil
}
